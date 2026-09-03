import 'dart:async';

import 'package:audio_service/audio_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
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
import 'features/player/services/volume_controller.dart';
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

  // One volume for the whole app: the live player and the recorded player
  // each own an AudioPlayer, and both follow this controller.
  final volume = VolumeController();

  // Lets a track keep playing — with a Spotify-style notification and lock
  // screen controls — after the app is backgrounded.
  final audioHandler = await AudioService.init(
    builder: () => LyoAudioHandler(volume: volume),
    config: const AudioServiceConfig(
      androidNotificationChannelId: 'com.lyo.mobile.audio',
      androidNotificationChannelName: 'Lyo playback',
      androidNotificationOngoing: false,
      androidStopForegroundOnPause: true,
      fastForwardInterval: Duration(seconds: 15),
      rewindInterval: Duration(seconds: 15),
    ),
  );

  // Reload a session persisted by a previous launch before the first frame,
  // so a returning user never sees the splash screen flash by.
  final auth = AuthNotifier();
  final restored = await auth.restoreSession();
  final router = createAppRouter(
    initialLocation: restored ? '/home' : '/splash',
  );

  runApp(
    MultiProvider(
      providers: [
        Provider<FeatureFlags>(create: (_) => const FeatureFlags()),
        ChangeNotifierProvider.value(value: auth),
        ChangeNotifierProvider(create: (_) => HomeNotifier()),
        ChangeNotifierProvider.value(value: volume),
        ChangeNotifierProvider(create: (_) => PlayerNotifier(volume: volume)),
        ChangeNotifierProvider(
          create: (_) => RecordedPlayerNotifier(audioHandler: audioHandler),
        ),
        ChangeNotifierProvider(create: (_) => BroadcasterNotifier()),
        ChangeNotifierProvider(create: (_) => UploadTrackNotifier()),
        ChangeNotifierProvider(create: (_) => PlaylistNotifier()),
        ChangeNotifierProvider(create: (_) => FavoritesNotifier()),
      ],
      child: LyoApp(router: router),
    ),
  );
  await DeepLinkListener(router).init();
}

class LyoApp extends StatefulWidget {
  const LyoApp({required this.router, super.key});

  final GoRouter router;

  @override
  State<LyoApp> createState() => _LyoAppState();
}

class _LyoAppState extends State<LyoApp> with WidgetsBindingObserver {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // The in-app refresh timer does not fire while the process is suspended,
    // so re-check the access token every time the app comes back to the
    // foreground.
    if (state == AppLifecycleState.resumed) {
      unawaited(context.read<AuthNotifier>().ensureFreshSession());
    }
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Lyo',
      theme: lyoTheme(Brightness.dark),
      darkTheme: lyoTheme(Brightness.dark),
      themeMode: ThemeMode.dark,
      routerConfig: widget.router,
      debugShowCheckedModeBanner: false,
    );
  }
}
