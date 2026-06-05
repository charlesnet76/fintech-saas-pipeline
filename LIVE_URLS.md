# Live Azure Deployments
5 microservices deployed to Azure Container Apps — Canada Central region.
CI/CD via GitHub Actions → Azure Container Registry → Container Apps.
| Service | Live URL | Stack |
|---------|----------|-------|
| Auth | https://auth-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health | Go · JWT · bcrypt |
| AI | https://ai-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health | Go · Claude API |
| Pipeline | https://pipeline-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health | Go · PostgreSQL · CSV |
| Features | https://features-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health | Go · ML feature store |
| Monitor | https://monitor-service.gentlebay-f6693cbb.canadacentral.azurecontainerapps.io/health | Go · Prometheus · SLA |
## Infrastructure
- Registry: carlossaasacr.azurecr.io
- Environment: fintech-saas-env · Canada Central
- CI/CD: GitHub Actions on every push to main
- IaC: Terraform (azure-devops-lab/)
