import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/home_models.dart';
import '../../player/models/stream_model.dart';

final homeNotifierProvider =
    NotifierProvider<HomeNotifier, HomeState>(HomeNotifier.new);

class HomeNotifier extends Notifier<HomeState> {
  @override
  HomeState build() => const HomeState();

  void switchTab(int index) => state = state.copyWith(selectedTab: index);

  void openLiveStream(LiveStream stream) {
    state = state.copyWith(
      miniPlayer: state.miniPlayer.copyWith(
        isVisible: true,
        isPlaying: true,
        trackTitle: stream.title,
        showName: stream.description?.isNotEmpty == true
            ? stream.description!
            : 'Live broadcast',
        artColor1: const Color(0xFF0D1F3D),
        artColor2: const Color(0xFF1B4F8A),
        type: PlayerType.live,
      ),
    );
  }

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
