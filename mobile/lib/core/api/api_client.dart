import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;

// Override at build time: --dart-define=API_BASE_URL=http://192.168.x.x:8080
const String _baseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://10.0.2.2:8080', // Android emulator → host machine
);

class ApiException implements Exception {
  const ApiException(this.statusCode, this.message);
  final int statusCode;
  final String message;

  @override
  String toString() => message;
}

class ApiClient {
  const ApiClient();

  Future<Map<String, dynamic>> post(
    String path,
    Map<String, dynamic> body,
  ) async {
    final uri = Uri.parse('$_baseUrl$path');
    try {
      final response = await http
          .post(
            uri,
            headers: {'Content-Type': 'application/json'},
            body: jsonEncode(body),
          )
          .timeout(const Duration(seconds: 15));

      final decoded = jsonDecode(response.body);
      if (response.statusCode >= 400) {
        final data = decoded is Map<String, dynamic> ? decoded : <String, dynamic>{};
        final msg = data['message'] as String? ??
            data['error'] as String? ??
            'Request failed (${response.statusCode})';
        throw ApiException(response.statusCode, msg);
      }
      return decoded as Map<String, dynamic>;
    } on ApiException {
      rethrow;
    } on SocketException {
      throw const ApiException(0, 'Cannot reach the server. Check your connection.');
    } on HttpException {
      throw const ApiException(0, 'Network error. Try again.');
    }
  }
}
