import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;

// Override at build time: --dart-define=API_BASE_URL=http://192.168.x.x:8080
const String _baseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://10.0.2.2:8080', // Android emulator → host machine
);

// WebSocket scheme derived from HTTP base URL
String get _wsBase => _baseUrl.replaceFirst('http', 'ws');

class ApiException implements Exception {
  const ApiException(this.statusCode, this.message);
  final int statusCode;
  final String message;

  @override
  String toString() => message;
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
        throw ApiException(
            response.statusCode, _extractError(decoded, response.statusCode));
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

  Future<dynamic> get(String path, {String? token}) async {
    final uri = Uri.parse('$_baseUrl$path');
    try {
      final response = await http
          .get(uri, headers: _authHeaders(token))
          .timeout(const Duration(seconds: 15));

      final decoded = jsonDecode(response.body);
      if (response.statusCode >= 400) {
        throw ApiException(
            response.statusCode, _extractError(decoded, response.statusCode));
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
        throw ApiException(
            response.statusCode, _extractError(decoded, response.statusCode));
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
