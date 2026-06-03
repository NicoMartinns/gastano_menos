CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    category_id UUID NOT NULL,
    description VARCHAR(255) NOT NULL,
    amount NUMERIC(10,2) NOT NULL,
    type VARCHAR(20) NOT NULL,
    date DATE NOT NULL,
    is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
    recurring_months INT NULL,
    recurrence_day INT NULL,
    recurring_origin_id UUID NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_transactions_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_transactions_category
        FOREIGN KEY (category_id)
        REFERENCES categories(id)
        ON DELETE RESTRICT,

    CONSTRAINT fk_transactions_origin
        FOREIGN KEY (recurring_origin_id)
        REFERENCES transactions(id)
        ON DELETE SET NULL,

    CONSTRAINT chk_transactions_type
        CHECK (type IN ('INCOME', 'EXPENSE')),

    CONSTRAINT chk_recurrence_day
        CHECK (recurrence_day IS NULL OR recurrence_day BETWEEN 1 AND 31),

    CONSTRAINT chk_recurring_fields
        CHECK (
            is_recurring = FALSE
            OR (is_recurring = TRUE AND recurrence_day IS NOT NULL)
        )
);