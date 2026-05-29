# Architecture
## FinTech Data Pipeline

---

## System overview

A batch data pipeline with five distinct layers. Each layer has a single responsibility and clean interfaces to the next — data flows in one direction, transformations are versioned, and every run is observable.

```
┌─────────────────────────────────────────────────────────────────┐
│                        DATA SOURCES                             │
│                                                                 │
│   transactions.csv  (5,000 rows · 13 columns · CAD amounts)    │
└──────────────────────────┬──────────────────────────────────────┘
                           │  pandas read_csv()
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                     INGESTION LAYER                             │
│                                                                 │
│   ingestion/ingest.py                                           │
│   ├── Extract    → read CSV into DataFrame                      │
│   ├── Transform  → validate, deduplicate, cast, clean           │
│   └── Load       → bulk insert to PostgreSQL (batch 500)        │
└──────────────────────────┬──────────────────────────────────────┘
                           │  psycopg2 execute_values()
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                      STORAGE LAYER                              │
│                                                                 │
│   PostgreSQL 15 (Docker)                                        │
│   ├── raw_transactions   → source of truth, append-friendly     │
│   └── pipeline_runs      → observability log                    │
│                                                                 │
│   Indexes: txn_date · category · status                         │
└──────────────────────────┬──────────────────────────────────────┘
                           │  dbt ref()
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                   TRANSFORMATION LAYER                          │
│                                                                 │
│   dbt project (transforms/)                                     │
│                                                                 │
│   Staging (views)                                               │
│   └── stg_transactions                                          │
│       rename · cast · filter nulls · standardise               │
│                                                                 │
│   Fact (table)                                                  │
│   └── fct_revenue                                               │
│       completed transactions only · enriched fields             │
│                                                                 │
│   Report (tables)                                               │
│   ├── rpt_monthly_summary     → revenue + volume by month       │
│   ├── rpt_category_breakdown  → revenue + volume by category    │
│   └── rpt_province_analysis   → geographic revenue split        │
│                                                                 │
│   Tests: not_null · unique · accepted_values · relationships    │
└──────────────────────────┬──────────────────────────────────────┘
                           │  SQL queries via API
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER                           │
│                                                                 │
│   React + Recharts dashboard                                    │
│   ├── Monthly revenue trend (LineChart)                         │
│   ├── Category breakdown (PieChart)                             │
│   ├── Transaction volume (BarChart)                             │
│   └── Province distribution (BarChart)                         │
└─────────────────────────────────────────────────────────────────┘
```

---

## Data flow detail

### 1. Ingestion

```
transactions.csv
      │
      │  pd.read_csv()
      ▼
  DataFrame (5,000 rows)
      │
      ├── assert no nulls
      ├── drop_duplicates(transaction_id)
      ├── cast txn_date → DATE
      ├── cast amount → NUMERIC
      ├── filter amount > 0
      └── filter status in [completed, pending, failed]
      │
      │  execute_values() · batch 500
      ▼
  raw_transactions (PostgreSQL)
      │
      └── pipeline_runs ← log run status + row count
```

### 2. dbt transformation layers

```
raw_transactions
      │
      ▼
stg_transactions (view)
  SELECT
    transaction_id::uuid,
    customer_id::uuid,
    TRIM(province)          AS province,
    TRIM(category)          AS category,
    amount::numeric(12,2)   AS amount,
    status,
    txn_date::date,
    txn_month,
    txn_year
  FROM raw_transactions
  WHERE amount > 0
    AND status IS NOT NULL

      │
      ▼
fct_revenue (table)
  SELECT * FROM stg_transactions
  WHERE status = 'completed'

      │
      ├──────────────────────────────────┐
      ▼                                  ▼
rpt_monthly_summary              rpt_category_breakdown
  SELECT                           SELECT
    txn_month,                       category,
    SUM(amount) AS revenue,          SUM(amount)   AS revenue,
    COUNT(*)    AS volume            COUNT(*)      AS volume,
  FROM fct_revenue                   AVG(amount)   AS avg_amount
  GROUP BY txn_month               FROM fct_revenue
  ORDER BY txn_month               GROUP BY category
```

---

## Key design decisions

### Idempotent ingestion
The ingestion script truncates `raw_transactions` before every load. Running it 10 times produces the same result as running it once. No duplicate data, no stale rows.

### Staging layer as a contract
`stg_transactions` is the contract between raw data and business logic. All downstream models reference `stg_transactions`, never `raw_transactions` directly. If the source schema changes, only the staging model needs updating.

### dbt tests as data contracts
Every model has schema tests. `not_null` and `unique` on `transaction_id`, `accepted_values` on `status` and `category`. Tests run in CI on every push — broken data breaks the build before it reaches the dashboard.

### Indexes for query performance
Three indexes on `raw_transactions`: `txn_date` (range queries), `category` (group by), `status` (filter). The `rpt_` tables are materialized as tables (not views) so dashboard queries are instant.

---

## Docker Compose architecture

```
docker-compose.yml
├── db (postgres:15-alpine)
│   ├── port: 5432
│   ├── volume: pgdata (persistent)
│   └── healthcheck: pg_isready
│
├── pipeline (python:3.12-slim)
│   ├── depends_on: db (healthy)
│   ├── runs: generate_data.py + ingest.py
│   └── exits after completion
│
└── dashboard (node:20-slim)     ← Phase 3
    ├── depends_on: db (healthy)
    └── port: 3001
```

---

## CI/CD flow

```
git push → GitHub Actions

Job 1: validate-data
├── python generate_data.py
└── 6 assertions on CSV output
      │
      │ (needs: validate-data)
      ▼
Job 2: test-ingestion
├── postgres:15-alpine service
├── create schema
├── python ingest.py
└── 4 assertions on DB state
```

---

## Future improvements

| Improvement | Why |
|-------------|-----|
| Apache Airflow DAG | Replace manual script execution with scheduled, observable DAG |
| Incremental dbt models | Only process new rows, not full refresh each run |
| Great Expectations | Richer data quality framework beyond pandas assertions |
| Prometheus + Grafana | Pipeline metrics (run duration, row counts, failure rate) |
| Multi-source ingestion | Add API source alongside CSV |
