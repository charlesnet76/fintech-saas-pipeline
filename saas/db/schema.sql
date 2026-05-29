-- =============================================================================
-- FinTech SaaS Platform — Multi-tenant Database Schema
-- =============================================================================
-- Design principles:
--   1. Every table has org_id — data isolation enforced at DB level
--   2. Row Level Security (RLS) — even if app code has a bug, data won't leak
--   3. Soft deletes — nothing is permanently deleted (audit trail)
--   4. UUID primary keys — safe to expose in URLs, no enumeration attacks
--   5. Timestamptz — always UTC, never ambiguous
-- =============================================================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- TIER 1: Platform tables (no org_id — platform-level data)
-- =============================================================================

-- Organizations (tenants)
CREATE TABLE IF NOT EXISTS organizations (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name         VARCHAR(255) NOT NULL,
  slug         VARCHAR(100) UNIQUE NOT NULL,        -- used in URLs: app.com/acme
  plan         VARCHAR(50) NOT NULL DEFAULT 'free', -- free, starter, pro, enterprise
  status       VARCHAR(50) NOT NULL DEFAULT 'active',
  max_users    INTEGER NOT NULL DEFAULT 5,
  max_rows     BIGINT NOT NULL DEFAULT 100000,      -- pipeline row limit per org
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at   TIMESTAMPTZ
);

-- Users (belong to one or more orgs via memberships)
CREATE TABLE IF NOT EXISTS users (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email          VARCHAR(255) UNIQUE NOT NULL,
  password_hash  VARCHAR(255) NOT NULL,
  full_name      VARCHAR(255),
  status         VARCHAR(50) NOT NULL DEFAULT 'active',
  email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  last_login_at  TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at     TIMESTAMPTZ
);

-- Memberships (user ↔ org with role)
CREATE TABLE IF NOT EXISTS memberships (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role       VARCHAR(50) NOT NULL DEFAULT 'member', -- owner, admin, member, viewer
  invited_by UUID REFERENCES users(id),
  joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  UNIQUE(org_id, user_id)
);

-- Refresh tokens (platform-level auth)
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  token       VARCHAR(512) UNIQUE NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at  TIMESTAMPTZ
);

