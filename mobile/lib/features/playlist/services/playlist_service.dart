import '../../../core/api/api_client.dart';
import '../models/playlist_model.dart';

class PlaylistService {
  const PlaylistService(this._api);
  final ApiClient _api;

  Future<List<Playlist>> listPlaylists({String? token}) async {
    final data = await _api.get('/playlists', token: token);
    final items = (data as Map<String, dynamic>)['items'] as List<dynamic>;
    return items.cast<Map<String, dynamic>>().map(Playlist.fromJson).toList();
  }

  Future<Playlist> getPlaylist(String id, {String? token}) async {
    final data = await _api.get('/playlists/$id', token: token);
    return Playlist.fromJson(data as Map<String, dynamic>);
  }

  Future<Playlist> createPlaylist({
    required String title,
    required String token,
    String? description,
    bool isPublic = false,
  }) async {
    final data = await _api.post(
      '/playlists',
      {
        'title': title,
        'description': ?description,
        'is_public': isPublic,
      },
      token: token,
    );
    return Playlist.fromJson(data);
  }

  Future<Playlist> updatePlaylist({
    required String id,
    required String token,
    String? title,
    String? description,
    bool? isPublic,
  }) async {
    final data = await _api.patch(
      '/playlists/$id',
      {
        'title': ?title,
        'description': ?description,
        'is_public': ?isPublic,
      },
      token: token,
    );
    return Playlist.fromJson(data);
  }

  Future<void> deletePlaylist(String id, String token) async {
    await _api.delete('/playlists/$id', token: token);
  }

  Future<Playlist> addTrack({
    required String playlistId,
    required String trackId,
    required String token,
  }) async {
    final data = await _api.post(
      '/playlists/$playlistId/tracks',
      {'track_id': trackId},
      token: token,
    );
    return Playlist.fromJson(data);
  }

  Future<Playlist> removeTrack({
    required String playlistId,
    required String trackId,
    required String token,
  }) async {
    await _api.delete('/playlists/$playlistId/tracks/$trackId', token: token);
    return getPlaylist(playlistId, token: token);
  }
}
