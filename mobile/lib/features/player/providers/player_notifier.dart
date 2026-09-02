// ignore_for_file: experimental_member_use

import 'dart:async';

import 'package:audio_session/audio_session.dart';
import 'package:flutter/foundation.dart';
import 'package:just_audio/just_audio.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../../../core/api/api_client.dart';
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
      offset: start ?? 0,
      stream: _stream,
      contentType: 'audio/aac',
    );
  }
}

enum PlayerStatus { idle, connecting, playing, error }

class PlayerNotifier extends ChangeNotifier {
  PlayerNotifier({ApiClient apiClient = const ApiClient()})
      : _apiClient = apiClient,
        _svc = PlayerService(apiClient);

  final ApiClient _apiClient;
  final PlayerService _svc;
  AudioPlayer? _player;
  WebSocketChannel? _channel;
  StreamController<Uint8List>? _byteController;
  StreamSubscription<dynamic>? _wsSub;

  PlayerStatus status = PlayerStatus.idle;
  LiveStream? stream;
  String? error;

  Future<void> connect(String streamId, String? token) async {
    if (status == PlayerStatus.connecting || status == PlayerStatus.playing) {
      return;
    }

    status = PlayerStatus.connecting;
    error = null;
    notifyListeners();

    try {
      final liveStream = await _svc.getStream(streamId, token ?? '');

      // Configure audio session for playback.
      final session = await AudioSession.instance;
      await session.configure(const AudioSessionConfiguration.music());

      // Open WebSocket.
      final wsUri = _apiClient.wsUri('/streams/$streamId/listen', token: token);
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

      // Transition to playing immediately — setAudioSource/play() can hang
      // waiting to buffer on a live stream, freezing the UI in "connecting".
      status = PlayerStatus.playing;
      stream = liveStream;
      notifyListeners();

      // Audio offload crashes ExoPlayer on some devices for this stream
      // format — IllegalArgumentException in DefaultAudioSink — so it's
      // disabled explicitly rather than left to the platform default.
      _player = AudioPlayer(
        androidAudioOffloadPreferences: const AndroidAudioOffloadPreferences(
          audioOffloadMode: AndroidAudioOffloadMode.disabled,
        ),
      );
      // Fire-and-forget: audio setup runs in the background without blocking state.
      _player!
          .setAudioSource(_WsAudioSource(_byteController!.stream))
          .then((_) => _player?.play())
          .catchError((Object e) {
        status = PlayerStatus.error;
        error = 'Playback failed. Try again.';
        notifyListeners();
        _cleanup();
      });
    } on ApiException catch (e) {
      status = PlayerStatus.error;
      error = e.message;
      notifyListeners();
      await _cleanup();
    } catch (e) {
      status = PlayerStatus.error;
      error = 'Connection failed. Try again.';
      notifyListeners();
      await _cleanup();
    }
  }

  void _onWsDone() {
    _cleanup().then((_) {
      if (status != PlayerStatus.idle) {
        status = PlayerStatus.idle;
        error = null;
        notifyListeners();
      }
    });
  }

  Future<void> disconnect() async {
    await _cleanup();
    status = PlayerStatus.idle;
    error = null;
    notifyListeners();
  }

  Future<void> _cleanup() async {
    await _wsSub?.cancel();
    _wsSub = null;
    final sinkClose = _channel?.sink.close();
    _channel = null;
    await sinkClose?.timeout(const Duration(seconds: 3), onTimeout: () {});
    await _byteController?.close();
    _byteController = null;
    await _player?.stop();
    await _player?.dispose();
    _player = null;
  }

  @override
  void dispose() {
    disconnect();
    super.dispose();
  }
}
