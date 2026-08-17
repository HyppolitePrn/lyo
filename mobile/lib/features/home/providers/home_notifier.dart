import 'package:flutter/material.dart';

import '../../../core/api/api_client.dart';
import '../../player/models/stream_model.dart';
import '../../player/services/player_service.dart';
import '../models/home_models.dart';

class HomeNotifier extends ChangeNotifier {
  HomeNotifier({PlayerService? playerService})
      : _playerSvc = playerService ?? const PlayerService(ApiClient());

  final PlayerService _playerSvc;

  int selectedTab = 0;
  MiniPlayerState miniPlayer = const MiniPlayerState();
  Future<List<LiveStream>>? liveStreams;

  void refreshLiveStreams(String token) {
    liveStreams = _playerSvc.listLive(token);
    notifyListeners();
  }

  void switchTab(int index) {
    selectedTab = index;
    notifyListeners();
  }

  void openLiveStream(LiveStream stream) {
    miniPlayer = miniPlayer.copyWith(
      isVisible: true,
      isPlaying: true,
      trackTitle: stream.title,
      showName: stream.description?.isNotEmpty == true
          ? stream.description!
          : 'Live broadcast',
      artColor1: const Color(0xFF0D1F3D),
      artColor2: const Color(0xFF1B4F8A),
      type: PlayerType.live,
    );
    notifyListeners();
  }

  void openLivePlayer(LyoLiveShow show) {
    miniPlayer = miniPlayer.copyWith(
      isVisible: true,
      isPlaying: true,
      trackTitle: show.title,
      showName: show.host,
      artColor1: show.colors[0],
      artColor2: show.colors[1],
      type: PlayerType.live,
    );
    notifyListeners();
  }

  void openRecordedPlayer(LyoEpisode ep) {
    miniPlayer = miniPlayer.copyWith(
      isVisible: true,
      isPlaying: true,
      trackTitle: ep.title,
      showName: ep.show,
      artColor1: ep.colors[0],
      artColor2: ep.colors[1],
      type: PlayerType.recorded,
    );
    notifyListeners();
  }

  void togglePlayPause() {
    miniPlayer = miniPlayer.copyWith(isPlaying: !miniPlayer.isPlaying);
    notifyListeners();
  }

  void dismissMiniPlayer() {
    miniPlayer = miniPlayer.copyWith(isVisible: false, isPlaying: false);
    notifyListeners();
  }
}
