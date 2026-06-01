# FinTech SaaS Pipeline — Project Document
## Technical Stack · Status · SOP · Roadmap

**Author:** Carlos Medina · carlosmedinat@gmail.com  
**GitHub:** github.com/charlesnet76  
**Last updated:** May 2026

---

## 1. What Are We Building?

A **multi-tenant SaaS analytics platform** for FinTech companies.

### The Problem
Small FinTech startups and accounting firms generate transaction data daily but have no data team to make sense of it. They can't afford Tableau or a data engineer. They just want to upload their data and see what's happening.

### The Solution
A self-serve platform where a FinTech company:
1. Signs up and gets their own isolated tenant
2. Uploads transaction CSV data (or connects an API)
3. Gets instant analytics — revenue trends, category breakdown, geographic distribution
4. Gets AI-generated insights — anomalies, recommendations, executive summaries
5. Everything monitored, observable, and production-grade

### Target Customer
Small FinTech startups, credit unions, accounting firms — anyone with transaction data but no data team. Priced $49–$199/month.

---

## 2. Project Status — What's Done vs Missing

### ✅ COMPLETE

| Component | Description | Location |
|-----------|-------------|----------|
| **SaaS API v1** | Node.js REST API — JWT auth, RBAC, versioned routes, unified response envelope, cursor pagination | `saas-api-starter/` · Live on Railway |
| **Multi-tenant DB schema** | 10 tables, org_id on every tenant table, Row Level Security | `saas/db/schema.sql` |
| **ETL ingestion** | pandas — extract, validate, clean, bulk load to PostgreSQL | `data-pipeline/ingestion/` |
| **Mock data generator** | 5,000 realistic Canadian FinTech transactions | `data-pipeline/ingestion/generate_data.py` |
| **dbt transform models** | 5 models, 16 tests all passing — stg → fct → rpt layers | `dbt-dashboard/fintech_dbt/` |
| **React dashboard** | Revenue trends, category breakdown, province analysis | `dbt-dashboard/dashboard/` |
| **Go auth service** | JWT, bcrypt, refresh token rotation, org-aware | `saas/services/auth/` |
| **Go AI service** | Claude API integration, 3 insight types, stored per tenant | `saas/services/ai/` |
| **Kubernetes deployment** | 3 pods running — auth, AI, postgres | Docker Desktop K8s |
| **Prometheus metrics** | `/metrics` endpoint on auth service, ServiceMonitor configured | `saas/services/auth/` |
| **Grafana dashboards** | Real-time PromQL graphs — request rate, latency | localhost:3001 |
| **GitHub Actions CI/CD** | 6 jobs — lint, test, build images, deploy | `.github/workflows/` |
| **Portfolio site** | Dark terminal aesthetic, all projects linked | charlesnet76.github.io |
| **GitHub profile README** | NPower Canada, live projects, Victoria BC | github.com/charlesnet76 |

---

### 🔄 IN PROGRESS

| Component | What's Missing | Priority |
|-----------|---------------|----------|
| **Prometheus on AI service** | `/metrics` endpoint not added yet | High |
| **Grafana custom dashboard** | Only using Explore — no saved dashboard | Medium |
| **Contact form** | Portfolio form not wired to email (Formspree) | Medium |
| **LinkedIn post** | Draft ready, not published yet | 🔴 High |

---

### ⏳ PLANNED (Not Started)

| Component | Description | Priority |
|-----------|-------------|----------|
| **CSV upload endpoint** | Tenants upload their own data via API | High |
| **dbt docs lineage graph** | `dbt docs generate` screenshot for README | Medium |
| **Stripe billing** | Free → Starter ($49) → Pro ($199) plans | Medium |
| **Email verification** | SendGrid/Resend transactional email | Medium |
| **Terraform Azure AKS** | Real cloud deployment — provision with IaC | Medium |
| **Airflow orchestration** | Scheduled pipeline runs per tenant nightly | Low |
| **Gateway service** | Go API gateway — JWT verify, routing, rate limiting | Low |
| **Analytics service** | Go service serving rpt_ model data via API | Low |
| **React dashboard wire-up** | Connect dashboard to real PostgreSQL data via API | Medium |
| **Upwork profile update** | Add live URLs, new stack | 🔴 High |

