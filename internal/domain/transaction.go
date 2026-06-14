package domain

import "time"

type TransactionType string

const (
    Income  TransactionType = "INCOME"
    Expense TransactionType = "EXPENSE"
)

type Transaction struct {
    ID                string
    UserID            string
    CategoryID        string
    CategoryName      string
    Description       string
    Amount            float64
    Type              TransactionType
    Date              time.Time
    IsRecurring       bool
    RecurringMonths   *int       // ponteiro porque pode ser NULL
    RecurrenceDay     *int       // ponteiro porque pode ser NULL
    RecurringOriginID *string    // ponteiro porque pode ser NULL
    CreatedAt         time.Time
}