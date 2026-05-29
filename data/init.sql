-- FinTech Pipeline — Database Schema
-- Runs once on first container start via docker-entrypoint-initdb.d

CREATE TABLE IF NOT EXISTS raw_transactions (
    id             SERIAL PRIMARY KEY,
    transaction_id UUID        NOT NULL UNIQUE,
    customer_id    UUID        NOT NULL,
    province       VARCHAR(5)  NOT NULL,
    age_group      VARCHAR(10) NOT NULL,
    account_type   VARCHAR(20) NOT NULL,
    merchant       VARCHAR(100) NOT NULL,
    category       VARCHAR(50) NOT NULL,
    amount         NUMERIC(12, 2) NOT NULL,
    currency       VARCHAR(5)  NOT NULL DEFAULT 'CAD',
    status         VARCHAR(20) NOT NULL,
    txn_date       DATE        NOT NULL,
    txn_month      VARCHAR(7)  NOT NULL,
    txn_year       INTEGER     NOT NULL,
    ingested_at    TIMESTAMP   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pipeline_runs (
    id          SERIAL PRIMARY KEY,
    run_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    rows_loaded INTEGER   NOT NULL,
    status      VARCHAR(20) NOT NULL,
    notes       TEXT
);

CREATE INDEX IF NOT EXISTS idx_raw_txn_status   ON raw_transactions(status);
CREATE INDEX IF NOT EXISTS idx_raw_txn_category ON raw_transactions(category);
CREATE INDEX IF NOT EXISTS idx_raw_txn_province ON raw_transactions(province);
CREATE INDEX IF NOT EXISTS idx_raw_txn_date     ON raw_transactions(txn_date);
