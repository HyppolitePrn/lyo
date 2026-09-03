import 'package:flutter/foundation.dart';

import '../../../core/api/api_client.dart';
import '../../player/models/stream_model.dart';
import '../../playlist/models/playlist_model.dart';
import '../../track/models/track_model.dart';
import '../services/favorites_service.dart';

enum FavoritesStatus { idle, loading, ready, error }

class FavoritesNotifier extends ChangeNotifier {
  FavoritesNotifier({FavoritesService? favoritesService})
    : _svc = favoritesService ?? const FavoritesService(ApiClient());

  final FavoritesService _svc;

  FavoritesStatus status = FavoritesStatus.idle;
  String? error;

  List<Track> tracks = [];
  List<LiveStream> streams = [];
  List<Playlist> playlists = [];

  final Set<String> _favoriteTrackIds = {};
  final Set<String> _favoriteStreamIds = {};
  final Set<String> _favoritePlaylistIds = {};

  bool isTrackFavorited(String id) => _favoriteTrackIds.contains(id);
  bool isStreamFavorited(String id) => _favoriteStreamIds.contains(id);
  bool isPlaylistFavorited(String id) => _favoritePlaylistIds.contains(id);

  Future<void> load(String token) async {
    status = FavoritesStatus.loading;
    error = null;
    notifyListeners();
    try {
      final favs = await _svc.getFavorites(token);
      tracks = favs.tracks;
      streams = favs.streams;
      playlists = favs.playlists;
      _favoriteTrackIds
        ..clear()
        ..addAll(tracks.map((t) => t.id));
      _favoriteStreamIds
        ..clear()
        ..addAll(streams.map((s) => s.id));
      _favoritePlaylistIds
        ..clear()
        ..addAll(playlists.map((p) => p.id));
      status = FavoritesStatus.ready;
      notifyListeners();
    } on ApiException catch (e) {
      status = FavoritesStatus.error;
      error = e.message;
      notifyListeners();
    } catch (_) {
      status = FavoritesStatus.error;
      error = 'Could not load favorites.';
      notifyListeners();
    }
  }

  Future<void> toggleTrack(String trackId, String token) async {
    final wasFavorited = _favoriteTrackIds.contains(trackId);
    wasFavorited
        ? _favoriteTrackIds.remove(trackId)
        : _favoriteTrackIds.add(trackId);
    notifyListeners();
    try {
      if (wasFavorited) {
        await _svc.unfavoriteTrack(trackId, token);
        tracks = tracks.where((t) => t.id != trackId).toList();
      } else {
        await _svc.favoriteTrack(trackId, token);
      }
    } catch (_) {
      wasFavorited
          ? _favoriteTrackIds.add(trackId)
          : _favoriteTrackIds.remove(trackId);
    }
    notifyListeners();
  }

  Future<void> toggleStream(String streamId, String token) async {
    final wasFavorited = _favoriteStreamIds.contains(streamId);
    wasFavorited
        ? _favoriteStreamIds.remove(streamId)
        : _favoriteStreamIds.add(streamId);
    notifyListeners();
    try {
      if (wasFavorited) {
        await _svc.unfavoriteStream(streamId, token);
        streams = streams.where((s) => s.id != streamId).toList();
      } else {
        await _svc.favoriteStream(streamId, token);
      }
    } catch (_) {
      wasFavorited
          ? _favoriteStreamIds.add(streamId)
          : _favoriteStreamIds.remove(streamId);
    }
    notifyListeners();
  }

  Future<void> togglePlaylist(String playlistId, String token) async {
    final wasFavorited = _favoritePlaylistIds.contains(playlistId);
    wasFavorited
        ? _favoritePlaylistIds.remove(playlistId)
        : _favoritePlaylistIds.add(playlistId);
    notifyListeners();
    try {
      if (wasFavorited) {
        await _svc.unfavoritePlaylist(playlistId, token);
        playlists = playlists.where((p) => p.id != playlistId).toList();
      } else {
        await _svc.favoritePlaylist(playlistId, token);
      }
    } catch (_) {
      wasFavorited
          ? _favoritePlaylistIds.add(playlistId)
          : _favoritePlaylistIds.remove(playlistId);
    }
    notifyListeners();
  }
}
