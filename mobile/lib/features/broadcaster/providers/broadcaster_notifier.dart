import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:record/record.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../../../core/api/api_client.dart';
import '../../player/models/stream_model.dart';
import '../services/broadcast_service.dart';

enum BroadcasterStatus { idle, creating, live, ending, error }

class BroadcasterNotifier extends ChangeNotifier {
  BroadcasterNotifier({ApiClient apiClient = const ApiClient()})
    : _apiClient = apiClient,
      _svc = BroadcastService(apiClient);

  final ApiClient _apiClient;
  final BroadcastService _svc;
  final AudioRecorder _recorder = AudioRecorder();
  WebSocketChannel? _channel;
  StreamSubscription<Uint8List>? _micSub;

  BroadcasterStatus status = BroadcasterStatus.idle;
  LiveStream? stream;
  String? error;

  Future<void> startBroadcast(
    String title,
    String? description,
    String token,
  ) async {
    if (status != BroadcasterStatus.idle && status != BroadcasterStatus.error) {
      return;
    }

    status = BroadcasterStatus.creating;
    error = null;
    notifyListeners();

    // Request mic permission before any network call.
    final micStatus = await Permission.microphone.request();
    if (!micStatus.isGranted) {
      status = BroadcasterStatus.error;
      error = 'Microphone permission denied.';
      notifyListeners();
      return;
    }

    try {
      // Create the stream record on the backend.
      final liveStream = await _svc.createStream(title, description, token);

      // Open WebSocket ingest connection.
      final wsUri = _apiClient.wsUri(
        '/streams/${liveStream.id}/ingest',
        token: token,
      );
      _channel = WebSocketChannel.connect(
        wsUri,
        protocols: const ['audio-ingest'],
      );
      await _channel!.ready.catchError((_) {});

      // Start mic capture with AAC-LC ADTS encoding.
      final micStream = await _recorder.startStream(
        const RecordConfig(
          encoder: AudioEncoder.aacLc,
          sampleRate: 44100,
          numChannels: 1,
          bitRate: 128000,
        ),
      );

      _micSub = micStream.listen(
        (chunk) => _channel?.sink.add(chunk),
        onDone: () => _onMicDone(token),
        onError: (_) => _onMicDone(token),
      );

      status = BroadcasterStatus.live;
      stream = liveStream;
      notifyListeners();
    } on ApiException catch (e) {
      status = BroadcasterStatus.error;
      error = e.message;
      notifyListeners();
      await _cleanup(null, token);
    } catch (e) {
      status = BroadcasterStatus.error;
      error = 'Failed to start broadcast. Try again.';
      notifyListeners();
      await _cleanup(null, token);
    }
  }

  Future<void> stopBroadcast(String token) async {
    if (status != BroadcasterStatus.live) {
      return;
    }
    status = BroadcasterStatus.ending;
    error = null;
    notifyListeners();

    final streamId = stream?.id;
    await _cleanup(streamId, token);
    status = BroadcasterStatus.idle;
    stream = null;
    error = null;
    notifyListeners();
  }

  void _onMicDone(String token) {
    _cleanup(null, token).then((_) {
      if (status != BroadcasterStatus.idle) {
        status = BroadcasterStatus.idle;
        stream = null;
        notifyListeners();
      }
    });
  }

  Future<void> _cleanup(String? streamId, String token) async {
    await _micSub?.cancel();
    _micSub = null;
    await _recorder.stop();

    // Close the WebSocket with a timeout — sink.close() can hang indefinitely
    // if the server never sends a close frame back (e.g. failed WS upgrade).
    final sinkClose = _channel?.sink.close();
    _channel = null;
    await sinkClose?.timeout(const Duration(seconds: 3), onTimeout: () {});

    if (streamId != null) {
      try {
        await _svc.endStream(streamId, token);
      } catch (_) {}
    }
  }

  // Called from dispose — must not throw.
  Future<void> _forceStop() async {
    await _micSub?.cancel();
    _micSub = null;
    await _recorder.stop();
    await _recorder.dispose();
    await _channel?.sink.close();
    _channel = null;
  }

  @override
  void dispose() {
    _forceStop();
    super.dispose();
  }
}
