# Product Requirements Document
## FinTech Data Pipeline

| Field       | Value                          |
|-------------|--------------------------------|
| Project     | fintech-data-pipeline          |
| Author      | Carlos                         |
| Status      | In Progress                    |
| Version     | 1.0                            |
| Last updated| 2026-05-21                     |

---

## 1. Problem statement

FinTech companies generate large volumes of transaction data daily. Without a reliable pipeline to ingest, validate, transform, and serve this data, business teams cannot answer critical questions: Which spending categories drive the most revenue? Which customer segments are most active? Are transaction failure rates within acceptable thresholds?

This project solves that problem by building a complete analytics pipeline — from raw transaction ingestion to business-ready reporting.

---

## 2. Goals

| Goal | Success metric |
|------|----------------|
| Reliable ingestion | Zero data loss between CSV source and PostgreSQL |
| Data quality | All 6 quality checks pass on every pipeline run |
| Reproducibility | Pipeline produces identical results on repeated runs (idempotent) |
| Observability | Every pipeline run logged with status, row count, and timestamp |
| Transformability | Raw data modelled into business-ready analytics layers via dbt |
| CI/CD | All tests pass automatically on every push to main |

---

## 3. Non-goals

- Real-time streaming (this pipeline is batch-oriented)
- Multi-source ingestion (single CSV source for this version)
- User authentication on the dashboard
- Production deployment (portfolio/demo scope)

---

## 4. Data model

### Source: `transactions.csv`

| Column         | Type    | Description                        |
|----------------|---------|------------------------------------|
| transaction_id | UUID    | Unique transaction identifier      |
| customer_id    | UUID    | Customer identifier                |
| province       | VARCHAR | Canadian province code (BC, ON...) |
| age_group      | VARCHAR | Customer age bracket               |
| account_type   | VARCHAR | chequing, savings, premium         |
| merchant       | VARCHAR | Merchant name                      |
| category       | VARCHAR | Spending category (10 values)      |
| amount         | NUMERIC | Transaction amount in CAD          |
| currency       | VARCHAR | Always CAD                         |
| status         | VARCHAR | completed, pending, failed         |
| txn_date       | DATE    | Transaction date                   |
| txn_month      | VARCHAR | YYYY-MM format for grouping        |
| txn_year       | INTEGER | Transaction year                   |

### PostgreSQL: `raw_transactions`

Mirrors the CSV schema plus:
- `id` SERIAL PRIMARY KEY
- `loaded_at` TIMESTAMPTZ — when the row was ingested

### PostgreSQL: `pipeline_runs`

| Column      | Type    | Description              |
|-------------|---------|--------------------------|
| id          | SERIAL  | Run identifier           |
| run_at      | TIMESTAMPTZ | When the run executed |
| rows_loaded | INTEGER | Rows successfully loaded |
| status      | VARCHAR | success or failed        |
| notes       | TEXT    | Error detail if failed   |

---

## 5. Pipeline stages

### Stage 1 — Extract
Read `transactions.csv` into a pandas DataFrame. Log row count and column names.

### Stage 2 — Transform / Validate
| Step | Action |
|------|--------|
| Null check | Assert zero nulls across all columns |
| Deduplication | Drop rows with duplicate `transaction_id` |
| Type casting | Cast `txn_date` to DATE, `amount` to NUMERIC |
| Amount validation | Filter out zero or negative amounts |
| Status validation | Keep only `completed`, `pending`, `failed` |
| String cleaning | Strip whitespace from all VARCHAR columns |

### Stage 3 — Load
Truncate `raw_transactions` (idempotent), bulk insert all cleaned rows using `execute_values` (batch size 500), log run to `pipeline_runs`.

### Stage 4 — dbt Transforms (Phase 2)
| Model | Layer | Description |
|-------|-------|-------------|
| `stg_transactions` | Staging | Renamed, cast, cleaned source |
| `fct_revenue` | Fact | Completed transactions only |
| `rpt_monthly_summary` | Report | Revenue + volume by month |
| `rpt_category_breakdown` | Report | Revenue + volume by category |
| `rpt_province_analysis` | Report | Geographic revenue distribution |

### Stage 5 — Dashboard (Phase 3)
React + Recharts dashboard consuming the `rpt_` models via a simple API layer. Charts: monthly revenue trend, category breakdown pie, province heatmap, transaction volume over time.

---

## 6. Data quality rules

| Rule | Implementation | Severity |
|------|---------------|----------|
| No nulls | pandas `isnull().sum()` | Block |
| No duplicate IDs | `drop_duplicates(subset=['transaction_id'])` | Block |
| Positive amounts | `df[df['amount'] > 0]` | Block |
| Valid statuses | `isin(['completed','pending','failed'])` | Block |
| Row count | Assert == 5,000 | Warning |
| Revenue floor | Assert > CAD $1,000,000 | Warning |

---

## 7. Observability

Every pipeline run is logged to `pipeline_runs` with:
- Timestamp
- Row count
- Status (success / failed)
- Error detail on failure

Future: structured logging to a monitoring service (Sentry, Datadog).

---

## 8. CI/CD pipeline

```
push to main
     │
     ├── Job 1: validate-data
     │   ├── generate_data.py
     │   └── 6 data quality assertions
     │
     └── Job 2: test-ingestion (needs Job 1)
         ├── spin up PostgreSQL service
         ├── create schema
         ├── run ingest.py
         └── verify DB state (4 assertions)
```

---

## 9. Phases

| Phase | Scope | Status |
|-------|-------|--------|
| 1 — Ingestion | generate → validate → load → CI | ✅ Complete |
| 2 — Transforms | dbt stg → fct → rpt models + tests | 🔄 In progress |
| 3 — Dashboard | React + Recharts analytics UI | ⏳ Planned |
| 4 — Orchestration | Docker Compose full stack | ⏳ Planned |

---

## 10. Open questions

- Should the dashboard be server-rendered or a static SPA?
- Should dbt run inside Docker or locally via CLI?
- Add Airflow for scheduling or keep it simple with cron?
