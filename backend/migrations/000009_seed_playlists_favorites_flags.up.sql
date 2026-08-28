INSERT INTO feature_flags (name, enabled, description) VALUES
    ('playlists', true, 'Playlist creation and management'),
    ('favorites', true, 'Favoriting tracks, streams, and playlists')
ON CONFLICT (name) DO NOTHING;
