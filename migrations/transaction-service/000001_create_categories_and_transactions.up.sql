CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT categories_type_check CHECK (type IN ('income', 'expense')),
    CONSTRAINT categories_name_not_empty_check CHECK (length(trim(name)) > 0),
    CONSTRAINT categories_user_id_id_unique UNIQUE (user_id, id),
    CONSTRAINT categories_user_id_id_type_unique UNIQUE (user_id, id, type)
);

CREATE UNIQUE INDEX categories_user_name_type_unique_idx
    ON categories (user_id, lower(name), type);

CREATE INDEX categories_user_id_idx
    ON categories (user_id);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    category_id UUID NOT NULL,
    type TEXT NOT NULL,
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transactions_type_check CHECK (type IN ('income', 'expense')),
    CONSTRAINT transactions_amount_positive_check CHECK (amount > 0),
    CONSTRAINT transactions_currency_check CHECK (currency = upper(currency) AND length(currency) = 3),
    CONSTRAINT transactions_category_fk
        FOREIGN KEY (user_id, category_id, type)
        REFERENCES categories (user_id, id, type)
);

CREATE INDEX transactions_user_occurred_at_idx
    ON transactions (user_id, occurred_at DESC);

CREATE INDEX transactions_user_category_idx
    ON transactions (user_id, category_id);

CREATE INDEX transactions_user_type_idx
    ON transactions (user_id, type);