-- Subscription / billing
CREATE TABLE IF NOT EXISTS subscriptions (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  stripe_customer_id  VARCHAR(255),
  stripe_sub_id       VARCHAR(255),
  plan                VARCHAR(50) NOT NULL DEFAULT 'free',
  status              VARCHAR(50) NOT NULL DEFAULT 'active',
  current_period_end  TIMESTAMPTZ,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- TIER 2: Tenant tables (all have org_id — isolated per organization)
-- =============================================================================

-- Data sources (each org connects their own data)
CREATE TABLE IF NOT EXISTS data_sources (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name        VARCHAR(255) NOT NULL,
  type        VARCHAR(50) NOT NULL, -- csv_upload, api, database, s3
  config      JSONB,                -- connection details (encrypted at app level)
  status      VARCHAR(50) NOT NULL DEFAULT 'active',
  created_by  UUID REFERENCES users(id),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMPTZ
);

-- Raw transactions (per tenant)
CREATE TABLE IF NOT EXISTS transactions (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  source_id       UUID REFERENCES data_sources(id),
  transaction_id  VARCHAR(255) NOT NULL,             -- original ID from source
  customer_ref    VARCHAR(255),
  province        VARCHAR(10),
  age_group       VARCHAR(20),
  account_type    VARCHAR(50),
  merchant        VARCHAR(255),
  category        VARCHAR(100),
  amount          NUMERIC(15,2) NOT NULL,
  currency        VARCHAR(10) NOT NULL DEFAULT 'CAD',
  status          VARCHAR(50) NOT NULL,
  txn_date        DATE NOT NULL,
  txn_month       VARCHAR(7),
  txn_year        INTEGER,
  metadata        JSONB,                              -- flexible extra fields
  loaded_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(org_id, transaction_id)                     -- no dupes per tenant
);

-- Pipeline runs (per tenant)
CREATE TABLE IF NOT EXISTS pipeline_runs (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  source_id    UUID REFERENCES data_sources(id),
  triggered_by UUID REFERENCES users(id),
  status       VARCHAR(50) NOT NULL DEFAULT 'running',
  rows_loaded  INTEGER,
  rows_failed  INTEGER DEFAULT 0,
  error        TEXT,
  started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at  TIMESTAMPTZ
);

-- AI insights (per tenant)
CREATE TABLE IF NOT EXISTS ai_insights (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  run_id       UUID REFERENCES pipeline_runs(id),
  type         VARCHAR(100) NOT NULL, -- anomaly, forecast, summary, recommendation
  title        VARCHAR(255) NOT NULL,
  content      TEXT NOT NULL,         -- AI-generated insight text
  confidence   NUMERIC(5,4),          -- 0.0000 to 1.0000
  metadata     JSONB,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit log (per tenant — immutable)
CREATE TABLE IF NOT EXISTS audit_logs (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id    UUID REFERENCES users(id),
  action     VARCHAR(255) NOT NULL,  -- user.login, pipeline.run, data.export
  resource   VARCHAR(255),
  metadata   JSONB,
  ip_address INET,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Organizations
CREATE INDEX IF NOT EXISTS idx_orgs_slug     ON organizations(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_orgs_status   ON organizations(status);

-- Users
CREATE INDEX IF NOT EXISTS idx_users_email   ON users(email) WHERE deleted_at IS NULL;

-- Memberships
CREATE INDEX IF NOT EXISTS idx_memberships_org  ON memberships(org_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_memberships_user ON memberships(user_id) WHERE deleted_at IS NULL;

-- Transactions (most queried table)
CREATE INDEX IF NOT EXISTS idx_txn_org_date     ON transactions(org_id, txn_date);
CREATE INDEX IF NOT EXISTS idx_txn_org_category ON transactions(org_id, category);
CREATE INDEX IF NOT EXISTS idx_txn_org_status   ON transactions(org_id, status);
CREATE INDEX IF NOT EXISTS idx_txn_org_month    ON transactions(org_id, txn_month);

-- Pipeline runs
CREATE INDEX IF NOT EXISTS idx_runs_org     ON pipeline_runs(org_id);
CREATE INDEX IF NOT EXISTS idx_runs_status  ON pipeline_runs(status);

-- AI insights
CREATE INDEX IF NOT EXISTS idx_insights_org  ON ai_insights(org_id);
CREATE INDEX IF NOT EXISTS idx_insights_type ON ai_insights(org_id, type);

-- Audit log
CREATE INDEX IF NOT EXISTS idx_audit_org  ON audit_logs(org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);

-- =============================================================================
-- ROW LEVEL SECURITY (RLS)
-- =============================================================================
-- RLS ensures that even if application code has a bug,
-- a tenant can NEVER see another tenant's data.

ALTER TABLE transactions   ENABLE ROW LEVEL SECURITY;
ALTER TABLE pipeline_runs  ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_insights    ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs     ENABLE ROW LEVEL SECURITY;
ALTER TABLE data_sources   ENABLE ROW LEVEL SECURITY;

-- App sets this at the start of every DB session:
-- SET app.current_org_id = 'uuid-of-current-tenant';

CREATE POLICY tenant_isolation_transactions ON transactions
  USING (org_id = current_setting('app.current_org_id')::UUID);

CREATE POLICY tenant_isolation_pipeline_runs ON pipeline_runs
  USING (org_id = current_setting('app.current_org_id')::UUID);

CREATE POLICY tenant_isolation_ai_insights ON ai_insights
  USING (org_id = current_setting('app.current_org_id')::UUID);

CREATE POLICY tenant_isolation_audit_logs ON audit_logs
  USING (org_id = current_setting('app.current_org_id')::UUID);

CREATE POLICY tenant_isolation_data_sources ON data_sources
  USING (org_id = current_setting('app.current_org_id')::UUID);

-- =============================================================================
-- SEED: Demo organization for development
-- =============================================================================

INSERT INTO organizations (id, name, slug, plan)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'Demo Corp',
  'demo-corp',
  'pro'
) ON CONFLICT DO NOTHING;
