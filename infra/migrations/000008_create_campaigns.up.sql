CREATE TABLE IF NOT EXISTS campaigns (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    status       TEXT        NOT NULL DEFAULT 'draft'
                             CHECK (status IN ('draft', 'active', 'paused', 'completed')),
    targeting    JSONB       NOT NULL DEFAULT '{}',
    budget_cents BIGINT      CHECK (budget_cents >= 0),
    currency     TEXT        NOT NULL DEFAULT 'USD',
    start_date   DATE,
    end_date     DATE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_campaigns_org_id         ON campaigns (org_id);
CREATE INDEX idx_campaigns_org_id_status  ON campaigns (org_id, status);
CREATE INDEX idx_campaigns_org_id_created ON campaigns (org_id, created_at DESC);
