import 'dart:async';

import 'package:audio_service/audio_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:provider/provider.dart';

import 'core/deep_links/deep_link_listener.dart';
import 'core/features/feature_flags_provider.dart';
import 'core/router/app_router.dart';
import 'core/theme/lyo_theme.dart';
import 'features/auth/providers/auth_notifier.dart';
import 'features/broadcaster/providers/broadcaster_notifier.dart';
import 'features/favorites/providers/favorites_notifier.dart';
import 'features/home/providers/home_notifier.dart';
import 'features/player/providers/player_notifier.dart';
import 'features/player/providers/recorded_player_notifier.dart';
import 'features/player/services/lyo_audio_handler.dart';
import 'features/playlist/providers/playlist_notifier.dart';
import 'features/track/providers/upload_track_notifier.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Lyo is portrait-only: every screen (player, broadcaster, lists) is laid out
  // as a single column, and landscape would only stretch it.
  await SystemChrome.setPreferredOrientations([
    DeviceOrientation.portraitUp,
    DeviceOrientation.portraitDown,
  ]);

  // Android 13+ hides the playback notification without this — the service
  // still plays in the background either way, this only affects visibility.
  unawaited(Permission.notification.request());

  // Lets a track keep playing — with a Spotify-style notification and lock
  // screen controls — after the app is backgrounded.
  final audioHandler = await AudioService.init(
    builder: LyoAudioHandler.new,
    config: const AudioServiceConfig(
      androidNotificationChannelId: 'com.lyo.mobile.audio',
      androidNotificationChannelName: 'Lyo playback',
      androidNotificationOngoing: false,
      androidStopForegroundOnPause: true,
      fastForwardInterval: Duration(seconds: 15),
      rewindInterval: Duration(seconds: 15),
    ),
  );

  runApp(
    MultiProvider(
      providers: [
        Provider<FeatureFlags>(create: (_) => const FeatureFlags()),
        ChangeNotifierProvider(create: (_) => AuthNotifier()),
        ChangeNotifierProvider(create: (_) => HomeNotifier()),
        ChangeNotifierProvider(create: (_) => PlayerNotifier()),
        ChangeNotifierProvider(
          create: (_) => RecordedPlayerNotifier(audioHandler: audioHandler),
        ),
        ChangeNotifierProvider(create: (_) => BroadcasterNotifier()),
        ChangeNotifierProvider(create: (_) => UploadTrackNotifier()),
        ChangeNotifierProvider(create: (_) => PlaylistNotifier()),
        ChangeNotifierProvider(create: (_) => FavoritesNotifier()),
      ],
      child: const LyoApp(),
    ),
  );
  await DeepLinkListener(appRouter).init();
}

class LyoApp extends StatelessWidget {
  const LyoApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Lyo',
      theme: lyoTheme(Brightness.dark),
      darkTheme: lyoTheme(Brightness.dark),
      themeMode: ThemeMode.dark,
      routerConfig: appRouter,
      debugShowCheckedModeBanner: false,
    );
  }
}
