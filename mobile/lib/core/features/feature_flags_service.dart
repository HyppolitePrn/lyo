import '../api/api_client.dart';

class FeatureFlagsService {
  const FeatureFlagsService(this._api);
  final ApiClient _api;

  // GET /features is open to anonymous callers: the app needs its flags before
  // login to know which sign-in options and tabs to render.
  Future<Map<String, bool>> fetch() async {
    final data = await _api.get('/features');
    return (data as Map<String, dynamic>).map(
      (name, enabled) => MapEntry(name, enabled as bool),
    );
  }
}
