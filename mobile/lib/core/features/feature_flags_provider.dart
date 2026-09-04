import 'package:flutter/foundation.dart';

import '../api/api_client.dart';
import 'feature_flags_service.dart';

/// App-wide feature flag state, backed by the `feature_flags` table.
///
/// [defaults] are what the UI renders with until [load] answers, and what it
/// falls back to when the API is unreachable — a flag the server never
/// confirmed must not silently hide a working feature. They mirror
/// `backend/internal/features/seed.go`; the server's answer always wins.
class FeatureFlags extends ChangeNotifier {
  FeatureFlags({FeatureFlagsService? service})
      : _svc = service ?? const FeatureFlagsService(ApiClient());

  final FeatureFlagsService _svc;

  Map<String, bool> _flags = Map.of(defaults);

  /// True once the server's flags have been applied at least once.
  bool loaded = false;

  bool isEnabled(String flag) => _flags[flag] ?? false;

  /// Fetches the live flags. Never throws: a failure leaves [defaults] in place
  /// so the app stays usable offline.
  Future<void> load() async {
    try {
      _flags = {...defaults, ...await _svc.fetch()};
      loaded = true;
    } on ApiException {
      return;
    } catch (_) {
      return;
    } finally {
      notifyListeners();
    }
  }

  /// Applies a flag change an admin just made, so the rest of the app reacts
  /// without a round trip.
  void applyToggle(String name, bool enabled) {
    if (_flags[name] == enabled) {
      return;
    }
    _flags = {..._flags, name: enabled};
    notifyListeners();
  }

  static const Map<String, bool> defaults = {
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
