import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../../../core/api/api_client.dart';
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
        jsonDecode(utf8.decode(base64Url.decode(payload))) as Map<String, dynamic>;
    return decoded['role'] as String?;
  } catch (_) {
    return null;
  }
}

class AuthNotifier extends ChangeNotifier {
  AuthNotifier({ApiClient apiClient = const ApiClient()})
      : _svc = AuthService(apiClient);

  final AuthService _svc;

  bool isLoading = false;
  String? error;
  bool isAuthenticated = false;
  bool isAnonymous = false;
  String? accessToken;
  String? role;

  bool get hasAccess => isAuthenticated || isAnonymous;
  bool get isBroadcaster => role == 'broadcaster' || role == 'admin';

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
}
