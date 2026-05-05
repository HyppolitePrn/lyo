import 'package:flutter/material.dart';

class LyoLiveShow {
  const LyoLiveShow({
    required this.id,
    required this.title,
    required this.host,
    required this.listeners,
    required this.colors,
  });
  final String id;
  final String title;
  final String host;
  final int listeners;
  final List<Color> colors;
}

class LyoEpisode {
  const LyoEpisode({
    required this.id,
    required this.title,
    required this.show,
    required this.duration,
    required this.colors,
  });
  final String id;
  final String title;
  final String show;
  final String duration;
  final List<Color> colors;
}

class LyoCategory {
  const LyoCategory({
    required this.label,
    required this.icon,
    required this.colors,
  });
  final String label;
  final IconData icon;
  final List<Color> colors;
}

enum PlayerType { live, recorded }

class MiniPlayerState {
  const MiniPlayerState({
    this.isVisible = false,
    this.isPlaying = false,
    this.trackTitle = '',
    this.showName = '',
    this.artColor1 = const Color(0xFF1A1A1A),
    this.artColor2 = const Color(0xFF3A3A3A),
    this.type = PlayerType.live,
  });

  final bool isVisible;
  final bool isPlaying;
  final String trackTitle;
  final String showName;
  final Color artColor1;
  final Color artColor2;
  final PlayerType type;

  MiniPlayerState copyWith({
    bool? isVisible,
    bool? isPlaying,
    String? trackTitle,
    String? showName,
    Color? artColor1,
    Color? artColor2,
    PlayerType? type,
  }) =>
      MiniPlayerState(
        isVisible: isVisible ?? this.isVisible,
        isPlaying: isPlaying ?? this.isPlaying,
        trackTitle: trackTitle ?? this.trackTitle,
        showName: showName ?? this.showName,
        artColor1: artColor1 ?? this.artColor1,
        artColor2: artColor2 ?? this.artColor2,
        type: type ?? this.type,
      );
}

class HomeState {
  const HomeState({
    this.selectedTab = 0,
    this.miniPlayer = const MiniPlayerState(),
  });

  final int selectedTab;
  final MiniPlayerState miniPlayer;

  HomeState copyWith({
    int? selectedTab,
    MiniPlayerState? miniPlayer,
  }) =>
      HomeState(
        selectedTab: selectedTab ?? this.selectedTab,
        miniPlayer: miniPlayer ?? this.miniPlayer,
      );
}