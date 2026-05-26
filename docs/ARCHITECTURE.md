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
