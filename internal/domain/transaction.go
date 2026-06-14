package domain

import "time"

type TransactionType string

const (
    Income  TransactionType = "INCOME"
    Expense TransactionType = "EXPENSE"
)

type PaymentMethod string

const (
    PaymentCredit PaymentMethod = "credit"
    PaymentDebit  PaymentMethod = "debit"
    PaymentPix    PaymentMethod = "pix"
    PaymentBoleto PaymentMethod = "boleto"
    PaymentCash   PaymentMethod = "cash"
)

type Transaction struct {
    ID                string
    UserID            string
    CategoryID        string
    CategoryName      string
    Description       string
    Amount            float64
    Type              TransactionType
    PaymentMethod     *PaymentMethod
    Date              time.Time
    IsRecurring       bool
    RecurringMonths   *int
    RecurrenceDay     *int
    RecurringOriginID *string
    CreatedAt         time.Time
}

