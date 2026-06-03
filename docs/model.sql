users
├── id (UUID)
├── name
├── email
├── password_hash
└── created_at

categories
├── id (UUID)
├── user_id (FK)
├── name
├── type (INCOME | EXPENSE)
└── created_at

transactions
├── id (UUID)
├── user_id (FK)
├── category_id (FK)
├── description
├── amount (NUMERIC 10,2)
├── type (INCOME | EXPENSE)
├── date (DATE)
├── is_recurring (BOOLEAN)
├── recurring_months (INT, nullable)  → NULL = sem prazo (Netflix)
│                                      → 12 = 12 parcelas
├── recurrence_day (INT)              → dia do mês que repete
├── recurring_origin_id (UUID, FK nullable) → aponta pra transação mãe
└── created_at