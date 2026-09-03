import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../../../core/api/api_client.dart';
import '../../user/services/user_service.dart';
import '../services/auth_service.dart';

// Decodes the JWT payload to extract the `role` claim without a library.
String? _jwtRole(String token) {
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
    final decoded =
        jsonDecode(utf8.decode(base64Url.decode(payload)))
            as Map<String, dynamic>;
    return decoded['role'] as String?;
  } catch (_) {
    return null;
  }
}

class AuthNotifier extends ChangeNotifier {
  AuthNotifier({ApiClient apiClient = const ApiClient()})
    : _svc = AuthService(apiClient),
      _userSvc = UserService(apiClient);

  final AuthService _svc;
  final UserService _userSvc;

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

  bool get hasAccess => isAuthenticated || isAnonymous;
  bool get isBroadcaster => role == 'broadcaster' || role == 'admin';
  bool get isAdmin => role == 'admin';

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
      isAuthenticated = true;
      accessToken = tokens.accessToken;
      role = _jwtRole(tokens.accessToken);
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
      isAuthenticated = true;
      accessToken = tokens.accessToken;
      role = _jwtRole(tokens.accessToken);
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
    isAuthenticated = false;
    isAnonymous = false;
    accessToken = null;
    role = null;
    username = null;
    email = null;
    notifyListeners();
  }
}
