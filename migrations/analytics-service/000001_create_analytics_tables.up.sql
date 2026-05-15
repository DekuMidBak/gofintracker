CREATE TABLE processed_events (
    event_id UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE monthly_aggregates (
    user_id UUID NOT NULL,
    year INT NOT NULL,
    month INT NOT NULL,
    currency TEXT NOT NULL,
    income_amount BIGINT NOT NULL DEFAULT 0,
    expense_amount BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, year, month, currency),
    CONSTRAINT monthly_aggregates_month_check CHECK (month BETWEEN 1 AND 12),
    CONSTRAINT monthly_aggregates_year_check CHECK (year >= 1970),
    CONSTRAINT monthly_aggregates_currency_check CHECK (currency = upper(currency) AND length(currency) = 3),
    CONSTRAINT monthly_aggregates_income_non_negative_check CHECK (income_amount >= 0),
    CONSTRAINT monthly_aggregates_expense_non_negative_check CHECK (expense_amount >= 0)
);

CREATE TABLE category_aggregates (
    user_id UUID NOT NULL,
    category_id UUID NOT NULL,
    year INT NOT NULL,
    month INT NOT NULL,
    currency TEXT NOT NULL,
    type TEXT NOT NULL,
    amount BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, category_id, year, month, currency, type),
    CONSTRAINT category_aggregates_month_check CHECK (month BETWEEN 1 AND 12),
    CONSTRAINT category_aggregates_year_check CHECK (year >= 1970),
    CONSTRAINT category_aggregates_currency_check CHECK (currency = upper(currency) AND length(currency) = 3),
    CONSTRAINT category_aggregates_type_check CHECK (type IN ('income', 'expense')),
    CONSTRAINT category_aggregates_amount_non_negative_check CHECK (amount >= 0)
);

CREATE INDEX monthly_aggregates_user_period_idx
    ON monthly_aggregates (user_id, year, month);

CREATE INDEX category_aggregates_user_period_idx
    ON category_aggregates (user_id, year, month);
