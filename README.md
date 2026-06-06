# FinTech SaaS Pipeline

[![CI/CD](https://github.com/charlesnet76/fintech-saas-pipeline/actions/workflows/azure-deploy.yml/badge.svg)](https://github.com/charlesnet76/fintech-saas-pipeline/actions)
[![Azure](https://img.shields.io/badge/Azure-5_Services_Live-0078d4?style=flat-square&logo=microsoftazure)](https://auth-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health)
[![dbt](https://img.shields.io/badge/dbt-16_tests_passing-FF694B?style=flat-square&logo=dbt)](https://github.com/charlesnet76/fintech-saas-pipeline)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)

Production-grade FinTech data platform demonstrating end-to-end engineering — from CSV ingestion to ML feature store to live Azure deployment. Built as a portfolio project to showcase full-stack data engineering capability.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     GitHub Actions CI/CD                         │
│         push to main → build → push ACR → deploy Azure          │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│              Azure Container Apps (canadacentral)                │
│                                                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────────┐    │
│  │ auth-service│  │  ai-service │  │   pipeline-service   │    │
│  │  :8081      │  │    :8085    │  │        :8086         │    │
│  │ JWT · RBAC  │  │ Claude API  │  │  CSV upload · ETL    │    │
│  └─────────────┘  └─────────────┘  └──────────────────────┘    │
│                                                                   │
│  ┌──────────────────────┐  ┌──────────────────────────────┐     │
│  │  features-service    │  │      monitor-service         │     │
│  │       :8087          │  │           :8088              │     │
│  │  ML feature store    │  │  Prometheus · SLA alerts     │     │
│  └──────────────────────┘  └──────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                    Data Layer                                     │
│                                                                   │
│   PostgreSQL 15          dbt 1.10           Python ETL           │
│   ├── transactions       ├── stg_transactions  ├── generate_data │
│   ├── organizations      ├── fct_monthly_revenue └── ingest.py   │
│   ├── feature_store      ├── fct_revenue_by_category             │
│   └── audit_logs         ├── fct_revenue_by_province             │
│   RLS multi-tenant       ├── fct_status_summary                  │
│                          └── 16/16 tests passing                  │
└─────────────────────────────────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│               Local Kubernetes (Docker Desktop)                   │
│   Prometheus + Grafana · auth /metrics scraping · PromQL        │
└─────────────────────────────────────────────────────────────────┘
```

---


## Dashboard Preview

![FinTech Analytics Dashboard](docs/dashboard.png)

## Live Services

| Service | URL | Health |
|---------|-----|--------|
| 🔐 **Auth** | [auth-service.canadacentral.azurecontainerapps.io](https://auth-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health) | `/health` |
| 🤖 **AI** | [ai-service.canadacentral.azurecontainerapps.io](https://ai-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health) | `/health` |
| 📊 **Pipeline** | [pipeline-service.canadacentral.azurecontainerapps.io](https://pipeline-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health) | `/health` |
| 🧠 **Features** | [features-service.canadacentral.azurecontainerapps.io](https://features-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health) | `/health` |
| 📡 **Monitor** | [monitor-service.canadacentral.azurecontainerapps.io](https://monitor-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health) | `/health` |

---

## Services

### 🔐 Auth Service (`saas/services/auth/`)
JWT authentication with refresh token rotation and RBAC.

```
POST /auth/register    → create account
POST /auth/login       → returns access token (15min) + refresh token
POST /auth/refresh     → rotate refresh token
POST /auth/logout      → invalidate refresh token
GET  /health           → service health
GET  /metrics          → Prometheus metrics
```

**Stack:** Go 1.26 · bcrypt · JWT · PostgreSQL · Prometheus

### 🤖 AI Service (`saas/services/ai/`)
Financial insights via Claude API. Analyzes transaction data and returns structured insights.

```
POST /insights/revenue    → monthly revenue analysis
POST /insights/anomaly    → transaction anomaly detection
POST /insights/forecast   → spend forecasting
GET  /health
```

**Stack:** Go · Anthropic Claude API · PostgreSQL

### 📊 Pipeline Service (`saas/services/pipeline/`)
CSV upload and ingestion endpoint. Accepts transaction data, validates, and loads into PostgreSQL.

```
POST /upload/csv       → upload CSV file (multipart/form-data)
                         Header: X-Org-ID: <org_id>
                         Returns: {ok, rows_received, rows_inserted, rows_skipped}
GET  /upload/status    → ingestion stats per org
GET  /health
GET  /metrics          → Prometheus upload/row counters
```

**Stack:** Go · PostgreSQL · Prometheus

### 🧠 Features Service (`saas/services/features/`)
ML feature store with point-in-time retrieval and RLS tenant isolation.

```
POST /features/register   → register a new feature definition
GET  /features/list       → list all features for org
POST /features/serve      → get latest feature values
POST /features/serve/pit  → point-in-time feature retrieval
POST /features/ingest     → ingest new feature values
GET  /health
GET  /metrics
```

**Stack:** Go · PostgreSQL · RLS · Prometheus

### 📡 Monitor Service (`saas/services/monitor/`)
Feature freshness monitor. Checks all features × orgs every 5min for SLA violations.

```
GET  /health    → service health
GET  /status    → freshness status per org/feature
GET  /metrics   → stale_features_total, age_seconds, sla_violations_total
```

**Stack:** Go · Prometheus · Slack webhook alerts

---

## Data Pipeline

### dbt Models
```
models/
├── staging/
│   └── stg_transactions.sql          -- clean + validate raw transactions
└── marts/
    ├── fct_monthly_revenue.sql        -- revenue aggregated by month + org
    ├── fct_revenue_by_category.sql   -- breakdown by merchant category
    ├── fct_revenue_by_province.sql   -- geographic breakdown
    └── fct_status_summary.sql        -- completed/pending/failed counts
```

**Tests:** 16/16 passing — `not_null`, `unique`, `accepted_values`, `relationships`

### Sample Data
`data-pipeline/ingestion/generate_data.py` generates 5,000 realistic Canadian FinTech transactions:
- Provinces: BC, ON, AB, QC, MB, SK, NS
- Categories: groceries, restaurants, gas, entertainment, healthcare, travel
- Age groups: 18-24, 25-34, 35-44, 45-54, 55+
- Account types: chequing, savings, credit

---

## Infrastructure

### Azure (Production)
```
Resource Group:  fintech-saas-rg
Region:          canadacentral
Registry:        carlossaasacr.azurecr.io
Environment:     fintech-saas-env
  defaultDomain: gentlebay-f6693cbb.canadacentral.azurecontainerapps.io
Log Analytics:   fintech-saas-logs
```

### GitHub Actions CI/CD (`.github/workflows/azure-deploy.yml`)
```
on: push to main
├── Debug file structure    (verify Dockerfiles present)
├── Build auth service      → push to ACR
├── Build AI service        → push to ACR
├── Build pipeline service  → push to ACR
├── Build features service  → push to ACR
├── Build monitor service   → push to ACR
└── Deploy to Azure         → az containerapp up (all 5)
```

### Local Kubernetes
```
Namespace: fintech-saas
  ├── auth-service   (1/1 Running)
  ├── ai-service     (1/1 Running)
  └── postgres       (1/1 Running)

Namespace: monitoring
  ├── prometheus     (kube-prometheus-stack)
  └── grafana        (admin/admin123 → localhost:3001)
```

---

## Local Development

### Prerequisites
- Docker Desktop (with Kubernetes enabled)
- Go 1.26+
- Python 3.12+
- Node.js 20+
- dbt 1.10+

### Start local stack
```bash
# 1. Start Docker Desktop

# 2. Start PostgreSQL
docker start fintech-db

# 3. Run dbt models
cd data-pipeline
dbt run
dbt test

# 4. Start a service
cd saas/services/auth
go run .

# 5. Check metrics
kubectl port-forward svc/auth-service 8095:8081 -n fintech-saas
curl http://localhost:8095/metrics
```

### Environment Variables
```bash
# Auth service
DATABASE_URL=postgresql://fintech_user:fintech_pass@localhost:5432/fintech
JWT_ACCESS_SECRET=your-access-secret
JWT_REFRESH_SECRET=your-refresh-secret
PORT=8081

# AI service
DATABASE_URL=...
ANTHROPIC_API_KEY=your-key
PORT=8085

# Pipeline service
DATABASE_URL=...
PORT=8086
```

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| **Services** | Go 1.26 |
| **Database** | PostgreSQL 15 · Row Level Security |
| **Transforms** | dbt 1.10 |
| **ETL** | Python 3.12 · pandas · numpy |
| **Containers** | Docker · Kubernetes 1.34 |
| **Cloud** | Azure Container Apps · ACR · Container Apps |
| **CI/CD** | GitHub Actions |
| **IaC** | Terraform |
| **Observability** | Prometheus · Grafana · PromQL |
| **Frontend** | React · Recharts |

---

## Project Structure

```
fintech-saas-pipeline/
├── .github/workflows/
│   ├── azure-deploy.yml       # production CI/CD
│   ├── ci.yml                 # lint + test
│   └── pipeline-ci.yml        # data pipeline tests
├── saas/
│   ├── db/schema.sql          # multi-tenant PostgreSQL schema
│   └── services/
│       ├── auth/              # Go · JWT · Prometheus
│       ├── ai/                # Go · Claude API
│       ├── pipeline/          # Go · CSV upload
│       ├── features/          # Go · ML feature store
│       └── monitor/           # Go · freshness monitor
├── data-pipeline/
│   ├── ingestion/
│   │   ├── generate_data.py   # 5,000 row synthetic dataset
│   │   └── ingest.py          # ETL loader
│   └── dbt/
│       ├── models/            # 5 dbt models
│       └── tests/             # 16 tests
├── k8s/                       # Kubernetes manifests
├── LIVE_URLS.md               # Azure live endpoints
└── README.md
```

---

## Author

**Carlos Mario Medina Taboada**
Victoria, BC, Canada · [carlosmedinat@gmail.com](mailto:carlosmedinat@gmail.com)
[LinkedIn](https://linkedin.com/in/carlosmedinat) · [GitHub](https://github.com/charlesnet76) · [Portfolio](https://charlesnet76.github.io)
