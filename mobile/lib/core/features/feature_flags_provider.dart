import 'package:flutter_riverpod/flutter_riverpod.dart';

class FeatureFlags {
  const FeatureFlags();

  bool isEnabled(String flag) => _flags[flag] ?? false;

  static const Map<String, bool> _flags = {
    'live_streaming': true,
    'chat_websocket': false,
    'recommendations': false,
    'offline_mode': false,
    'transcoding': false,
    'social_auth': false,
  };
}

final featureFlagsProvider = Provider<FeatureFlags>((_) => const FeatureFlags());
