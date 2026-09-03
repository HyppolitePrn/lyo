import '../../../core/api/api_client.dart';
import '../models/feature_flag_model.dart';

class AdminFeatureService {
  const AdminFeatureService(this._api);
  final ApiClient _api;

  Future<List<FeatureFlag>> listFlags({required String token}) async {
    final data = await _api.get('/admin/features', token: token);
    final items = (data as Map<String, dynamic>)['items'] as List<dynamic>;
    return items
        .cast<Map<String, dynamic>>()
        .map(FeatureFlag.fromJson)
        .toList();
  }

  Future<FeatureFlag> toggleFlag({
    required String name,
    required bool enabled,
    required String token,
  }) async {
    final data = await _api.patch(
      '/admin/features/$name/toggle',
      {'enabled': enabled},
      token: token,
    );
    return FeatureFlag.fromJson(data);
  }
}
