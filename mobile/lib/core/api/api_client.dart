import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;

// Override at build time: --dart-define=API_BASE_URL=http://192.168.x.x:8080
const String _baseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://10.0.2.2:8080', // Android emulator → host machine
);

// WebSocket scheme derived from the HTTP base URL. In production the API is
// served over TLS (see docs/adr/012-tls-reverse-proxy.md), so this must yield
// wss:// there — a ws:// URL against an https:// host simply fails to connect,
// and a release APK cannot fall back to cleartext either.
String get _wsBase => _baseUrl.startsWith('https://')
    ? _baseUrl.replaceFirst('https://', 'wss://')
    : _baseUrl.replaceFirst('http://', 'ws://');

class ApiException implements Exception {
  const ApiException(this.statusCode, this.message);
  final int statusCode;
  final String message;

  /// True when the server refused because a feature flag is off, as opposed to
  /// the other things that answer 503 (a request timeout, a database outage).
  /// Every feature gate in `internal/api/handlers.go` phrases its message as
  /// `<subject> is/are disabled`; the suffix is the contract.
  bool get isFeatureDisabled =>
      statusCode == 503 && message.toLowerCase().endsWith('disabled');

  @override
  String toString() => message;
}

/// Called whenever the server rejects a call because a feature flag is off.
/// Reaching this point means the client's cached flags are stale, so the app
/// wires this to [FeatureFlags.load] in `main.dart` to re-sync and stop
/// offering the feature. Left null in tests, which make no network calls.
void Function()? onFeatureDisabled;

// Builds the exception for a failed response and, when the cause is a disabled
// feature, nudges the app to refresh its flags.
ApiException _errorFor(int statusCode, dynamic decoded) {
  final e = ApiException(statusCode, _extractError(decoded, statusCode));
  if (e.isFeatureDisabled) {
    onFeatureDisabled?.call();
  }
  return e;
}

Map<String, String> _authHeaders(String? token) => {
      'Content-Type': 'application/json',
      if (token?.isNotEmpty == true) 'Authorization': 'Bearer $token',
    };

String _extractError(dynamic decoded, int statusCode) {
  final data = decoded is Map<String, dynamic> ? decoded : <String, dynamic>{};
  return data['message'] as String? ??
      data['error'] as String? ??
      'Request failed ($statusCode)';
}

class ApiClient {
  const ApiClient();

  Future<Map<String, dynamic>> post(
    String path,
    Map<String, dynamic> body, {
    String? token,
  }) async {
    final uri = Uri.parse('$_baseUrl$path');
    try {
      final response = await http
          .post(
            uri,
            headers: _authHeaders(token),
            body: jsonEncode(body),
          )
          .timeout(const Duration(seconds: 15));

      final decoded = jsonDecode(response.body);
      if (response.statusCode >= 400) {
        throw _errorFor(response.statusCode, decoded);
      }
      return decoded as Map<String, dynamic>;
    } on ApiException {
      rethrow;
    } on SocketException {
      throw const ApiException(
          0, 'Cannot reach the server. Check your connection.');
    } on HttpException {
      throw const ApiException(0, 'Network error. Try again.');
    }
  }

  Future<Map<String, dynamic>> patch(
    String path,
    Map<String, dynamic> body, {
    String? token,
  }) async {
    final uri = Uri.parse('$_baseUrl$path');
    try {
      final response = await http
          .patch(
            uri,
            headers: _authHeaders(token),
            body: jsonEncode(body),
          )
          .timeout(const Duration(seconds: 15));

      final decoded = jsonDecode(response.body);
      if (response.statusCode >= 400) {
        throw _errorFor(response.statusCode, decoded);
      }
      return decoded as Map<String, dynamic>;
    } on ApiException {
      rethrow;
    } on SocketException {
      throw const ApiException(
          0, 'Cannot reach the server. Check your connection.');
    } on HttpException {
      throw const ApiException(0, 'Network error. Try again.');
    }
  }

  // For endpoints that respond 204 No Content on success (e.g. favorite toggles):
  // POST with no request or response body.
  Future<void> postEmpty(String path, {String? token}) async {
    final uri = Uri.parse('$_baseUrl$path');
    try {
      final response = await http
          .post(uri, headers: _authHeaders(token))
          .timeout(const Duration(seconds: 15));

      if (response.statusCode >= 400) {
        final decoded =
            response.body.isNotEmpty ? jsonDecode(response.body) : null;
        throw _errorFor(response.statusCode, decoded);
      }
    } on ApiException {
      rethrow;
    } on SocketException {
      throw const ApiException(
          0, 'Cannot reach the server. Check your connection.');
    } on HttpException {
      throw const ApiException(0, 'Network error. Try again.');
    }
  }

  Future<dynamic> get(String path, {String? token}) async {
    final uri = Uri.parse('$_baseUrl$path');
    try {
      final response = await http
          .get(uri, headers: _authHeaders(token))
          .timeout(const Duration(seconds: 15));

      final decoded = jsonDecode(response.body);
      if (response.statusCode >= 400) {
        throw _errorFor(response.statusCode, decoded);
      }
      return decoded;
    } on ApiException {
      rethrow;
    } on SocketException {
      throw const ApiException(
          0, 'Cannot reach the server. Check your connection.');
    } on HttpException {
      throw const ApiException(0, 'Network error. Try again.');
    }
  }

  Future<void> delete(String path, {String? token}) async {
    final uri = Uri.parse('$_baseUrl$path');
    try {
      final response = await http
          .delete(uri, headers: _authHeaders(token))
          .timeout(const Duration(seconds: 15));

      if (response.statusCode >= 400) {
        final decoded =
            response.body.isNotEmpty ? jsonDecode(response.body) : null;
        throw _errorFor(response.statusCode, decoded);
      }
    } on ApiException {
      rethrow;
    } on SocketException {
      throw const ApiException(
          0, 'Cannot reach the server. Check your connection.');
    } on HttpException {
      throw const ApiException(0, 'Network error. Try again.');
    }
  }

  // WebSocket URL with token passed as query param (headers unsupported on WS upgrade).
  Uri wsUri(String path, {String? token}) {
    final base = '$_wsBase$path';
    if (token?.isNotEmpty != true) {
      return Uri.parse(base);
    }
    return Uri.parse('$base?token=${Uri.encodeQueryComponent(token!)}');
  }
}
