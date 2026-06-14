package domain

import "time"

type CategoryType string

const (
    CategoryIncome  CategoryType = "INCOME"
    CategoryExpense CategoryType = "EXPENSE"
)

type Category struct {
    ID        string
    UserID    string
    Name      string
    Type      CategoryType
    ParentID  *string
    CreatedAt time.Time
}