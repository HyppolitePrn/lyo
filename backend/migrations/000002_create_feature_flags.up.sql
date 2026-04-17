CREATE TABLE feature_flags (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL UNIQUE,
    enabled     BOOLEAN     NOT NULL DEFAULT false,
    description TEXT        NOT NULL DEFAULT '',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO feature_flags (name, enabled, description) VALUES
    ('live_streaming',  true,  'Live broadcast feature'),
    ('chat_websocket',  false, 'Live chat between listeners'),
    ('recommendations', false, 'Listen-history recommendations'),
    ('offline_mode',    false, 'Playlist caching for offline playback'),
    ('transcoding',     false, 'Adaptive bitrate transcoding')
ON CONFLICT (name) DO NOTHING;
