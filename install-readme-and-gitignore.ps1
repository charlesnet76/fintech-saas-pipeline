# install-readme-and-gitignore.ps1
# Writes the polished README.md and expanded .gitignore directly into the current directory.
# Run from C:\Users\carlo\fintech-saas-pipeline\

Write-Host "Writing README.md and .gitignore..." -ForegroundColor Cyan

# ============ README.md ============
$readme = @'
# FinTech SaaS Pipeline

> Production-grade data and transaction pipeline for modern FinTech - Go microservices, dbt analytics, PostgreSQL, Kubernetes, and a React dashboard.

[![CI](https://github.com/charlesnet76/fintech-saas-pipeline/actions/workflows/ci.yml/badge.svg)](https://github.com/charlesnet76/fintech-saas-pipeline/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![dbt](https://img.shields.io/badge/dbt-1.8-FF694B?logo=dbt&logoColor=white)](https://www.getdbt.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.30-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)](https://react.dev/)

---

## Why this project

Most FinTech tutorials stop at "user signs up and sees a balance." This pipeline is built to answer the questions that matter in production:

- How do you process **transactions safely at scale** with audit trails and idempotency?
- How do you ship **trustworthy analytics** to ops, finance, and compliance - without bypassing the data team?
- How do you keep **infrastructure reproducible** across local, staging, and production?

The repo is structured as a small but complete reference: services that own their domain, a data layer that exposes clean marts, and infrastructure that can be brought up locally in under a minute.

---

## Architecture

```
                          +---------------------+
                          |  React Dashboard    |
                          |  (Vite + TS)        |
                          +----------+----------+
                                     | HTTPS
                          +----------v----------+
                          |     API Gateway     |  JWT auth, routing, rate limit
                          +-+---------+-------+-+
              +-------------+         |       +--------------+
              |                       |                      |
   +----------v---------+  +----------v--------+  +----------v---------+
   |  Transactions Svc  |  |   Accounts Svc    |  | Notifications Svc  |
   +----------+---------+  +----------+--------+  +----------+---------+
              |                       |                      |
              +-----------+-----------+----------+-----------+
                          |                      |
                +---------v--------+    +--------v--------+
                |   PostgreSQL     |    |     Redis       |
                |   (OLTP + dbt)   |    |  (cache/queues) |
                +---------+--------+    +-----------------+
                          |
                +---------v--------+
                |   dbt Marts      |   staging -> intermediate -> marts
                |   (analytics)    |
                +------------------+
```

Detailed write-up: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)

---

## Tech stack

| Layer            | Tech                                          |
|------------------|-----------------------------------------------|
| Services         | Go 1.22, Gin / chi, sqlc, JWT                 |
| Data             | PostgreSQL 16, Redis 7, dbt 1.8               |
| Frontend         | React 18, TypeScript, Vite, TanStack Query    |
| Infrastructure   | Docker, Kubernetes (Kustomize), Terraform     |
| CI/CD            | GitHub Actions                                |
| Observability    | OpenTelemetry, Prometheus, Grafana            |

---

## Quick start

### Prerequisites
- Docker Desktop
- Go 1.22+
- Node 20+
- `make`

### Run locally
```bash
git clone https://github.com/charlesnet76/fintech-saas-pipeline.git
cd fintech-saas-pipeline

make up        # Start Postgres + Redis
make build     # Build all Go services
make test      # Run the test suite
```

Dashboard runs at `http://localhost:5173`, API gateway at `http://localhost:8080`.

### Tear down
```bash
make down
```

---

## Repo layout

```
.
|-- services/                # Go microservices
|   |-- api-gateway/
|   |-- transactions-svc/
|   |-- accounts-svc/
|   `-- notifications-svc/
|-- data-pipeline/
|   |-- dbt/                 # Analytics transformations
|   `-- airflow/             # (optional) orchestration
|-- infrastructure/
|   |-- k8s/                 # Kustomize manifests
|   |-- terraform/           # IaC
|   `-- docker/              # Local compose
|-- dashboard/               # React frontend
|-- docs/                    # Architecture, API, contributing
`-- scripts/                 # Dev tooling
```

---

## Roadmap

- [x] Repo scaffold + CI workflow
- [ ] Transactions service with idempotency keys
- [ ] dbt marts for daily transaction volume and chargebacks
- [ ] OpenAPI 3.1 specs per service
- [ ] Helm chart for production deployment
- [ ] OpenTelemetry tracing end-to-end
- [ ] Load test report (k6) at 1k RPS

---

## Contributing

See [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md) for branch strategy and PR workflow.

---

## License

[MIT](LICENSE) - built by [@charlesnet76](https://github.com/charlesnet76).
'@

Set-Content -Path "README.md" -Value $readme -Encoding UTF8
$readmeSize = (Get-Item "README.md").Length
Write-Host "  [ok] README.md written ($readmeSize bytes)" -ForegroundColor Green

# ============ .gitignore ============
$gitignore = @'
# ---------- Go ----------
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
coverage.html
coverage.txt
vendor/
go.work
go.work.sum

# ---------- Node / React ----------
node_modules/
.pnp
.pnp.js
.pnp.cjs
.pnpm-store/
dashboard/dist/
dashboard/build/
dashboard/.vite/
pids
*.pid
*.seed
*.pid.lock
coverage/
*.lcov
.nyc_output
npm-debug.log*
yarn-debug.log*
yarn-error.log*
pnpm-debug.log*
.npm
.yarn-integrity

# ---------- Python / dbt ----------
__pycache__/
*.py[cod]
*$py.class
.Python
*.egg-info/
.eggs/
dist/
build/
.venv/
venv/
env/
ENV/
data-pipeline/dbt/target/
data-pipeline/dbt/dbt_packages/
data-pipeline/dbt/logs/
data-pipeline/dbt/profiles.yml
data-pipeline/airflow/logs/
data-pipeline/airflow/airflow.cfg
data-pipeline/airflow/airflow.db
data-pipeline/airflow/webserver_config.py

# ---------- Terraform ----------
*.tfstate
*.tfstate.*
*.tfvars
*.tfvars.json
.terraform/
.terraform.lock.hcl
crash.log
crash.*.log
override.tf
override.tf.json
*_override.tf
*_override.tf.json

# ---------- Docker ----------
docker-compose.override.yml

# ---------- Kubernetes / Helm ----------
*.kubeconfig
charts/*.tgz

# ---------- Environment & secrets ----------
.env
.env.local
.env.*.local
.env.development
.env.production
*.pem
*.key
secrets/

# ---------- IDE / editors ----------
.idea/
.vscode/
*.swp
*.swo
*~
.project
.classpath
.settings/
*.sublime-project
*.sublime-workspace

# ---------- OS ----------
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db
Desktop.ini
$RECYCLE.BIN/

# ---------- Logs ----------
logs/
*.log

# ---------- Misc ----------
tmp/
temp/
.cache/
.history/
'@

Set-Content -Path ".gitignore" -Value $gitignore -Encoding UTF8
$gitignoreSize = (Get-Item ".gitignore").Length
Write-Host "  [ok] .gitignore written ($gitignoreSize bytes)" -ForegroundColor Green

Write-Host ""
Write-Host "Done! Verifying..." -ForegroundColor Cyan
Write-Host ""
Write-Host "--- README.md (first 3 lines) ---" -ForegroundColor Yellow
Get-Content "README.md" -TotalCount 3
Write-Host ""
Write-Host "--- .gitignore (first 3 lines) ---" -ForegroundColor Yellow
Get-Content ".gitignore" -TotalCount 3
Write-Host ""
Write-Host "Next: run 'git status' to see modified files." -ForegroundColor Yellow
