import 'dart:async';

import 'package:audio_service/audio_service.dart';
import 'package:just_audio/just_audio.dart';

import '../../track/models/track_model.dart';

// Wraps a single just_audio player and exposes it through audio_service, so
// the OS media notification / lock screen / headset buttons can control the
// currently playing track even after the app is backgrounded — similar to
// Spotify's mini controls outside the app.
class LyoAudioHandler extends BaseAudioHandler with SeekHandler {
  LyoAudioHandler()
    : _player = AudioPlayer(
        // Audio offload crashes ExoPlayer on some devices for this track
        // format — IllegalArgumentException in DefaultAudioSink.
        androidAudioOffloadPreferences: const AndroidAudioOffloadPreferences(
          audioOffloadMode: AndroidAudioOffloadMode.disabled,
        ),
      ) {
    _player.playbackEventStream.listen(_broadcastState, onError: (Object _) {});
  }

  final AudioPlayer _player;

  Track? track;
  bool get playing => _player.playing;
  Duration get position => _player.position;
  Duration get duration => _player.duration ?? Duration.zero;
  Stream<Duration> get positionStream => _player.positionStream;
  Stream<Duration?> get durationStream => _player.durationStream;

  Future<void> loadTrack(Track t) async {
    track = t;
    mediaItem.add(
      MediaItem(
        id: t.id,
        title: t.title,
        artist: t.artist?.isNotEmpty == true ? t.artist : 'Unknown artist',
        duration: Duration(seconds: t.durationSeconds),
      ),
    );
    await _player.setUrl(t.audioUrl);
    // just_audio's play() Future only resolves once playback stops, not once
    // it starts — awaiting it here would leave callers of loadTrack() (and
    // the loading spinner tied to them) hanging for the entire track.
    unawaited(play());
  }

  void _broadcastState(PlaybackEvent event) {
    final playing = _player.playing;
    playbackState.add(
      playbackState.value.copyWith(
        controls: [
          MediaControl.rewind,
          playing ? MediaControl.pause : MediaControl.play,
          MediaControl.fastForward,
        ],
        systemActions: const {
          MediaAction.seek,
          MediaAction.seekForward,
          MediaAction.seekBackward,
        },
        androidCompactActionIndices: const [0, 1, 2],
        processingState: switch (_player.processingState) {
          ProcessingState.idle => AudioProcessingState.idle,
          ProcessingState.loading => AudioProcessingState.loading,
          ProcessingState.buffering => AudioProcessingState.buffering,
          ProcessingState.ready => AudioProcessingState.ready,
          ProcessingState.completed => AudioProcessingState.completed,
        },
        playing: playing,
        updatePosition: _player.position,
        bufferedPosition: _player.bufferedPosition,
        speed: _player.speed,
        queueIndex: 0,
      ),
    );
  }

  @override
  Future<void> play() => _player.play();

  @override
  Future<void> pause() => _player.pause();

  @override
  Future<void> seek(Duration position) => _player.seek(position);

  @override
  Future<void> stop() async {
    await _player.stop();
    track = null;
    mediaItem.add(null);
    // idle deactivates the notification and stops the foreground service.
    playbackState.add(
      playbackState.value.copyWith(
        processingState: AudioProcessingState.idle,
        playing: false,
      ),
    );
  }

  // Only fires when the app is swiped away from the recent-apps list, not
  // when it's simply backgrounded (e.g. home button) — that's the signal to
  // cut playback and dismiss the notification, matching the requested UX.
  @override
  Future<void> onTaskRemoved() async {
    await stop();
  }
}
