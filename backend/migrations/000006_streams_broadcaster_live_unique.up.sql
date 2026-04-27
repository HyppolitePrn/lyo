-- Enforce at most one live stream per broadcaster at the DB level.
-- This prevents the TOCTOU race between HasLive() and Create() in service.go.
CREATE UNIQUE INDEX uidx_streams_broadcaster_live
    ON streams (broadcaster_id)
    WHERE status = 'live';
