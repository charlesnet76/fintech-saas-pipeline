Excited to share what I've shipped to production this week. 🚀

**5 microservices — live on Azure Container Apps 🇨🇦**

Built and deployed an end-to-end FinTech data platform from scratch:

🔐 Auth service — JWT + refresh token rotation + RBAC
🤖 AI service — financial insights via Claude API
📊 Pipeline service — CSV ingestion → PostgreSQL
🧠 Features service — ML feature store with point-in-time retrieval
📡 Monitor service — freshness monitoring + SLA alerts

All 5 running live in Canada Central:
→ https://auth-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health
→ https://pipeline-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health

**Also live: SaaS API on Railway**
Production REST API — versioned routes, JWT auth, RBAC, rate limiting.
→ https://saas-api-starter-production.up.railway.app/health

**What's under the hood:**
→ Go microservices with Prometheus metrics
→ PostgreSQL + dbt (16/16 tests passing)
→ Python ETL — 5,000 row Canadian FinTech dataset
→ GitHub Actions CI/CD → Azure Container Registry → Container Apps
→ Kubernetes + Grafana observability stack locally

This isn't tutorials. These are real systems running in production.

Portfolio: https://charlesnet76.github.io
GitHub: https://github.com/charlesnet76

NPower Canada alumni 🎓 · AZ-900 certified ☁️ · Open to full-time dev roles in Canada 🇨🇦

Stack: Go · Node.js · Python · PostgreSQL · dbt · Docker · Kubernetes · Azure · GitHub Actions · Terraform · Prometheus · Grafana

#FinTech #SaaS #BackendDevelopment #DevOps #Azure #DataEngineering #OpenToWork #BuildInPublic #NPowerCanada #Victoria #Canada
