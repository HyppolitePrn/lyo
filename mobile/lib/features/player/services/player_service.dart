import '../../../core/api/api_client.dart';
import '../models/stream_model.dart';

class PlayerService {
  const PlayerService(this._api);
  final ApiClient _api;

  Future<List<LiveStream>> listLive(String token) async {
    final data = await _api.get('/streams', token: token);
    final items = (data as Map<String, dynamic>)['items'] as List<dynamic>;
    return items
        .cast<Map<String, dynamic>>()
        .map(LiveStream.fromJson)
        .toList();
  }

  Future<LiveStream> getStream(String id, String token) async {
    final data = await _api.get('/streams/$id', token: token);
    return LiveStream.fromJson(data as Map<String, dynamic>);
  }
}
