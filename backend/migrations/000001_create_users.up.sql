CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username              TEXT        NOT NULL UNIQUE,
    email                 TEXT        NOT NULL UNIQUE,
    password_hash         TEXT        NOT NULL,
    role                  TEXT        NOT NULL DEFAULT 'user'
                                      CHECK (role IN ('anonymous','user','broadcaster','admin')),
    favorite_track_ids    UUID[]      NOT NULL DEFAULT '{}',
    favorite_stream_ids   UUID[]      NOT NULL DEFAULT '{}',
    favorite_playlist_ids UUID[]      NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_email    ON users (email);
CREATE INDEX idx_users_username ON users (username);
