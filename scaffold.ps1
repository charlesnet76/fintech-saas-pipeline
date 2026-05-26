# scaffold.ps1
# Creates remaining files for fintech-saas-pipeline
# Run from repo root: .\scaffold.ps1

Write-Host "Scaffolding fintech-saas-pipeline..." -ForegroundColor Cyan

# --- Service READMEs ---
$services = @("api-gateway", "transactions-svc", "accounts-svc", "notifications-svc")
foreach ($svc in $services) {
    $content = "# $svc`n`n> Service description here.`n`n## Run locally`n`n``````bash`ngo run cmd/main.go`n```````n"
    Set-Content -Path "services/$svc/README.md" -Value $content -Encoding UTF8
}
Write-Host "  [ok] Service READMEs" -ForegroundColor Green

# --- GitHub Actions CI ---
$ci = @'
name: CI

on:
  pull_request:
    branches: [main, develop]
  push:
    branches: [develop]

jobs:
  lint-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Set up Node
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Lint Go services
        run: echo "Lint placeholder - replace with golangci-lint"

      - name: Test Go services
        run: echo "Test placeholder - replace with go test ./..."

      - name: Lint dashboard
        working-directory: dashboard
        run: echo "npm run lint placeholder"
'@
Set-Content -Path ".github/workflows/ci.yml" -Value $ci -Encoding UTF8
Write-Host "  [ok] CI workflow" -ForegroundColor Green

# --- dbt project ---
$dbtProject = @'
name: 'fintech_pipeline'
version: '1.0.0'
config-version: 2

profile: 'fintech_pipeline'

model-paths: ["models"]
test-paths: ["tests"]
macro-paths: ["macros"]

target-path: "target"
clean-targets:
  - "target"
  - "dbt_packages"

models:
  fintech_pipeline:
    staging:
      +materialized: view
    intermediate:
      +materialized: ephemeral
    marts:
      +materialized: table
'@
Set-Content -Path "data-pipeline/dbt/dbt_project.yml" -Value $dbtProject -Encoding UTF8

$dbtProfiles = @'
fintech_pipeline:
  target: dev
  outputs:
    dev:
      type: postgres
      host: localhost
      user: postgres
      password: postgres
      port: 5432
      dbname: fintech_dev
      schema: public
      threads: 4
'@
Set-Content -Path "data-pipeline/dbt/profiles.yml.example" -Value $dbtProfiles -Encoding UTF8
Write-Host "  [ok] dbt project files" -ForegroundColor Green

# --- Docker Compose ---
$dockerCompose = @'
version: '3.9'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: fintech_dev
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
'@
Set-Content -Path "infrastructure/docker/docker-compose.yml" -Value $dockerCompose -Encoding UTF8
Write-Host "  [ok] docker-compose.yml" -ForegroundColor Green

# --- Makefile ---
$makefile = @'
.PHONY: help up down test lint build

help:
	@echo "Available commands:"
	@echo "  make up       - Start local dev environment"
	@echo "  make down     - Stop local dev environment"
	@echo "  make test     - Run all tests"
	@echo "  make lint     - Run linters"
	@echo "  make build    - Build all services"

up:
	docker compose -f infrastructure/docker/docker-compose.yml up -d

down:
	docker compose -f infrastructure/docker/docker-compose.yml down

test:
	@for svc in services/*/; do echo "Testing $$svc"; done

lint:
	@for svc in services/*/; do echo "Linting $$svc"; done

build:
	@for svc in services/*/; do echo "Building $$svc"; done
'@
Set-Content -Path "Makefile" -Value $makefile -Encoding UTF8
Write-Host "  [ok] Makefile" -ForegroundColor Green

# --- Docs ---
$arch = @'
# Architecture

## Overview
FinTech SaaS pipeline built with Go microservices, dbt analytics, PostgreSQL, Kubernetes, and a React dashboard.

## Services
- **api-gateway** - Entry point, JWT auth, request routing
- **transactions-svc** - Core transaction processing
- **accounts-svc** - User and account management
- **notifications-svc** - Email and webhook delivery

## Data Layer
- **PostgreSQL** - Primary OLTP database
- **dbt** - Analytics transformations into marts
- **Redis** - Caching and rate limiting

## Infrastructure
- **Kubernetes** - Container orchestration (Kustomize for env overlays)
- **Terraform** - Cloud resource provisioning
- **GitHub Actions** - CI/CD

## Diagrams
See `docs/diagrams/` for system architecture diagrams.
'@
Set-Content -Path "docs/ARCHITECTURE.md" -Value $arch -Encoding UTF8

$apiDoc = @'
# API Reference

Coming soon - OpenAPI 3.1 specs per service.
'@
Set-Content -Path "docs/API.md" -Value $apiDoc -Encoding UTF8

$contrib = @'
# Contributing

## Branch strategy
- `main` - Production-ready code (protected)
- `develop` - Integration branch
- `feature/*` - Feature branches off develop
- `fix/*` - Bug fixes
- `hotfix/*` - Emergency prod fixes off main

## Workflow
1. Branch off `develop`
2. Open PR back to `develop`
3. Tag a release to deploy to production
'@
Set-Content -Path "docs/CONTRIBUTING.md" -Value $contrib -Encoding UTF8
Write-Host "  [ok] Docs" -ForegroundColor Green

# --- Setup script ---
$setupScript = @'
#!/usr/bin/env bash
set -euo pipefail
echo "Setting up local dev environment..."
docker compose -f infrastructure/docker/docker-compose.yml up -d
echo "Done."
'@
Set-Content -Path "scripts/setup-dev.sh" -Value $setupScript -Encoding UTF8
Write-Host "  [ok] setup-dev.sh" -ForegroundColor Green

Write-Host ""
Write-Host "Scaffolding complete!" -ForegroundColor Cyan
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Run: git status" -ForegroundColor White
Write-Host "  2. Tell Claude to send the polished README and .gitignore" -ForegroundColor White
