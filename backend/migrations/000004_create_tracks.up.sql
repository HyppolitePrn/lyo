CREATE TABLE tracks (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    broadcaster_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title            TEXT        NOT NULL,
    artist           TEXT        NOT NULL DEFAULT '',
    audio_url        TEXT        NOT NULL,
    duration_seconds INT         NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tracks_broadcaster ON tracks (broadcaster_id);
