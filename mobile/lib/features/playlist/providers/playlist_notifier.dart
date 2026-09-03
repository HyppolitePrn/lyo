import 'package:flutter/foundation.dart';

import '../../../core/api/api_client.dart';
import '../models/playlist_model.dart';
import '../services/playlist_service.dart';

enum PlaylistStatus { idle, loading, ready, error }

class PlaylistNotifier extends ChangeNotifier {
  PlaylistNotifier({PlaylistService? playlistService})
    : _svc = playlistService ?? const PlaylistService(ApiClient());

  final PlaylistService _svc;

  PlaylistStatus status = PlaylistStatus.idle;
  List<Playlist> playlists = [];
  Playlist? selected;
  String? error;

  Future<void> loadMine(String token) async {
    status = PlaylistStatus.loading;
    error = null;
    notifyListeners();
    try {
      playlists = await _svc.listPlaylists(token: token);
      status = PlaylistStatus.ready;
      notifyListeners();
    } on ApiException catch (e) {
      status = PlaylistStatus.error;
      error = e.message;
      notifyListeners();
    } catch (_) {
      status = PlaylistStatus.error;
      error = 'Could not load playlists.';
      notifyListeners();
    }
  }

  Future<void> loadOne(String id, {String? token}) async {
    status = PlaylistStatus.loading;
    error = null;
    notifyListeners();
    try {
      selected = await _svc.getPlaylist(id, token: token);
      status = PlaylistStatus.ready;
      notifyListeners();
    } on ApiException catch (e) {
      status = PlaylistStatus.error;
      error = e.message;
      notifyListeners();
    } catch (_) {
      status = PlaylistStatus.error;
      error = 'Could not load playlist.';
      notifyListeners();
    }
  }

  Future<bool> create({
    required String title,
    required String token,
    String? description,
    bool isPublic = false,
  }) async {
    try {
      final p = await _svc.createPlaylist(
        title: title,
        token: token,
        description: description,
        isPublic: isPublic,
      );
      playlists = [p, ...playlists];
      notifyListeners();
      return true;
    } on ApiException catch (e) {
      error = e.message;
      notifyListeners();
      return false;
    } catch (_) {
      error = 'Could not create playlist.';
      notifyListeners();
      return false;
    }
  }

  Future<bool> delete(String id, String token) async {
    try {
      await _svc.deletePlaylist(id, token);
      playlists = playlists.where((p) => p.id != id).toList();
      if (selected?.id == id) {
        selected = null;
      }
      notifyListeners();
      return true;
    } on ApiException catch (e) {
      error = e.message;
      notifyListeners();
      return false;
    } catch (_) {
      error = 'Could not delete playlist.';
      notifyListeners();
      return false;
    }
  }

  Future<bool> addTrack({
    required String playlistId,
    required String trackId,
    required String token,
  }) async {
    try {
      final p = await _svc.addTrack(
        playlistId: playlistId,
        trackId: trackId,
        token: token,
      );
      _replace(p);
      return true;
    } on ApiException catch (e) {
      error = e.message;
      notifyListeners();
      return false;
    } catch (_) {
      error = 'Could not add track to playlist.';
      notifyListeners();
      return false;
    }
  }

  Future<bool> removeTrack({
    required String playlistId,
    required String trackId,
    required String token,
  }) async {
    try {
      final p = await _svc.removeTrack(
        playlistId: playlistId,
        trackId: trackId,
        token: token,
      );
      _replace(p);
      return true;
    } on ApiException catch (e) {
      error = e.message;
      notifyListeners();
      return false;
    } catch (_) {
      error = 'Could not remove track from playlist.';
      notifyListeners();
      return false;
    }
  }

  void _replace(Playlist p) {
    if (selected?.id == p.id) {
      selected = p;
    }
    playlists = [
      for (final existing in playlists) existing.id == p.id ? p : existing,
    ];
    notifyListeners();
  }
}
