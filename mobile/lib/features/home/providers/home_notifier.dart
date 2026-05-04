import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/home_models.dart';

final homeNotifierProvider =
    NotifierProvider<HomeNotifier, HomeState>(HomeNotifier.new);

class HomeNotifier extends Notifier<HomeState> {
  @override
  HomeState build() => const HomeState();

  void switchTab(int index) => state = state.copyWith(selectedTab: index);

  void openLivePlayer(LyoLiveShow show) {
    state = state.copyWith(
      miniPlayer: state.miniPlayer.copyWith(
        isVisible: true,
        isPlaying: true,
        trackTitle: show.title,
        showName: show.host,
        artColor1: show.colors[0],
        artColor2: show.colors[1],
        type: PlayerType.live,
      ),
    );
  }

  void openRecordedPlayer(LyoEpisode ep) {
    state = state.copyWith(
      miniPlayer: state.miniPlayer.copyWith(
        isVisible: true,
        isPlaying: true,
        trackTitle: ep.title,
        showName: ep.show,
        artColor1: ep.colors[0],
        artColor2: ep.colors[1],
        type: PlayerType.recorded,
      ),
    );
  }

  void togglePlayPause() {
    state = state.copyWith(
      miniPlayer:
          state.miniPlayer.copyWith(isPlaying: !state.miniPlayer.isPlaying),
    );
  }

  void dismissMiniPlayer() {
    state = state.copyWith(
      miniPlayer:
          state.miniPlayer.copyWith(isVisible: false, isPlaying: false),
    );
  }
}
