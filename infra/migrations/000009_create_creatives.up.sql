CREATE TABLE IF NOT EXISTS creatives (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    campaign_id     UUID        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    type            TEXT        NOT NULL CHECK (type IN ('html5', 'image', 'video')),
    file_url        TEXT        NOT NULL,
    file_size_bytes BIGINT      CHECK (file_size_bytes > 0),
    preview_url     TEXT,
    is_active       BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_creatives_campaign_id        ON creatives (campaign_id);
CREATE INDEX idx_creatives_org_id             ON creatives (org_id);
CREATE INDEX idx_creatives_campaign_id_active ON creatives (campaign_id, is_active);
