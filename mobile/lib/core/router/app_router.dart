import 'package:go_router/go_router.dart';

import '../../features/admin/screens/supervision_screen.dart';
import '../../features/auth/screens/forgot_password_screen.dart';
import '../../features/auth/screens/login_screen.dart';
import '../../features/auth/screens/register_screen.dart';
import '../../features/auth/screens/reset_password_screen.dart';
import '../../features/auth/screens/splash_screen.dart';
import '../../features/broadcaster/screens/broadcaster_screen.dart';
import '../../features/favorites/screens/favorites_screen.dart';
import '../../features/home/screens/home_screen.dart';
import '../../features/player/screens/live_player_screen.dart';
import '../../features/player/screens/player_screen.dart';
import '../../features/player/screens/recorded_player_screen.dart';
import '../../features/player/screens/stream_list_screen.dart';
import '../../features/playlist/screens/playlist_detail_screen.dart';
import '../../features/playlist/screens/playlists_screen.dart';
import '../../features/profile/screens/profile_screen.dart';
import '../../features/track/screens/upload_track_screen.dart';

/// Builds the app router.
///
/// [initialLocation] lets `main` skip the splash screen when a persisted
/// session was restored, so a returning user lands straight on Home.
GoRouter createAppRouter({String initialLocation = '/splash'}) => GoRouter(
  initialLocation: initialLocation,
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
      path: '/forgot-password',
      builder: (context, state) => const ForgotPasswordScreen(),
    ),
    GoRoute(
      path: '/reset-password',
      builder: (context, state) =>
          ResetPasswordScreen(token: state.uri.queryParameters['token']),
    ),
    GoRoute(
      path: '/home',
      builder: (context, state) => const HomeScreen(),
    ),
    GoRoute(
      path: '/profile',
      builder: (context, state) => const ProfileScreen(),
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
    GoRoute(
      path: '/upload-track',
      builder: (context, state) => const UploadTrackScreen(),
    ),
    GoRoute(
      path: '/playlists',
      builder: (context, state) => const PlaylistsScreen(),
    ),
    GoRoute(
      path: '/playlists/:id',
      builder: (context, state) =>
          PlaylistDetailScreen(playlistId: state.pathParameters['id']!),
    ),
    GoRoute(
      path: '/favorites',
      builder: (context, state) => const FavoritesScreen(),
    ),
    // Admin supervision. The screen re-checks the role, and the backend
    // rejects every request below admin regardless.
    GoRoute(
      path: '/admin/supervision',
      builder: (context, state) => const SupervisionScreen(),
    ),
  ],
);
