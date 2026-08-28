import '../../../core/api/api_client.dart';
import '../../playlist/models/playlist_model.dart';
import '../../player/models/stream_model.dart';
import '../../track/models/track_model.dart';

class Favorites {
  const Favorites({
    required this.tracks,
    required this.streams,
    required this.playlists,
  });

  final List<Track> tracks;
  final List<LiveStream> streams;
  final List<Playlist> playlists;

  factory Favorites.fromJson(Map<String, dynamic> json) {
    return Favorites(
      tracks: (json['tracks'] as List)
          .cast<Map<String, dynamic>>()
          .map(Track.fromJson)
          .toList(),
      streams: (json['streams'] as List)
          .cast<Map<String, dynamic>>()
          .map(LiveStream.fromJson)
          .toList(),
      playlists: (json['playlists'] as List)
          .cast<Map<String, dynamic>>()
          .map(Playlist.fromJson)
          .toList(),
    );
  }
}

class FavoritesService {
  const FavoritesService(this._api);
  final ApiClient _api;

  Future<Favorites> getFavorites(String token) async {
    final data = await _api.get('/users/me/favorites', token: token);
    return Favorites.fromJson(data as Map<String, dynamic>);
  }

  Future<void> favoriteTrack(String trackId, String token) =>
      _api.postEmpty('/users/me/favorites/tracks/$trackId', token: token);

  Future<void> unfavoriteTrack(String trackId, String token) =>
      _api.delete('/users/me/favorites/tracks/$trackId', token: token);

  Future<void> favoriteStream(String streamId, String token) =>
      _api.postEmpty('/users/me/favorites/streams/$streamId', token: token);

  Future<void> unfavoriteStream(String streamId, String token) =>
      _api.delete('/users/me/favorites/streams/$streamId', token: token);

  Future<void> favoritePlaylist(String playlistId, String token) => _api
      .postEmpty('/users/me/favorites/playlists/$playlistId', token: token);

  Future<void> unfavoritePlaylist(String playlistId, String token) =>
      _api.delete('/users/me/favorites/playlists/$playlistId', token: token);
}
