CREATE TYPE stream_status AS ENUM ('live', 'ended');

CREATE TABLE streams (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    broadcaster_id UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title          TEXT          NOT NULL,
    description    TEXT          NOT NULL DEFAULT '',
    status         stream_status NOT NULL DEFAULT 'live',
    started_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    ended_at       TIMESTAMPTZ
);

CREATE INDEX idx_streams_broadcaster ON streams (broadcaster_id);
CREATE INDEX idx_streams_status      ON streams (status);
