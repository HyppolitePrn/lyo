import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Persists the token pair across app launches.
///
/// Abstracted so tests (and any future platform without a keystore) can swap in
/// [InMemoryTokenStore] instead of touching the real keychain.
abstract class TokenStore {
  Future<({String access, String refresh})?> read();
  Future<void> write({required String access, required String refresh});
  Future<void> clear();
}

/// Keychain (iOS) / EncryptedSharedPreferences (Android) backed store.
class SecureTokenStore implements TokenStore {
  const SecureTokenStore([
    this._storage = const FlutterSecureStorage(
      aOptions: AndroidOptions(encryptedSharedPreferences: true),
    ),
  ]);

  static const _accessKey = 'lyo.access_token';
  static const _refreshKey = 'lyo.refresh_token';

  final FlutterSecureStorage _storage;

  @override
  Future<({String access, String refresh})?> read() async {
    try {
      final access = await _storage.read(key: _accessKey);
      final refresh = await _storage.read(key: _refreshKey);
      if (access == null || refresh == null) {
        return null;
      }
      return (access: access, refresh: refresh);
    } catch (_) {
      // A corrupt or unreadable keystore must not block app start — the user
      // simply lands on the splash screen and signs in again.
      return null;
    }
  }

  @override
  Future<void> write({
    required String access,
    required String refresh,
  }) async {
    try {
      await _storage.write(key: _accessKey, value: access);
      await _storage.write(key: _refreshKey, value: refresh);
    } catch (_) {
      // Best-effort: the in-memory session stays usable for this run.
    }
  }

  @override
  Future<void> clear() async {
    try {
      await _storage.delete(key: _accessKey);
      await _storage.delete(key: _refreshKey);
    } catch (_) {
      // Nothing to recover from: sign-out already cleared the in-memory state.
    }
  }
}

/// Non-persistent store used by tests.
class InMemoryTokenStore implements TokenStore {
  ({String access, String refresh})? _pair;

  @override
  Future<({String access, String refresh})?> read() async => _pair;

  @override
  Future<void> write({required String access, required String refresh}) async {
    _pair = (access: access, refresh: refresh);
  }

  @override
  Future<void> clear() async {
    _pair = null;
  }
}
