package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/NicoMartinns/gastano-menos/internal/domain"
	"github.com/lib/pq"
)

type UserRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

const userColumns = `id, name, email, password_hash, role, is_active, session_version, created_at`

type scanner interface{ Scan(...interface{}) error }

func scanUser(row scanner) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.SessionVersion, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func userWriteError(err error) error {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.ErrEmailInUse
	}
	return err
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE email = $1`, email))
}

func (r *UserRepository) FindByID(id string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(`SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (r *UserRepository) List(limit, offset int) ([]domain.User, int, error) {
	// One snapshot keeps the count and page consistent during account creation.
	tx, err := r.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()
	var total int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := tx.Query(`SELECT `+userColumns+` FROM users ORDER BY created_at DESC, id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := make([]domain.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func insertUser(tx *sql.Tx, u *domain.User) error {
	err := tx.QueryRow(`INSERT INTO users (name, email, password_hash, role, is_active)
        VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		u.Name, u.Email, u.PasswordHash, u.Role, u.IsActive).Scan(&u.ID, &u.CreatedAt)
	return userWriteError(err)
}

// All access changes share this transaction lock, including bootstrap.
// This serializes the last-admin check even across multiple API instances.
func lockUserAccess(tx *sql.Tx) error {
	_, err := tx.Exec(`SELECT pg_advisory_xact_lock(735014001)`)
	return err
}

func (r *UserRepository) Create(u *domain.User) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := lockUserAccess(tx); err != nil {
		return err
	}
	if err := insertUser(tx, u); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *UserRepository) CreateInitialAdmin(u *domain.User) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := lockUserAccess(tx); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM users WHERE role = 'ADMIN')`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return domain.ErrAdminExists
	}
	if err := insertUser(tx, u); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *UserRepository) Update(u *domain.User) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := lockUserAccess(tx); err != nil {
		return err
	}
	existing, err := scanUser(tx.QueryRow(`SELECT `+userColumns+` FROM users WHERE id = $1 FOR UPDATE`, u.ID))
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrUserNotFound
	}
	if existing.Role == domain.RoleAdmin && existing.IsActive && (u.Role != domain.RoleAdmin || !u.IsActive) {
		var count int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'ADMIN' AND is_active`).Scan(&count); err != nil {
			return err
		}
		if count <= 1 {
			return domain.ErrLastAdmin
		}
	}
	// Explicit casts keep PostgreSQL's parameter inference consistent between
	// assignment to varchar(10) and the comparisons used for session revocation.
	err = tx.QueryRow(`UPDATE users SET name = $1, email = $2, role = $3::varchar(10), is_active = $4::boolean,
        session_version = session_version + CASE WHEN role <> $3::varchar(10) OR is_active <> $4::boolean THEN 1 ELSE 0 END
        WHERE id = $5 RETURNING created_at, session_version`,
		u.Name, u.Email, u.Role, u.IsActive, u.ID).Scan(&u.CreatedAt, &u.SessionVersion)
	if err != nil {
		return userWriteError(err)
	}
	return tx.Commit()
}
