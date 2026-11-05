package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
)

// Scripted SQL driver: tests query scope and access guards without a real DB,
// credentials, migrations, or writes to persistent data.
type sqlStep struct {
	contains  []string
	forbidden []string
	args      []driver.Value
	rows      [][]driver.Value
	err       error
}
type scriptedDB struct {
	t         *testing.T
	steps     []sqlStep
	committed bool
}
type scriptedConn struct{ db *scriptedDB }
type scriptedTx struct{ db *scriptedDB }
type scriptedRows struct {
	rows    [][]driver.Value
	columns []string
}
type noDriver struct{}

func (noDriver) Open(string) (driver.Conn, error)                  { return nil, errors.New("use connector") }
func (d *scriptedDB) Connect(context.Context) (driver.Conn, error) { return &scriptedConn{d}, nil }
func (d *scriptedDB) Driver() driver.Driver                        { return noDriver{} }
func (c *scriptedConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *scriptedConn) Close() error              { return nil }
func (c *scriptedConn) Begin() (driver.Tx, error) { return &scriptedTx{c.db}, nil }
func (c *scriptedConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return c.Begin()
}
func (tx *scriptedTx) Commit() error   { tx.db.committed = true; return nil }
func (tx *scriptedTx) Rollback() error { return nil }
func (d *scriptedDB) take(query string, args []driver.NamedValue) sqlStep {
	d.t.Helper()
	if len(d.steps) == 0 {
		d.t.Fatalf("unexpected query: %s", query)
	}
	step := d.steps[0]
	d.steps = d.steps[1:]
	normalized := strings.Join(strings.Fields(query), " ")
	for _, text := range step.contains {
		if !strings.Contains(normalized, text) {
			d.t.Fatalf("query missing %q: %s", text, normalized)
		}
	}
	for _, text := range step.forbidden {
		if strings.Contains(normalized, text) {
			d.t.Fatalf("forbidden scope %q: %s", text, normalized)
		}
	}
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	if len(values) != len(step.args) || (len(values) > 0 && !reflect.DeepEqual(values, step.args)) {
		d.t.Fatalf("query args = %#v, want %#v", values, step.args)
	}
	return step
}
func (c *scriptedConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	step := c.db.take(q, args)
	if step.err != nil {
		return nil, step.err
	}
	cols := []string{}
	if len(step.rows) > 0 {
		for i := range step.rows[0] {
			cols = append(cols, fmt.Sprint(i))
		}
	}
	return &scriptedRows{rows: step.rows, columns: cols}, nil
}
func (c *scriptedConn) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	step := c.db.take(q, args)
	if step.err != nil {
		return nil, step.err
	}
	return driver.RowsAffected(1), nil
}
func (r *scriptedRows) Columns() []string { return r.columns }
func (r *scriptedRows) Close() error      { return nil }
func (r *scriptedRows) Next(values []driver.Value) error {
	if len(r.rows) == 0 {
		return io.EOF
	}
	copy(values, r.rows[0])
	r.rows = r.rows[1:]
	return nil
}
func testDB(t *testing.T, steps ...sqlStep) (*sql.DB, *scriptedDB) {
	t.Helper()
	script := &scriptedDB{t: t, steps: steps}
	db := sql.OpenDB(script)
	t.Cleanup(func() {
		db.Close()
		if len(script.steps) > 0 {
			t.Errorf("%d expected queries not executed", len(script.steps))
		}
	})
	return db, script
}

