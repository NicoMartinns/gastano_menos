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