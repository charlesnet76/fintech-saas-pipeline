-- ── Migration 001: Feature Store ─────────────────────────────────────────────
-- Run this against your existing PostgreSQL database.
-- Assumes: organizations table with id UUID already exists (from auth service).

-- ── Feature Registry ──────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS feature_registry (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id               UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name                 TEXT        NOT NULL,
    description          TEXT,
    dtype                TEXT        NOT NULL CHECK (dtype IN ('float', 'int', 'bool', 'string', 'json')),
    owner                TEXT,
    freshness_sla_minutes INT        NOT NULL DEFAULT 60,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMPTZ,
    UNIQUE(org_id, name)
);

ALTER TABLE feature_registry ENABLE ROW LEVEL SECURITY;

CREATE POLICY feature_registry_tenant_isolation ON feature_registry
    USING (org_id = current_setting('app.org_id')::UUID);

CREATE INDEX idx_feature_registry_org ON feature_registry(org_id)
    WHERE deleted_at IS NULL;

-- ── Feature Values (Point-in-Time) ────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS feature_values (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    feature_id  UUID        NOT NULL REFERENCES feature_registry(id) ON DELETE CASCADE,
    entity_id   TEXT        NOT NULL,
    value       JSONB       NOT NULL,
    valid_at    TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE feature_values ENABLE ROW LEVEL SECURITY;

CREATE POLICY feature_values_tenant_isolation ON feature_values
    USING (org_id = current_setting('app.org_id')::UUID);

-- Core point-in-time index — the most important index in this whole migration
CREATE INDEX idx_feature_values_pit
    ON feature_values(feature_id, entity_id, valid_at DESC);

CREATE INDEX idx_feature_values_org
    ON feature_values(org_id, feature_id);

-- ── Updated_at trigger ────────────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER feature_registry_updated_at
    BEFORE UPDATE ON feature_registry
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
