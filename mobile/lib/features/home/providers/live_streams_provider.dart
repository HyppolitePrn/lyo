import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../auth/providers/auth_notifier.dart';
import '../../player/models/stream_model.dart';
import '../../player/services/player_service.dart';

final liveStreamsProvider = FutureProvider.autoDispose<List<LiveStream>>((ref) {
  final token = ref.watch(authNotifierProvider).accessToken ?? '';
  final svc = PlayerService(ref.watch(apiClientProvider));
  return svc.listLive(token);
});
