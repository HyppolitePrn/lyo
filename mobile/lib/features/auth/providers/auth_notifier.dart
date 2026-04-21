import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/api/api_client.dart';
import '../services/auth_service.dart';

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
  });

  final bool isLoading;
  final String? error;
  final bool isAuthenticated;
  final bool isAnonymous;

  bool get hasAccess => isAuthenticated || isAnonymous;

  AuthState copyWith({
    bool? isLoading,
    String? error,
    bool? isAuthenticated,
    bool? isAnonymous,
    bool clearError = false,
  }) {
    return AuthState(
      isLoading: isLoading ?? this.isLoading,
      error: clearError ? null : (error ?? this.error),
      isAuthenticated: isAuthenticated ?? this.isAuthenticated,
      isAnonymous: isAnonymous ?? this.isAnonymous,
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
      await _svc.login(email, password);
      state = state.copyWith(isLoading: false, isAuthenticated: true);
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
      await _svc.register(username, email, password);
      state = state.copyWith(isLoading: false, isAuthenticated: true);
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
