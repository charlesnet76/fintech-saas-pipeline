# FinTech SaaS Pipeline — Disaster Recovery Runbook

**Version:** 1.0  
**Owner:** Carlos Medina · carlosmedinat@gmail.com  
**Last updated:** June 2026  
**Platform:** Azure Container Apps · Canada Central

---

## RTO / RPO Targets

| Metric | Target | Definition |
|--------|--------|------------|
| **RTO** | 15 minutes | Time to restore service after failure |
| **RPO** | 24 hours | Maximum acceptable data loss window |
| **Uptime SLA** | 99.5% | ~3.6 hours downtime/month acceptable |

---

## Architecture Overview

```
GitHub Pages (dashboard) → Azure Container Apps (5 services) → PostgreSQL (Docker local)
                                    ↓
                          Azure Container Registry (images)
                                    ↓
                          GitHub Actions (CI/CD redeploy)
```

---

## Service Health Endpoints

| Service | Health URL | Expected Response |
|---------|-----------|-------------------|
| auth-service | /health | `{"service":"auth","status":"ok"}` |
| ai-service | /health | `{"service":"ai","status":"ok"}` |
| pipeline-service | /health | `{"service":"pipeline","status":"ok"}` |
| features-service | /health | `{"service":"features","status":"ok"}` |
| monitor-service | /health | `{"status":"degraded"}` (no DB — expected) |

---

## Incident Scenarios & Recovery Steps

### Scenario 1 — Single service down

**Symptoms:** One /health endpoint returning 5xx or not responding  
**RTO:** 5 minutes

```bash
# 1. Check logs
az containerapp logs show --name <service-name> --resource-group fintech-saas-rg --tail 50

# 2. Restart the revision
az containerapp revision restart \
  --name <service-name> \
  --resource-group fintech-saas-rg \
  --revision $(az containerapp show --name <service-name> \
    --resource-group fintech-saas-rg \
    --query "properties.latestRevisionName" -o tsv)

# 3. Verify health
curl https://<service-name>.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health
```

---

### Scenario 2 — All services down (environment failure)

**Symptoms:** All /health endpoints failing  
**RTO:** 15 minutes

```bash
# 1. Check environment status
az containerapp env show \
  --name fintech-saas-env \
  --resource-group fintech-saas-rg \
  --query "properties.provisioningState"

# 2. Check Azure status
# https://status.azure.com

# 3. Trigger full redeploy via GitHub Actions
# Go to: github.com/charlesnet76/fintech-saas-pipeline/actions
# Click: azure-deploy.yml → Run workflow

# 4. Verify all 5 services
for svc in auth-service ai-service pipeline-service features-service monitor-service; do
  curl https://$svc.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health
done
```

---

### Scenario 3 — Bad deployment (broken code pushed)

**Symptoms:** Services failing after a push to main  
**RTO:** 5 minutes

```bash
# 1. Identify last good commit
git log --oneline -10

# 2. Revert to last good commit
git revert HEAD
git push origin main
# GitHub Actions auto-redeploys

# OR rollback to specific commit
git revert <commit-hash>
git push origin main
```

---

### Scenario 4 — Database data loss

**Symptoms:** Queries failing, empty tables  
**RPO:** 24 hours (manual backup schedule)  
**RTO:** 30 minutes

```bash
# 1. Backup PostgreSQL (run daily)
docker exec fintech-db pg_dump \
  -U fintech_user \
  -d fintech \
  -F c \
  -f /tmp/fintech_backup_$(date +%Y%m%d).dump

# Copy backup out of container
docker cp fintech-db:/tmp/fintech_backup_$(date +%Y%m%d).dump ./backups/

# 2. Restore from backup
docker exec -i fintech-db pg_restore \
  -U fintech_user \
  -d fintech \
  -F c < ./backups/fintech_backup_YYYYMMDD.dump

# 3. Re-run dbt models
cd dbt-dashboard/fintech_dbt
dbt run
dbt test
```

---

### Scenario 5 — ACR image corruption

**Symptoms:** Container failing to start, image pull errors  
**RTO:** 10 minutes

```bash
# 1. Check ACR images
az acr repository list --name carlossaasacr --output table

# 2. Trigger fresh build and push
git commit --allow-empty -m "chore: force redeploy"
git push origin main

# GitHub Actions rebuilds all images from source
```

---

### Scenario 6 — GitHub Pages dashboard down

**Symptoms:** charlesnet76.github.io/fintech-saas-pipeline not loading  
**RTO:** 5 minutes

```bash
# 1. Check GitHub Pages status
# https://githubstatus.com

# 2. Redeploy dashboard
cd dbt-dashboard/dashboard
npm run deploy

# 3. Verify
curl https://charlesnet76.github.io/fintech-saas-pipeline/
```

---

## Backup Schedule (Manual — automate with cron)

```bash
# Daily backup script — run every night
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="./backups"
mkdir -p $BACKUP_DIR

# PostgreSQL backup
docker exec fintech-db pg_dump \
  -U fintech_user -d fintech -F c \
  -f /tmp/fintech_$DATE.dump

docker cp fintech-db:/tmp/fintech_$DATE.dump $BACKUP_DIR/

# Keep last 7 days only
find $BACKUP_DIR -name "*.dump" -mtime +7 -delete

echo "Backup complete: $BACKUP_DIR/fintech_$DATE.dump"
```

---

## Circuit Breaker — Already Implemented

All services have graceful degradation:
- **Redis unavailable** → rate limiting disabled, service continues ✅
- **PostgreSQL unavailable** → services start, return degraded status ✅  
- **Sentry unavailable** → error tracking disabled, service continues ✅
- **Claude API unavailable** → returns context-only response ✅

---

## Contact & Escalation

| Level | Contact | When |
|-------|---------|------|
| L1 | Carlos Medina · carlosmedinat@gmail.com | All incidents |
| L2 | Azure Support · portal.azure.com/support | Azure infrastructure issues |
| L3 | GitHub Support · support.github.com | CI/CD pipeline issues |

---

## Recovery Verification Checklist

After any incident, verify:

- [ ] All 5 /health endpoints returning 200
- [ ] GitHub Actions showing green on latest commit  
- [ ] Dashboard loading at charlesnet76.github.io/fintech-saas-pipeline
- [ ] /predict/forecast returning data
- [ ] /insights/ask returning AI response
- [ ] Sentry showing no new critical errors
- [ ] dbt tests passing (16/16)

---

*This runbook covers L12 — Availability & Disaster Recovery of the 15-layer architecture.*
