package repository

import (
    "database/sql"
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
             is_recurring, recurring_months, recurrence_day, recurring_origin_id, created_at)
        VALUES 
            (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        RETURNING id, created_at
    `

    return r.db.QueryRow(
        query,
        t.UserID, t.CategoryID, t.Description, t.Amount, t.Type,
        t.Date, t.IsRecurring, t.RecurringMonths, t.RecurrenceDay,
        t.RecurringOriginID, time.Now(),
    ).Scan(&t.ID, &t.CreatedAt)
}

func (r *TransactionRepository) FindByMonth(userID string, year int, month int) ([]domain.Transaction, error) {
	query := `
		SELECT id, user_id, category_id, description, amount, type, date,
		       is_recurring, recurring_months, recurrence_day, recurring_origin_id, created_at
		FROM transactions
		WHERE user_id = $1
		  AND EXTRACT(YEAR FROM date) = $2
		  AND EXTRACT(MONTH FROM date) = $3
		ORDER BY date ASC
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
			&t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}

	return transactions, nil
}