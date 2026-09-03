import 'dart:async';

import 'package:flutter/foundation.dart';

import '../../../core/api/api_client.dart';
import '../../track/models/track_model.dart';
import '../../track/services/track_service.dart';
import '../services/lyo_audio_handler.dart';

// Plays a single track's static audio_url through the app-wide LyoAudioHandler
// (so the OS media notification / lock screen stay in sync) — distinct from
// PlayerNotifier, which pipes live WebSocket frames into just_audio instead.
class RecordedPlayerNotifier extends ChangeNotifier {
  RecordedPlayerNotifier({
    required LyoAudioHandler audioHandler,
    TrackService? trackService,
  }) : _audioHandler = audioHandler,
       _trackSvc = trackService ?? const TrackService(ApiClient()) {
    _stateSub = _audioHandler.playbackState.listen((s) {
      isPlaying = s.playing;
      notifyListeners();
    });
    _posSub = _audioHandler.positionStream.listen((p) {
      position = p;
      notifyListeners();
    });
    _durSub = _audioHandler.durationStream.listen((d) {
      duration = d ?? Duration.zero;
      notifyListeners();
    });
  }

  final LyoAudioHandler _audioHandler;
  final TrackService _trackSvc;

  Track? track;
  String? error;
  bool isLoading = true;
  bool isPlaying = false;
  Duration position = Duration.zero;
  Duration duration = Duration.zero;

  StreamSubscription<dynamic>? _stateSub;
  StreamSubscription<Duration>? _posSub;
  StreamSubscription<Duration?>? _durSub;

  // Called when navigating to the player screen. Keeps the current track
  // playing (instead of restarting it) if it's already loaded — this is what
  // lets playback survive leaving and re-entering the screen.
  Future<void> loadIfNeeded(String trackId, String? token) async {
    if (track?.id == trackId) {
      return;
    }
    await load(trackId, token);
  }

  Future<void> load(String trackId, String? token) async {
    track = null;
    isLoading = true;
    error = null;
    notifyListeners();

    try {
      final t = await _trackSvc.getTrack(trackId, token: token);
      await _audioHandler.loadTrack(t);

      track = t;
      duration = _audioHandler.duration;
      isLoading = false;
      notifyListeners();
    } catch (_) {
      isLoading = false;
      error = 'Could not load this track. Try again.';
      notifyListeners();
    }
  }

  void togglePlayPause() {
    if (_audioHandler.playing) {
      _audioHandler.pause();
    } else {
      _audioHandler.play();
    }
  }

  void seek(Duration to) => _audioHandler.seek(to);

  // Stops playback and clears the track — used when the mini player is
  // dismissed. Unlike dispose(), this notifier stays alive for the next track.
  Future<void> close() async {
    await _audioHandler.stop();
    track = null;
    isPlaying = false;
    position = Duration.zero;
    duration = Duration.zero;
    notifyListeners();
  }

  @override
  void dispose() {
    _stateSub?.cancel();
    _posSub?.cancel();
    _durSub?.cancel();
    super.dispose();
  }
}
