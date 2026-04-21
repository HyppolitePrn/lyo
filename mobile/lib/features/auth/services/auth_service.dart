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
}
