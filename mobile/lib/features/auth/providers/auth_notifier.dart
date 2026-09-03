import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../../../core/api/api_client.dart';
import '../../user/services/user_service.dart';
import '../services/auth_service.dart';
import '../services/token_store.dart';

// Decodes the JWT payload without a library. Returns null on anything
// malformed so a corrupt stored token is treated as "no session".
Map<String, dynamic>? _jwtPayload(String token) {
  try {
    final parts = token.split('.');
    if (parts.length != 3) {
      return null;
    }
    var payload = parts[1];
    final mod = payload.length % 4;
    if (mod == 2) {
      payload += '==';
    }
    if (mod == 3) {
      payload += '=';
    }
    return jsonDecode(utf8.decode(base64Url.decode(payload)))
        as Map<String, dynamic>;
  } catch (_) {
    return null;
  }
}

String? _jwtRole(String token) => _jwtPayload(token)?['role'] as String?;

DateTime? _jwtExpiry(String token) {
  final exp = _jwtPayload(token)?['exp'];
  if (exp is! num) {
    return null;
  }
  return DateTime.fromMillisecondsSinceEpoch(exp.toInt() * 1000, isUtc: true);
}

class AuthNotifier extends ChangeNotifier {
  AuthNotifier({
    ApiClient apiClient = const ApiClient(),
    TokenStore? tokenStore,
  }) : _svc = AuthService(apiClient),
       _userSvc = UserService(apiClient),
       _store = tokenStore ?? const SecureTokenStore();

  // Refresh this early so an in-flight request never carries a token that
  // expires mid-flight.
  static const _refreshSkew = Duration(minutes: 1);

  final AuthService _svc;
  final UserService _userSvc;
  final TokenStore _store;

  bool isLoading = false;
  String? error;
  bool isAuthenticated = false;
  bool isAnonymous = false;
  String? accessToken;
  String? role;
  String? username;
  String? email;
  bool resetEmailSent = false;
  bool resetPasswordSuccess = false;

  String? _refreshToken;
  DateTime? _accessExpiry;
  Timer? _refreshTimer;
  Future<bool>? _inFlightRefresh;

  bool get hasAccess => isAuthenticated || isAnonymous;
  bool get isBroadcaster => role == 'broadcaster' || role == 'admin';

  /// Reloads a session persisted by a previous launch, refreshing the access
  /// token when it has already expired. Returns true when a usable session
  /// was restored.
  Future<bool> restoreSession() async {
    final stored = await _store.read();
    if (stored == null) {
      return false;
    }

    final expiry = _jwtExpiry(stored.access);
    final stale =
        expiry == null ||
        expiry.isBefore(DateTime.now().toUtc().add(_refreshSkew));

    if (stale) {
      // The stored access token is unusable; the refresh token is the only way
      // back in, and it may have expired too.
      _refreshToken = stored.refresh;
      // A rejected refresh token clears the stored pair from inside
      // `_refreshSession`; a transient failure deliberately keeps it so the
      // next launch can try again instead of forcing a re-login.
      return refreshSession();
    }

    _adopt(access: stored.access, refresh: stored.refresh, persist: false);
    notifyListeners();
    await fetchProfile();
    return true;
  }

  /// Refreshes the access token if it is expired or about to be. Safe to call
  /// on every app resume — it is a no-op while the current token is good.
  Future<void> ensureFreshSession() async {
    if (!isAuthenticated || _refreshToken == null) {
      return;
    }
    final expiry = _accessExpiry;
    if (expiry != null &&
        expiry.isAfter(DateTime.now().toUtc().add(_refreshSkew))) {
      return;
    }
    await refreshSession();
  }

  /// Exchanges the refresh token for a new pair. Concurrent callers share one
  /// request so a resume plus a timer tick cannot burn two refresh tokens.
  Future<bool> refreshSession() {
    final pending = _inFlightRefresh;
    if (pending != null) {
      return pending;
    }
    final started = _refreshSession();
    _inFlightRefresh = started;
    return started.whenComplete(() {
      if (_inFlightRefresh == started) {
        _inFlightRefresh = null;
      }
    });
  }

  Future<bool> _refreshSession() async {
    final token = _refreshToken;
    if (token == null) {
      return false;
    }
    try {
      final tokens = await _svc.refresh(token);
      _adopt(access: tokens.accessToken, refresh: tokens.refreshToken);
      notifyListeners();
      return true;
    } on ApiException catch (e) {
      // Only an outright rejection means the session is gone; a transient
      // failure (5xx, timeout) keeps the tokens so a later attempt can work.
      if (e.statusCode == 401 || e.statusCode == 403) {
        await _forgetSession();
        notifyListeners();
      }
      return false;
    } catch (_) {
      return false;
    }
  }

