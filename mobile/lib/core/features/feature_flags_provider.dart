class FeatureFlags {
  const FeatureFlags();

  bool isEnabled(String flag) => _flags[flag] ?? false;

  static const Map<String, bool> _flags = {
    'live_streaming': true,
    'track_uploads': true,
    'chat_websocket': false,
    'recommendations': false,
    'offline_mode': false,
    'transcoding': false,
    'social_auth': false,
    'playlists': true,
    'favorites': true,
    'admin_supervision': true,
  };
}