---

## 3. Technical Stack

### Backend
| Technology | Version | Purpose |
|------------|---------|---------|
| Node.js | 20 | SaaS API v1 — REST endpoints |
| Express | 4.18 | HTTP framework |
| Go | 1.26 | Auth + AI microservices |
| Python | 3.12 | ETL pipeline, data generation |

### Database & Data
| Technology | Version | Purpose |
|------------|---------|---------|
| PostgreSQL | 15 | Primary database — multi-tenant with RLS |
| dbt | 1.10 | SQL transformation layer |
| pandas | 2.x | ETL ingestion and validation |

### DevOps & Infrastructure
| Technology | Version | Purpose |
|------------|---------|---------|
| Docker | 29.x | Containerization |
| Kubernetes | 1.34 | Container orchestration (Docker Desktop) |
| GitHub Actions | — | CI/CD — lint, test, build, deploy |
| Terraform | 1.15 | IaC for Azure AKS (ready, not applied) |
| Helm | — | K8s package manager |

### Observability
| Technology | Purpose |
|------------|---------|
| Prometheus | Metrics collection — scrapes `/metrics` every 15s |
| Grafana | Visualization — PromQL dashboards |
| Winston | Structured JSON logging in Node.js |

### Frontend
| Technology | Purpose |
|------------|---------|
| React | Analytics dashboard |
| Recharts | Charts — revenue, categories, provinces |

### Cloud & Hosting
| Technology | Purpose |
|------------|---------|
| Railway | SaaS API production deployment |
| GitHub Pages | Portfolio site |
| Azure | Target cloud for AKS (Terraform ready) |

---

## 4. Architecture

```
CLIENT (browser / API consumer)
        │
        ▼
┌─────────────────────┐
│   SaaS API v1       │  Node.js · Railway · Live
│   /v1/auth/*        │  JWT + RBAC + rate limiting
│   /v1/users/*       │  Cursor pagination
│   /health /ready    │  K8s probes
└──────────┬──────────┘
           │
           ▼
┌─────────────────────────────────────────┐
│           PostgreSQL (multi-tenant)     │
│   organizations · users · memberships  │
│   transactions · pipeline_runs         │
│   ai_insights · audit_logs             │
│   Row Level Security on all tenant     │
│   tables — org_id enforced at DB level │
└──────────┬──────────────────────────────┘
           │
           ▼
┌─────────────────────┐
│   ETL Pipeline      │  Python · pandas · dbt
│   generate_data.py  │  Mock data generator
│   ingest.py         │  Extract → validate → load
│   dbt models        │  stg → fct → rpt (16 tests)
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│   React Dashboard   │  Recharts · localhost:3000
│   Revenue trends    │  Category breakdown
│   Province analysis │  AI insights tab
└─────────────────────┘

KUBERNETES (Docker Desktop)
├── fintech-saas namespace
│   ├── auth-service (Go) → /metrics → Prometheus
│   ├── ai-service (Go)   → Claude API
│   └── postgres
└── monitoring namespace
    ├── Prometheus → scrapes every 15s
    └── Grafana   → PromQL dashboards
```

---

## 5. Standard Operating Procedures (SOP)

### SOP-01: Start the local development stack