  void _adopt({
    required String access,
    required String refresh,
    bool persist = true,
  }) {
    isAuthenticated = true;
    accessToken = access;
    _refreshToken = refresh;
    role = _jwtRole(access);
    _accessExpiry = _jwtExpiry(access);
    _scheduleRefresh();
    if (persist) {
      // Fire-and-forget: the session is already live in memory, and callers
      // must not wait on the keystore.
      unawaited(_store.write(access: access, refresh: refresh));
    }
  }

  void _scheduleRefresh() {
    _refreshTimer?.cancel();
    final expiry = _accessExpiry;
    if (expiry == null) {
      return;
    }
    final delay = expiry.difference(DateTime.now().toUtc()) - _refreshSkew;
    _refreshTimer = Timer(
      delay.isNegative ? Duration.zero : delay,
      () => unawaited(refreshSession()),
    );
  }

  Future<void> _forgetSession() async {
    _refreshTimer?.cancel();
    _refreshTimer = null;
    isAuthenticated = false;
    accessToken = null;
    _refreshToken = null;
    _accessExpiry = null;
    role = null;
    username = null;
    email = null;
    await _store.clear();
  }

  Future<void> fetchProfile() async {
    final token = accessToken;
    if (token == null) {
      return;
    }
    try {
      final user = await _userSvc.getMe(token);
      username = user.username;
      email = user.email;
      notifyListeners();
    } catch (_) {
      // Best-effort: the session stays valid even if the profile fetch fails.
    }
  }

  Future<bool> signIn(String email, String password) async {
    isLoading = true;
    error = null;
    notifyListeners();
    try {
      final tokens = await _svc.login(email, password);
      isLoading = false;
      _adopt(access: tokens.accessToken, refresh: tokens.refreshToken);
      notifyListeners();
      await fetchProfile();
      return true;
    } on ApiException catch (e) {
      isLoading = false;
      error = e.message;
      notifyListeners();
      return false;
    } catch (_) {
      isLoading = false;
      error = 'Connection failed. Check your network.';
      notifyListeners();
      return false;
    }
  }

  Future<bool> register(
    String username,
    String email,
    String password,
  ) async {
    isLoading = true;
    error = null;
    notifyListeners();
    try {
      final tokens = await _svc.register(username, email, password);
      isLoading = false;
      _adopt(access: tokens.accessToken, refresh: tokens.refreshToken);
      notifyListeners();
      await fetchProfile();
      return true;
    } on ApiException catch (e) {
      isLoading = false;
      error = e.message;
      notifyListeners();
      return false;
    } catch (_) {
      isLoading = false;
      error = 'Connection failed. Check your network.';
      notifyListeners();
      return false;
    }
  }

  Future<bool> forgotPassword(String email) async {
    isLoading = true;
    error = null;
    resetEmailSent = false;
    notifyListeners();
    try {
      await _svc.forgotPassword(email);
      isLoading = false;
      resetEmailSent = true;
      notifyListeners();
      return true;
    } on ApiException catch (e) {
      isLoading = false;
      error = e.message;
      notifyListeners();
      return false;
    } catch (_) {
      isLoading = false;
      error = 'Connection failed. Check your network.';
      notifyListeners();
      return false;
    }
  }

  Future<bool> resetPassword(String token, String password) async {
    isLoading = true;
    error = null;
    resetPasswordSuccess = false;
    notifyListeners();
    try {
      await _svc.resetPassword(token, password);
      isLoading = false;
      resetPasswordSuccess = true;
      notifyListeners();
      return true;
    } on ApiException catch (e) {
      isLoading = false;
      error = e.message;
      notifyListeners();
      return false;
    } catch (_) {
      isLoading = false;
      error = 'Connection failed. Check your network.';
      notifyListeners();
      return false;
    }
  }

  void continueAnonymously() {
    isAnonymous = true;
    error = null;
    notifyListeners();
  }

  void clearError() {
    error = null;
    notifyListeners();
  }

  void signOut() {
    isLoading = false;
    error = null;
    isAnonymous = false;
    resetEmailSent = false;
    resetPasswordSuccess = false;
    unawaited(_forgetSession());
    notifyListeners();
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }
}
