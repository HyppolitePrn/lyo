import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:just_audio/just_audio.dart';

import '../../../core/api/api_client.dart';
import '../../track/models/track_model.dart';
import '../../track/services/track_service.dart';

// Plays a single track's static audio_url — distinct from PlayerNotifier,
// which pipes live WebSocket frames into just_audio instead.
class RecordedPlayerNotifier extends ChangeNotifier {
  RecordedPlayerNotifier({TrackService? trackService})
      : _trackSvc = trackService ?? const TrackService(ApiClient());

  final TrackService _trackSvc;
  final AudioPlayer _player = AudioPlayer();

  Track? track;
  String? error;
  bool isLoading = true;
  bool isPlaying = false;
  Duration position = Duration.zero;
  Duration duration = Duration.zero;

  StreamSubscription<PlayerState>? _stateSub;
  StreamSubscription<Duration>? _posSub;
  StreamSubscription<Duration?>? _durSub;

  Future<void> load(String trackId, String? token) async {
    isLoading = true;
    error = null;
    notifyListeners();

    try {
      final t = await _trackSvc.getTrack(trackId, token: token);
      final d = await _player.setUrl(t.audioUrl);

      track = t;
      duration = d ?? Duration.zero;
      isLoading = false;
      notifyListeners();

      _stateSub = _player.playerStateStream.listen((s) {
        isPlaying = s.playing;
        notifyListeners();
      });
      _posSub = _player.positionStream.listen((p) {
        position = p;
        notifyListeners();
      });
      _durSub = _player.durationStream.listen((d) {
        duration = d ?? Duration.zero;
        notifyListeners();
      });

      await _player.play();
    } catch (_) {
      isLoading = false;
      error = 'Could not load this track. Try again.';
      notifyListeners();
    }
  }

  void togglePlayPause() {
    if (_player.playing) {
      _player.pause();
    } else {
      _player.play();
    }
  }

  void seek(Duration to) => _player.seek(to);

  @override
  void dispose() {
    _stateSub?.cancel();
    _posSub?.cancel();
    _durSub?.cancel();
    _player.dispose();
    super.dispose();
  }
}
