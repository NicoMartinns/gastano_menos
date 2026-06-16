package repository

import (
    "database/sql"
    "fmt"
    "time"

    "github.com/NicoMartinns/gastano-menos/internal/domain"
)

type TransactionRepository struct {
    db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
    return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(t *domain.Transaction) error {
	query := `
		INSERT INTO transactions 
			(id, user_id, category_id, description, amount, type, date, 
			 is_recurring, recurring_months, recurrence_day, recurring_origin_id,
			 payment_method, created_at)
		VALUES 
			(gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`

	return r.db.QueryRow(
		query,
		t.UserID, t.CategoryID, t.Description, t.Amount, t.Type,
		t.Date, t.IsRecurring, t.RecurringMonths, t.RecurrenceDay,
		t.RecurringOriginID, t.PaymentMethod, time.Now(),
	).Scan(&t.ID, &t.CreatedAt)
}

func (r *TransactionRepository) FindByMonth(userID string, year int, month int) ([]domain.Transaction, error) {
	query := `
		SELECT t.id, t.user_id, t.category_id, t.description, t.amount, t.type, t.date,
			t.is_recurring, t.recurring_months, t.recurrence_day, t.recurring_origin_id,
			t.created_at, t.payment_method, c.name as category_name
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.user_id = $1
		  AND EXTRACT(YEAR FROM t.date) = $2
		  AND EXTRACT(MONTH FROM t.date) = $3
		ORDER BY t.date ASC
	`

	rows, err := r.db.Query(query, userID, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		err := rows.Scan(
			&t.ID, &t.UserID, &t.CategoryID, &t.Description,
			&t.Amount, &t.Type, &t.Date, &t.IsRecurring,
			&t.RecurringMonths, &t.RecurrenceDay, &t.RecurringOriginID,
			&t.CreatedAt, &t.PaymentMethod, &t.CategoryName,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	return transactions, nil
}

func (r *TransactionRepository) Delete(id string, userID string) error {
	query := `DELETE FROM transactions WHERE id = $1 AND user_id = $2`
	result, err := r.db.Exec(query, id, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("transação não encontrada")
	}

	return nil
}

func (r *TransactionRepository) Update(t *domain.Transaction) error {
	query := `
		UPDATE transactions
		SET category_id     = $1,
		    description     = $2,
		    amount          = $3,
		    type            = $4,
		    date            = $5,
		    payment_method  = $6
		WHERE id = $7 AND user_id = $8
	`
	result, err := r.db.Exec(query,
		t.CategoryID, t.Description, t.Amount,
		t.Type, t.Date, t.PaymentMethod, t.ID, t.UserID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("transação não encontrada")
	}

	return nil
}

func (r *TransactionRepository) GetMonthlySummary(userID string, year int) ([]domain.MonthlySummary, error) {
	query := `
		WITH months AS (
			SELECT generate_series(1, 12) AS month
		)
		SELECT 
			m.month,
			COALESCE(SUM(CASE WHEN t.type = 'INCOME'  THEN t.amount ELSE 0 END), 0) AS income,
			COALESCE(SUM(CASE WHEN t.type = 'EXPENSE' THEN t.amount ELSE 0 END), 0) AS expense
		FROM months m
		LEFT JOIN transactions t
			ON EXTRACT(MONTH FROM t.date)::int = m.month
			AND EXTRACT(YEAR FROM t.date)::int = $2
			AND t.user_id = $1
		GROUP BY m.month
		ORDER BY m.month ASC
	`

	rows, err := r.db.Query(query, userID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summary []domain.MonthlySummary
	for rows.Next() {
		var s domain.MonthlySummary
		s.Year = year
		if err := rows.Scan(&s.Month, &s.Income, &s.Expense); err != nil {
			return nil, err
		}
		summary = append(summary, s)
	}

	return summary, nil
}

func (r *TransactionRepository) DeleteFuture(id string, userID string, date time.Time, originID string) error {
	query := `
		DELETE FROM transactions
		WHERE user_id = $1
		  AND date >= $2
		  AND (id = $3 OR (recurring_origin_id = $4 AND date >= $2))
	`
	_, err := r.db.Exec(query, userID, date, id, originID)
	return err
}

func (r *TransactionRepository) UpdateFuture(originID string, userID string, date time.Time, t *domain.Transaction) error {
	query := `
		UPDATE transactions
		SET category_id    = $1,
		    description    = $2,
		    amount         = $3,
		    type           = $4,
		    payment_method = $5
		WHERE user_id = $6
		  AND date >= $7
		  AND (id = $8 OR recurring_origin_id = $9)
	`
	_, err := r.db.Exec(query,
		t.CategoryID, t.Description, t.Amount, t.Type, t.PaymentMethod,
		userID, date, t.ID, originID,
	)
	return err
}

func (r *TransactionRepository) FindByID(id string, userID string) (*domain.Transaction, error) {
	query := `
		SELECT id, user_id, category_id, description, amount, type, date,
		       is_recurring, recurring_months, recurrence_day, recurring_origin_id, created_at
		FROM transactions
		WHERE id = $1 AND user_id = $2
	`
	t := &domain.Transaction{}
	err := r.db.QueryRow(query, id, userID).Scan(
		&t.ID, &t.UserID, &t.CategoryID, &t.Description,
		&t.Amount, &t.Type, &t.Date, &t.IsRecurring,
		&t.RecurringMonths, &t.RecurrenceDay, &t.RecurringOriginID,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}