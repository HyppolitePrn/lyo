-- Incidents are alerts that were actually delivered to the platform, kept so
-- that an admin can see what fired while nobody was watching Grafana, and so
-- that acknowledgement survives a Grafana restart.
CREATE TABLE incidents (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Grafana's per-alert-instance fingerprint. It is the deduplication key:
    -- the same alert re-firing must update one incident, not create a queue.
    fingerprint     TEXT        NOT NULL,
    rule_uid        TEXT        NOT NULL DEFAULT '',
    title           TEXT        NOT NULL,
    summary         TEXT        NOT NULL DEFAULT '',
    severity        TEXT        NOT NULL DEFAULT 'warning'
                                CHECK (severity IN ('critical', 'warning', 'info')),
    -- technical  = something is broken.
    -- experience = nothing is broken and users are suffering anyway.
    family          TEXT        NOT NULL DEFAULT 'technical'
                                CHECK (family IN ('technical', 'experience')),
    status          TEXT        NOT NULL DEFAULT 'firing'
                                CHECK (status IN ('firing', 'acknowledged', 'resolved')),
    dashboard_url   TEXT        NOT NULL DEFAULT '',
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by UUID        REFERENCES users(id) ON DELETE SET NULL,
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- At most one open incident per fingerprint. Resolved rows are excluded so the
-- same alert firing again next week opens a new incident rather than reopening
-- last week's, which would destroy its history.
CREATE UNIQUE INDEX idx_incidents_open_fingerprint
    ON incidents (fingerprint) WHERE status <> 'resolved';

CREATE INDEX idx_incidents_status_started ON incidents (status, started_at DESC);

-- The supervision surface is a feature like any other and ships gated.
INSERT INTO feature_flags (name, enabled, description) VALUES
    ('admin_supervision', true, 'Admin supervision dashboard, incident feed and alert notifications')
ON CONFLICT (name) DO NOTHING;
