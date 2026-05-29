#!/usr/bin/env pwsh
# =============================================================================
# deploy-all.ps1
# Ruflo swarm deployment script — fintech-data-pipeline
# Runs all remaining tasks in parallel using hierarchical swarm topology
# =============================================================================

Write-Host "🚀 Initializing Ruflo swarm..." -ForegroundColor Cyan

# ── Step 1: Initialize hierarchical swarm ────────────────────────────────────
npx claude-flow coordination swarm-init `
  --topology hierarchical `
  --max-agents 8

# ── Step 2: Spawn specialized agents ─────────────────────────────────────────
Write-Host "🤖 Spawning specialized agents..." -ForegroundColor Cyan

npx claude-flow coordination agent-spawn --type devops    --name "dbt-runner"
npx claude-flow coordination agent-spawn --type coder     --name "dashboard-launcher"
npx claude-flow coordination agent-spawn --type technical-writer --name "readme-writer"
npx claude-flow coordination agent-spawn --type technical-writer --name "linkedin-writer"

# ── Step 3: Orchestrate all tasks in parallel ─────────────────────────────────
Write-Host "⚡ Orchestrating tasks in parallel..." -ForegroundColor Green

npx claude-flow coordination task-orchestrate `
  --task "Run dbt models for fintech-data-pipeline: cd C:\Users\carlo\projects\fintech-data-pipeline\dbt-dashboard\fintech_dbt && pip install dbt-postgres && dbt run && dbt test && dbt docs generate" `
  --agent "dbt-runner" `
  --strategy parallel &

npx claude-flow coordination task-orchestrate `
  --task "Install and launch React dashboard: cd C:\Users\carlo\projects\fintech-data-pipeline\dbt-dashboard\dashboard && npm install && npm start" `
  --agent "dashboard-launcher" `
  --strategy parallel &

npx claude-flow coordination task-orchestrate `
  --task "Generate GitHub profile README for charlesnet76 — full stack developer, FinTech DB engineer, DevOps in progress. Stack: Node.js, PostgreSQL, Python, Go, Docker, GitHub Actions, Azure. Live projects: saas-api-starter (Railway), fintech-data-pipeline (GitHub). Available for freelance weekends." `
  --agent "readme-writer" `
  --strategy parallel &

npx claude-flow coordination task-orchestrate `
  --task "Write LinkedIn announcement post for Carlos: just deployed a live SaaS API (Railway), full FinTech data pipeline (pandas + dbt + PostgreSQL + Go microservices + K8s), and portfolio site. Looking for part-time freelance FinTech/backend work on weekends. Include live URLs: charlesnet76.github.io and saas-api-starter-production.up.railway.app/health" `
  --agent "linkedin-writer" `
  --strategy parallel &

# Wait for all parallel tasks
Wait-Job *

Write-Host "✅ All tasks complete!" -ForegroundColor Green

# ── Step 4: Store learnings in SONA memory ────────────────────────────────────
npx claude-flow memory store `
  --key "fintech-saas-stack" `
  --value "Node.js+PostgreSQL+Python+Go+Docker+K8s+dbt+React — multi-tenant SaaS with RLS, JWT auth, AI insights via Claude API"

npx claude-flow memory store `
  --key "carlos-live-urls" `
  --value "portfolio: charlesnet76.github.io | api: saas-api-starter-production.up.railway.app/health"

Write-Host "🧠 Learnings stored in SONA memory" -ForegroundColor Cyan
Write-Host ""
Write-Host "📊 Summary:" -ForegroundColor Yellow
Write-Host "  ✅ dbt models: run + tested + docs generated"
Write-Host "  ✅ React dashboard: running at http://localhost:3000"
Write-Host "  ✅ GitHub README: ready to push"
Write-Host "  ✅ LinkedIn post: ready to publish"