func TestFinancialQueriesAreShared(t *testing.T) {
	date := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		step sqlStep
		run  func(*sql.DB) error
	}{
		{"categories list", sqlStep{contains: []string{"FROM categories"}, forbidden: []string{"WHERE user_id"}}, func(db *sql.DB) error { _, err := NewCategoryRepository(db).FindAll(); return err }},
		{"categories delete", sqlStep{contains: []string{"DELETE FROM categories WHERE id = $1"}, forbidden: []string{"user_id ="}, args: []driver.Value{"category"}}, func(db *sql.DB) error { return NewCategoryRepository(db).Delete("category") }},
		{"transactions list", sqlStep{contains: []string{"EXTRACT(YEAR FROM t.date) = $1", "EXTRACT(MONTH FROM t.date) = $2"}, forbidden: []string{"t.user_id ="}, args: []driver.Value{int64(2026), int64(11)}}, func(db *sql.DB) error { _, err := NewTransactionRepository(db).FindByMonth(2026, 11); return err }},
		{"transactions summary", sqlStep{contains: []string{"EXTRACT(YEAR FROM t.date)::int = $1"}, forbidden: []string{"t.user_id ="}, args: []driver.Value{int64(2026)}}, func(db *sql.DB) error { _, err := NewTransactionRepository(db).GetMonthlySummary(2026); return err }},
		{"transactions delete", sqlStep{contains: []string{"WHERE id = $1"}, forbidden: []string{"user_id ="}, args: []driver.Value{"transaction"}}, func(db *sql.DB) error { return NewTransactionRepository(db).Delete("transaction") }},
		{"transactions update", sqlStep{contains: []string{"WHERE id = $7"}, forbidden: []string{"user_id ="}, args: []driver.Value{"category", "shared", float64(10), "EXPENSE", date, nil, "transaction"}}, func(db *sql.DB) error {
			return NewTransactionRepository(db).Update(&domain.Transaction{ID: "transaction", UserID: "other-creator", CategoryID: "category", Description: "shared", Amount: 10, Type: domain.Expense, Date: date})
		}},
		{"future delete", sqlStep{contains: []string{"WHERE date >= $1", "id = $2 OR recurring_origin_id = $3"}, forbidden: []string{"user_id ="}, args: []driver.Value{date, "transaction", "origin"}}, func(db *sql.DB) error {
			return NewTransactionRepository(db).DeleteFuture("transaction", date, "origin")
		}},
		{"future update", sqlStep{contains: []string{"WHERE date >= $6", "id = $7 OR recurring_origin_id = $8"}, forbidden: []string{"user_id ="}, args: []driver.Value{"category", "shared", float64(10), "EXPENSE", nil, date, "transaction", "origin"}}, func(db *sql.DB) error {
			return NewTransactionRepository(db).UpdateFuture("origin", date, &domain.Transaction{ID: "transaction", CategoryID: "category", Description: "shared", Amount: 10, Type: domain.Expense})
		}},
		{"find other creator transaction", sqlStep{contains: []string{"WHERE id = $1"}, forbidden: []string{"user_id ="}, args: []driver.Value{"transaction"}, rows: [][]driver.Value{{"transaction", "other-creator", "category", "shared", float64(10), "EXPENSE", date, false, nil, nil, nil, date}}}, func(db *sql.DB) error { _, err := NewTransactionRepository(db).FindByID("transaction"); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, _ := testDB(t, tc.step)
			if err := tc.run(db); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func storedUser(role string, active bool) [][]driver.Value {
	return [][]driver.Value{{"target", "Admin", "admin@example.com", "hash", role, active, int64(0), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}}
}
func accessLockStep() sqlStep { return sqlStep{contains: []string{"pg_advisory_xact_lock(735014001)"}} }

func TestLastAdminCannotBeBlockedOrDemoted(t *testing.T) {
	for _, u := range []*domain.User{
		{ID: "target", Name: "Admin", Email: "admin@example.com", Role: domain.RoleUser, IsActive: true},
		{ID: "target", Name: "Admin", Email: "admin@example.com", Role: domain.RoleAdmin, IsActive: false},
	} {
		db, script := testDB(t, accessLockStep(), sqlStep{contains: []string{"WHERE id = $1 FOR UPDATE"}, args: []driver.Value{"target"}, rows: storedUser("ADMIN", true)}, sqlStep{contains: []string{"COUNT(*)", "role = 'ADMIN' AND is_active"}, rows: [][]driver.Value{{int64(1)}}})
		if err := NewUserRepository(db).Update(u); !errors.Is(err, domain.ErrLastAdmin) {
			t.Fatalf("last admin change accepted: %v", err)
		}
		if script.committed {
			t.Fatal("invalid admin update committed")
		}
	}
}

func TestAccessUpdateRevokesTokensAndCommits(t *testing.T) {
	date := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	db, script := testDB(t, accessLockStep(), sqlStep{args: []driver.Value{"target"}, rows: storedUser("USER", true)}, sqlStep{
		contains: []string{"role = $3::varchar(10)", "is_active = $4::boolean", "session_version = session_version + CASE WHEN role <> $3::varchar(10) OR is_active <> $4::boolean THEN 1 ELSE 0 END"},
		args:     []driver.Value{"User", "user@example.com", "USER", false, "target"}, rows: [][]driver.Value{{date, int64(1)}},
	})
	user := &domain.User{ID: "target", Name: "User", Email: "user@example.com", Role: domain.RoleUser, IsActive: false}
	if err := NewUserRepository(db).Update(user); err != nil {
		t.Fatal(err)
	}
	if !script.committed || user.SessionVersion != 1 {
		t.Fatal("access change was not committed with a new session version")
	}
}

func TestBootstrapRefusesExistingAdministrator(t *testing.T) {
	db, script := testDB(t, accessLockStep(), sqlStep{contains: []string{"SELECT EXISTS", "role = 'ADMIN'"}, rows: [][]driver.Value{{true}}})
	if err := NewUserRepository(db).CreateInitialAdmin(&domain.User{}); !errors.Is(err, domain.ErrAdminExists) {
		t.Fatalf("bootstrap overwrote admin: %v", err)
	}
	if script.committed {
		t.Fatal("duplicate bootstrap committed")
	}
}
