import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:record/record.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../../../core/api/api_client.dart';
import '../../auth/providers/auth_notifier.dart';
import '../../player/models/stream_model.dart';
import '../services/broadcast_service.dart';

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------
enum BroadcasterStatus { idle, creating, live, ending, error }

class BroadcasterState {
  const BroadcasterState({
    this.status = BroadcasterStatus.idle,
    this.stream,
    this.error,
  });

  final BroadcasterStatus status;
  final LiveStream? stream;
  final String? error;

  BroadcasterState copyWith({
    BroadcasterStatus? status,
    LiveStream? stream,
    String? error,
    bool clearError = false,
    bool clearStream = false,
  }) {
    return BroadcasterState(
      status: status ?? this.status,
      stream: clearStream ? null : (stream ?? this.stream),
      error: clearError ? null : (error ?? this.error),
    );
  }
}

// ---------------------------------------------------------------------------
// Providers
// ---------------------------------------------------------------------------
final _broadcastServiceProvider = Provider<BroadcastService>(
  (ref) => BroadcastService(ref.watch(apiClientProvider)),
);

final broadcasterNotifierProvider =
    NotifierProvider<BroadcasterNotifier, BroadcasterState>(
        BroadcasterNotifier.new);

// ---------------------------------------------------------------------------
// Notifier
// ---------------------------------------------------------------------------
class BroadcasterNotifier extends Notifier<BroadcasterState> {
  final AudioRecorder _recorder = AudioRecorder();
  WebSocketChannel? _channel;
  StreamSubscription<Uint8List>? _micSub;

  @override
  BroadcasterState build() {
    ref.onDispose(_forceStop);
    return const BroadcasterState();
  }

  BroadcastService get _svc => ref.read(_broadcastServiceProvider);

  Future<void> startBroadcast(String title, String? description) async {
    if (state.status != BroadcasterStatus.idle &&
        state.status != BroadcasterStatus.error) {
      return;
    }

    state =
        state.copyWith(status: BroadcasterStatus.creating, clearError: true);

    // Request mic permission before any network call.
    final micStatus = await Permission.microphone.request();
    if (!micStatus.isGranted) {
      state = state.copyWith(
        status: BroadcasterStatus.error,
        error: 'Microphone permission denied.',
      );
      return;
    }

    try {
      final token = ref.read(authNotifierProvider).accessToken ?? '';

      // Create the stream record on the backend.
      final liveStream = await _svc.createStream(title, description, token);

      // Open WebSocket ingest connection.
      final wsUri = ref
          .read(apiClientProvider)
          .wsUri('/streams/${liveStream.id}/ingest', token: token);
      _channel = WebSocketChannel.connect(wsUri,
          protocols: const ['audio-ingest']);
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
        onDone: _onMicDone,
        onError: (_) => _onMicDone(),
      );

      state = state.copyWith(
          status: BroadcasterStatus.live, stream: liveStream);
    } on ApiException catch (e) {
      state = state.copyWith(
          status: BroadcasterStatus.error, error: e.message);
      await _cleanup(null);
    } catch (e) {
      state = state.copyWith(
        status: BroadcasterStatus.error,
        error: 'Failed to start broadcast. Try again.',
      );
      await _cleanup(null);
    }
  }

  Future<void> stopBroadcast() async {
    if (state.status != BroadcasterStatus.live) return;
    state = state.copyWith(status: BroadcasterStatus.ending, clearError: true);

    final streamId = state.stream?.id;
    await _cleanup(streamId);
    state = state.copyWith(
        status: BroadcasterStatus.idle, clearStream: true, clearError: true);
  }

  void _onMicDone() {
    _cleanup(null);
    state = state.copyWith(status: BroadcasterStatus.idle, clearStream: true);
  }

  Future<void> _cleanup(String? streamId) async {
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
        final token = ref.read(authNotifierProvider).accessToken ?? '';
        await _svc.endStream(streamId, token);
      } catch (_) {}
    }
  }

  // Called by onDispose — must not throw.
  Future<void> _forceStop() async {
    await _micSub?.cancel();
    _micSub = null;
    await _recorder.stop();
    await _recorder.dispose();
    await _channel?.sink.close();
    _channel = null;
  }
}
