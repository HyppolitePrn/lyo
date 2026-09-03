import '../../../core/api/api_client.dart';
import '../models/supervision_model.dart';

/// Reads the admin-only supervision endpoints. Every call is rejected with 403
/// by the backend for anything below the admin role, so the screen's own role
/// check is a convenience, not the security boundary.
class SupervisionService {
  const SupervisionService(this._api);
  final ApiClient _api;

  Future<SupervisionSummary> getSummary(String token) async {
    final data = await _api.get('/admin/supervision', token: token);
    return SupervisionSummary.fromJson(data as Map<String, dynamic>);
  }

  Future<List<Incident>> listIncidents({
    required String token,
    IncidentStatus? status,
    int limit = 50,
  }) async {
    final query = StringBuffer('/admin/incidents?limit=$limit');
    if (status != null) {
      query.write('&status=${status.name}');
    }
    final data = await _api.get(query.toString(), token: token);
    final items = (data as Map<String, dynamic>)['items'] as List<dynamic>;
    return items.cast<Map<String, dynamic>>().map(Incident.fromJson).toList();
  }

  Future<Incident> updateIncident({
    required String id,
    required String action,
    required String token,
  }) async {
    final data = await _api.patch(
      '/admin/incidents/$id',
      {'action': action},
      token: token,
    );
    return Incident.fromJson(data);
  }
}
