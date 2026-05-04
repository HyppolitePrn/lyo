// ignore_for_file: experimental_member_use

import 'dart:async';
import 'dart:typed_data';

import 'package:audio_session/audio_session.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:just_audio/just_audio.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../../../core/api/api_client.dart';
import '../../auth/providers/auth_notifier.dart';
import '../models/stream_model.dart';
import '../services/player_service.dart';

// ---------------------------------------------------------------------------
// Custom StreamAudioSource that feeds WebSocket binary frames to just_audio.
// ---------------------------------------------------------------------------
class _WsAudioSource extends StreamAudioSource {
  _WsAudioSource(this._stream);

  final Stream<Uint8List> _stream;

  @override
  Future<StreamAudioResponse> request([int? start, int? end]) async {
    return StreamAudioResponse(
      sourceLength: null,
      contentLength: null,
      offset: 0,
      stream: _stream,
      contentType: 'audio/aac',
    );
  }
}

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------
enum PlayerStatus { idle, connecting, playing, error }

class PlayerState {
  const PlayerState({
    this.status = PlayerStatus.idle,
    this.stream,
    this.error,
  });

  final PlayerStatus status;
  final LiveStream? stream;
  final String? error;

  PlayerState copyWith({
    PlayerStatus? status,
    LiveStream? stream,
    String? error,
    bool clearError = false,
  }) {
    return PlayerState(
      status: status ?? this.status,
      stream: stream ?? this.stream,
      error: clearError ? null : (error ?? this.error),
    );
  }
}

// ---------------------------------------------------------------------------
// Providers
// ---------------------------------------------------------------------------
final _playerServiceProvider = Provider<PlayerService>(
  (ref) => PlayerService(ref.watch(apiClientProvider)),
);

final playerNotifierProvider =
    NotifierProvider<PlayerNotifier, PlayerState>(PlayerNotifier.new);

// ---------------------------------------------------------------------------
// Notifier
// ---------------------------------------------------------------------------
class PlayerNotifier extends Notifier<PlayerState> {
  AudioPlayer? _player;
  WebSocketChannel? _channel;
  StreamController<Uint8List>? _byteController;
  StreamSubscription<dynamic>? _wsSub;

  @override
  PlayerState build() {
    ref.onDispose(disconnect);
    return const PlayerState();
  }

  PlayerService get _svc => ref.read(_playerServiceProvider);

  Future<void> connect(String streamId) async {
    if (state.status == PlayerStatus.connecting ||
        state.status == PlayerStatus.playing) {
      return;
    }

    state = state.copyWith(status: PlayerStatus.connecting, clearError: true);

    try {
      final token = ref.read(authNotifierProvider).accessToken;
      final liveStream = await _svc.getStream(streamId, token ?? '');

      // Configure audio session for playback.
      final session = await AudioSession.instance;
      await session.configure(const AudioSessionConfiguration.music());

      // Open WebSocket.
      final wsUri = ref
          .read(apiClientProvider)
          .wsUri('/streams/$streamId/listen', token: token);
      _channel = WebSocketChannel.connect(wsUri,
          protocols: const ['audio-stream']);
      await _channel!.ready.catchError((_) {});

      // Pipe binary frames into a broadcast stream controller.
      _byteController = StreamController<Uint8List>.broadcast();
      _wsSub = _channel!.stream.listen(
        (data) {
          if (data is Uint8List) {
            _byteController?.add(data);
          } else if (data is List<int>) {
            _byteController?.add(Uint8List.fromList(data));
          }
        },
        onDone: _onWsDone,
        onError: (_) => _onWsDone(),
      );

      // Start just_audio with the WebSocket source.
      _player = AudioPlayer();
      await _player!.setAudioSource(_WsAudioSource(_byteController!.stream));
      await _player!.play();

      state = state.copyWith(status: PlayerStatus.playing, stream: liveStream);
    } on ApiException catch (e) {
      state =
          state.copyWith(status: PlayerStatus.error, error: e.message);
      await _cleanup();
    } catch (e) {
      state = state.copyWith(
          status: PlayerStatus.error, error: 'Connection failed. Try again.');
      await _cleanup();
    }
  }

  void _onWsDone() {
    // Stream ended server-side — return to idle gracefully.
    _cleanup();
    state = state.copyWith(status: PlayerStatus.idle, clearError: true);
  }

  Future<void> disconnect() async {
    await _cleanup();
    state = state.copyWith(status: PlayerStatus.idle, clearError: true);
  }

  Future<void> _cleanup() async {
    await _wsSub?.cancel();
    _wsSub = null;
    await _channel?.sink.close();
    _channel = null;
    await _byteController?.close();
    _byteController = null;
    await _player?.stop();
    await _player?.dispose();
    _player = null;
  }
}