```powershell
# 1. Start Docker Desktop (whale icon in taskbar)

# 2. Start PostgreSQL
docker start fintech-db

# 3. Start Kubernetes services
kubectl get pods -n fintech-saas
# If pods not running:
kubectl apply -k C:\Users\carlo\projects\fintech-saas-pipeline\saas\k8s\k8s-base\

# 4. Start port-forwards (each in separate window)
kubectl port-forward service/auth-service 8095:80 -n fintech-saas
kubectl port-forward svc/monitoring-grafana 3001:80 -n monitoring

# 5. Run dbt models
cd C:\Users\carlo\projects\fintech-saas-pipeline\dbt-dashboard\fintech_dbt
dbt run && dbt test

# 6. Start React dashboard
cd C:\Users\carlo\projects\fintech-saas-pipeline\dbt-dashboard\dashboard
npm start
```

### SOP-02: Run the data pipeline

```powershell
cd C:\Users\carlo\projects\fintech-saas-pipeline

# Generate mock data
python data-pipeline/ingestion/generate_data.py

# Run ingestion
python data-pipeline/ingestion/ingest.py

# Run dbt transforms
cd dbt-dashboard/fintech_dbt
dbt run
dbt test
```

### SOP-03: Deploy changes to production (SaaS API)

```powershell
cd C:\Users\carlo\projects\saas-api-starter
git add .
git commit -m "feat: description"
git push origin main
# Railway auto-deploys on push — check Railway dashboard
# Test: https://saas-api-starter-production.up.railway.app/health
```

### SOP-04: Check observability

```powershell
# Check pods
kubectl get pods -n fintech-saas
kubectl top pods -n fintech-saas

# Check metrics
kubectl port-forward service/auth-service 8095:80 -n fintech-saas
# Then: http://localhost:8095/metrics

# Grafana dashboard
kubectl port-forward svc/monitoring-grafana 3001:80 -n monitoring
# Then: http://localhost:3001 (admin/admin123)
# Query: rate(auth_http_requests_total{path="/health",status="200"}[1m])
```

### SOP-05: Push to GitHub

```powershell
# fintech-saas-pipeline
cd C:\Users\carlo\projects\fintech-saas-pipeline
git add .
git commit -m "feat: description"
git push origin main

# saas-api-starter
cd C:\Users\carlo\projects\saas-api-starter
git add .
git commit -m "feat: description"
git push origin main

# portfolio
cd C:\Users\carlo\projects\portfolio
git add .
git commit -m "feat: description"
git push origin main

# GitHub profile README
cd C:\Users\carlo\projects\github-profile
git add README.md
git commit -m "feat: description"
git push origin main
```

---

## 6. Live URLs

| Resource | URL |
|----------|-----|
| Portfolio | https://charlesnet76.github.io |
| GitHub profile | https://github.com/charlesnet76 |
| SaaS API health | https://saas-api-starter-production.up.railway.app/health |
| SaaS API version | https://saas-api-starter-production.up.railway.app/v1/version |
| Local auth service | http://localhost:8095 |
| Local Grafana | http://localhost:3001 |
| Local dashboard | http://localhost:3000 |

---

## 7. MVP → SaaS Roadmap

| Phase | Scope | Status |
|-------|-------|--------|
| **Phase 1 — MVP** | ETL + dbt + auth + AI + K8s + monitoring | ✅ Complete |
| **Phase 2 — Beta** | CSV upload, React dashboard wired to DB, Stripe billing | 🔄 Next |
| **Phase 3 — v1.0** | Email verification, usage limits, Airflow scheduling | ⏳ Planned |
| **Phase 4 — SaaS** | Self-serve onboarding, multi-source, white-label | ⏳ Future |

---

## 8. Repositories

| Repo | Purpose | Status |
|------|---------|--------|
| `charlesnet76/saas-api-starter` | Production SaaS API v1 | ✅ Live on Railway |
| `charlesnet76/fintech-saas-pipeline` | Full platform — ETL, dbt, Go services, K8s | ✅ GitHub |
| `charlesnet76/charlesnet76.github.io` | Portfolio site | ✅ Live |
| `charlesnet76/charlesnet76` | GitHub profile README | ✅ Live |
| `carlos-local/azure-devops-lab` | Terraform for Azure AKS | ✅ Ready (not applied) |
EOF