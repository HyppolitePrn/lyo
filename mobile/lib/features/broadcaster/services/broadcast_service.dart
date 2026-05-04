import '../../../core/api/api_client.dart';
import '../../player/models/stream_model.dart';

class BroadcastService {
  const BroadcastService(this._api);
  final ApiClient _api;

  Future<LiveStream> createStream(
    String title,
    String? description,
    String token,
  ) async {
    final body = <String, dynamic>{'title': title};
    if (description != null && description.isNotEmpty) {
      body['description'] = description;
    }
    final data = await _api.post('/streams', body, token: token);
    return LiveStream.fromJson(data);
  }

  Future<void> endStream(String id, String token) async {
    await _api.delete('/streams/$id', token: token);
  }
}
