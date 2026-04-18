CREATE TABLE IF NOT EXISTS publisher_rules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    app_id     UUID NOT NULL REFERENCES publisher_apps(id) ON DELETE CASCADE,
    type       TEXT NOT NULL CHECK (type IN ('blocklist', 'allowlist', 'frequency_cap', 'geo_filter', 'platform_filter')),
    config     JSONB NOT NULL DEFAULT '{}',
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_publisher_rules_app_id ON publisher_rules (app_id);
CREATE INDEX idx_publisher_rules_org_id ON publisher_rules (org_id);
CREATE INDEX idx_publisher_rules_app_id_is_active ON publisher_rules (app_id, is_active);
