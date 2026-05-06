import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';

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

final apiClientProvider = Provider<ApiClient>((_) => const ApiClient());

final _authServiceProvider = Provider<AuthService>(
  (ref) => AuthService(ref.watch(apiClientProvider)),
);

class AuthState {
  const AuthState({
    this.isLoading = false,
    this.error,
    this.isAuthenticated = false,
    this.isAnonymous = false,
    this.accessToken,
    this.role,
  });

  final bool isLoading;
  final String? error;
  final bool isAuthenticated;
  final bool isAnonymous;
  final String? accessToken;
  final String? role;

  bool get hasAccess => isAuthenticated || isAnonymous;
  bool get isBroadcaster => role == 'broadcaster' || role == 'admin';

  AuthState copyWith({
    bool? isLoading,
    String? error,
    bool? isAuthenticated,
    bool? isAnonymous,
    String? accessToken,
    String? role,
    bool clearError = false,
  }) {
    return AuthState(
      isLoading: isLoading ?? this.isLoading,
      error: clearError ? null : (error ?? this.error),
      isAuthenticated: isAuthenticated ?? this.isAuthenticated,
      isAnonymous: isAnonymous ?? this.isAnonymous,
      accessToken: accessToken ?? this.accessToken,
      role: role ?? this.role,
    );
  }
}

class AuthNotifier extends Notifier<AuthState> {
  @override
  AuthState build() => const AuthState();

  AuthService get _svc => ref.read(_authServiceProvider);

  Future<bool> signIn(String email, String password) async {
    state = state.copyWith(isLoading: true, clearError: true);
    try {
      final tokens = await _svc.login(email, password);
      state = state.copyWith(
        isLoading: false,
        isAuthenticated: true,
        accessToken: tokens.accessToken,
        role: _jwtRole(tokens.accessToken),
      );
      return true;
    } on ApiException catch (e) {
      state = state.copyWith(isLoading: false, error: e.message);
      return false;
    } catch (_) {
      state = state.copyWith(
        isLoading: false,
        error: 'Connection failed. Check your network.',
      );
      return false;
    }
  }

  Future<bool> register(
    String username,
    String email,
    String password,
  ) async {
    state = state.copyWith(isLoading: true, clearError: true);
    try {
      final tokens = await _svc.register(username, email, password);
      state = state.copyWith(
        isLoading: false,
        isAuthenticated: true,
        accessToken: tokens.accessToken,
        role: _jwtRole(tokens.accessToken),
      );
      return true;
    } on ApiException catch (e) {
      state = state.copyWith(isLoading: false, error: e.message);
      return false;
    } catch (_) {
      state = state.copyWith(
        isLoading: false,
        error: 'Connection failed. Check your network.',
      );
      return false;
    }
  }

  void continueAnonymously() {
    state = state.copyWith(isAnonymous: true, clearError: true);
  }

  void clearError() => state = state.copyWith(clearError: true);
}

final authNotifierProvider =
    NotifierProvider<AuthNotifier, AuthState>(AuthNotifier.new);
