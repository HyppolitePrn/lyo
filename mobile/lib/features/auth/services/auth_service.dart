import '../../../core/api/api_client.dart';

class AuthTokens {
  const AuthTokens({required this.accessToken, required this.refreshToken});
  final String accessToken;
  final String refreshToken;
}

class AuthService {
  const AuthService(this._api);
  final ApiClient _api;

  Future<AuthTokens> login(String email, String password) async {
    final data = await _api.post('/auth/login', {
      'email': email,
      'password': password,
    });
    return AuthTokens(
      accessToken: data['access_token'] as String,
      refreshToken: data['refresh_token'] as String,
    );
  }

  Future<AuthTokens> register(
    String username,
    String email,
    String password,
  ) async {
    final data = await _api.post('/auth/register', {
      'username': username,
      'email': email,
      'password': password,
    });
    return AuthTokens(
      accessToken: data['access_token'] as String,
      refreshToken: data['refresh_token'] as String,
    );
  }

  Future<AuthTokens> refresh(String refreshToken) async {
    final data = await _api.post('/auth/refresh', {
      'refresh_token': refreshToken,
    });
    return AuthTokens(
      accessToken: data['access_token'] as String,
      refreshToken: data['refresh_token'] as String,
    );
  }

  Future<void> forgotPassword(String email) async {
    await _api.post('/auth/forgot-password', {'email': email});
  }

  Future<void> resetPassword(String token, String password) async {
    await _api.post('/auth/reset-password', {
      'token': token,
      'password': password,
    });
  }
}
