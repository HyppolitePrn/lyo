import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/screens/login_screen.dart';
import '../../features/auth/screens/register_screen.dart';
import '../../features/auth/screens/splash_screen.dart';
import '../../features/broadcaster/screens/broadcaster_screen.dart';
import '../../features/home/screens/home_screen.dart';
import '../../features/player/screens/live_player_screen.dart';
import '../../features/player/screens/player_screen.dart';
import '../../features/player/screens/recorded_player_screen.dart';
import '../../features/player/screens/stream_list_screen.dart';

final routerProvider = Provider<GoRouter>((_) {
  return GoRouter(
    initialLocation: '/splash',
    routes: [
      GoRoute(
        path: '/splash',
        builder: (context, state) => const SplashScreen(),
      ),
      GoRoute(
        path: '/login',
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: '/register',
        builder: (context, state) => const RegisterScreen(),
      ),
      GoRoute(
        path: '/home',
        builder: (context, state) => const HomeScreen(),
      ),
      // Live stream list (full API-backed screen)
      GoRoute(
        path: '/streams',
        builder: (context, state) => const StreamListScreen(),
      ),
      // WebSocket live player (API-backed)
      GoRoute(
        path: '/player/:id',
        builder: (context, state) =>
            PlayerScreen(streamId: state.pathParameters['id']!),
      ),
      // Stub player screens (navigated from Home mock data)
      GoRoute(
        path: '/live-player/:id',
        builder: (context, state) =>
            LivePlayerScreen(showId: state.pathParameters['id']!),
      ),
      GoRoute(
        path: '/recorded-player/:id',
        builder: (context, state) =>
            RecordedPlayerScreen(episodeId: state.pathParameters['id']!),
      ),
      // Broadcaster screen (role-gated via FAB visibility)
      GoRoute(
        path: '/broadcaster',
        builder: (context, state) => const BroadcasterScreen(),
      ),
    ],
  );
});
