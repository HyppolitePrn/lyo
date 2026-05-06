import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';
import '../../player/providers/player_notifier.dart';
import '../models/home_models.dart';
import '../providers/home_notifier.dart';
import '../providers/live_streams_provider.dart';
import '../widgets/live_eq_widget.dart';
import '../widgets/lyo_artwork_tile.dart';
import '../widgets/mini_player.dart';

// ── Helpers ───────────────────────────────────────────────────────────────────

const _streamPalettes = [
  [Color(0xFF0D1F3D), Color(0xFF1B4F8A)],
  [Color(0xFF1A0D3D), Color(0xFF4A2090)],
  [Color(0xFF1A2A0D), Color(0xFF3D6B1A)],
  [Color(0xFF3D1A0A), Color(0xFF6B3320)],
  [Color(0xFF0A1A3D), Color(0xFF203D7A)],
];

List<Color> _streamColors(String id) {
  final idx = id.codeUnits.fold(0, (a, b) => a + b) % _streamPalettes.length;
  return _streamPalettes[idx];
}

const _episodes = [
  LyoEpisode(
    id: 'ep-1',
    title: 'The Science of Sleep',
    show: 'Mind & Body Radio',
    duration: '38 min',
    colors: [Color(0xFF3D1A0A), Color(0xFF6B3320)],
  ),
  LyoEpisode(
    id: 'ep-2',
    title: 'Building with AI',
    show: 'Dev Insider',
    duration: '52 min',
    colors: [Color(0xFF0A1A3D), Color(0xFF203D7A)],
  ),
  LyoEpisode(
    id: 'ep-3',
    title: 'Stoic Living',
    show: 'Philosophy Hour',
    duration: '41 min',
    colors: [Color(0xFF1A1A0A), Color(0xFF4A4A1A)],
  ),
  LyoEpisode(
    id: 'ep-4',
    title: 'Ocean Sounds',
    show: 'Ambient Zone',
    duration: '60 min',
    colors: [Color(0xFF0A2A3D), Color(0xFF1A5A6B)],
  ),
];

const _categories = [
  LyoCategory(
    label: 'News & Politics',
    icon: Icons.radio,
    colors: [Color(0xFF1A0D0D), Color(0xFF4A1A1A)],
  ),
  LyoCategory(
    label: 'Technology',
    icon: Icons.mic_none,
    colors: [Color(0xFF0D1A1A), Color(0xFF1A4A4A)],
  ),
  LyoCategory(
    label: 'Health & Mind',
    icon: Icons.favorite_border,
    colors: [Color(0xFF0D1A0D), Color(0xFF1A4A1A)],
  ),
  LyoCategory(
    label: 'Music & Arts',
    icon: Icons.queue_music,
    colors: [Color(0xFF1A0D2A), Color(0xFF3A1A5A)],
  ),
];

// ── Helpers ───────────────────────────────────────────────────────────────────

String _greeting() {
  final h = DateTime.now().hour;
  if (h < 12) {
    return 'Good morning';
  }
  if (h < 17) {
    return 'Good afternoon';
  }
  if (h < 22) {
    return 'Good evening';
  }
  return 'Good night';
}

String _fmt(int n) => n >= 1000 ? '${(n / 1000).toStringAsFixed(1)}k' : '$n';

// ── Screen ────────────────────────────────────────────────────────────────────

class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final home = ref.watch(homeNotifierProvider);
    final auth = ref.watch(authNotifierProvider);
    final notifier = ref.read(homeNotifierProvider.notifier);

    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final navBg = dark ? lyoNavDark : lyoNavLight;
    final border = dark ? lyoBorderDark : lyoBorderLight;

    return Scaffold(
      backgroundColor: bg,
      body: Stack(
        children: [
          _buildTabBody(context, ref, home, notifier, dark),
          if (home.miniPlayer.isVisible)
            Positioned(
              bottom: 8,
              left: 12,
              right: 12,
              child: MiniPlayer(
                state: home.miniPlayer,
                onTap: () {
                  if (home.miniPlayer.type == PlayerType.live) {
                    context.push('/live-player/current');
                  } else {
                    context.push('/recorded-player/current');
                  }
                },
                onToggle: notifier.togglePlayPause,
                onDismiss: () {
                  ref.read(playerNotifierProvider.notifier).disconnect();
                  notifier.dismissMiniPlayer();
                },
              ),
            ),
        ],
      ),
      bottomNavigationBar: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(height: 1, color: border),
          BottomNavigationBar(
            currentIndex: home.selectedTab,
            onTap: notifier.switchTab,
            type: BottomNavigationBarType.fixed,
            backgroundColor: navBg,
            selectedItemColor: lyoAccent,
            unselectedItemColor: dark ? lyoSubDark : lyoSubLight,
            selectedFontSize: 10,
            unselectedFontSize: 10,
            elevation: 0,
            selectedLabelStyle:
                const TextStyle(fontWeight: FontWeight.w700),
            unselectedLabelStyle:
                const TextStyle(fontWeight: FontWeight.w500),
            items: const [
              BottomNavigationBarItem(
                icon: Icon(Icons.home_outlined),
                activeIcon: Icon(Icons.home),
                label: 'Home',
              ),
              BottomNavigationBarItem(
                icon: Icon(Icons.grid_view_outlined),
                activeIcon: Icon(Icons.grid_view),
                label: 'Browse',
              ),
              BottomNavigationBarItem(
                icon: Icon(Icons.search),
                label: 'Search',
              ),
              BottomNavigationBarItem(
                icon: Icon(Icons.person_outline),
                activeIcon: Icon(Icons.person),
                label: 'Profile',
              ),
            ],
          ),
        ],
      ),
      floatingActionButton: auth.isBroadcaster
          ? FloatingActionButton(
              backgroundColor: lyoAccent,
              foregroundColor: Colors.white,
              tooltip: 'Go Live',
              onPressed: () => context.push('/broadcaster'),
              child: const Icon(Icons.mic),
            )
          : null,
    );
  }

  Widget _buildTabBody(
    BuildContext context,
    WidgetRef ref,
    HomeState home,
    HomeNotifier notifier,
    bool dark,
  ) {
    switch (home.selectedTab) {
      case 1:
        return _StubBody(
            label: 'Browse', icon: Icons.grid_view, dark: dark);
      case 2:
        return _StubBody(
            label: 'Search', icon: Icons.search, dark: dark);
      case 3:
        return _StubBody(
            label: 'Profile', icon: Icons.person, dark: dark);
      default:
        return _HomeBody(dark: dark);
    }
  }
}

// ── Home tab body ─────────────────────────────────────────────────────────────

class _HomeBody extends ConsumerWidget {
  const _HomeBody({required this.dark});
  final bool dark;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return SafeArea(
      child: CustomScrollView(
        slivers: [
          SliverToBoxAdapter(child: _buildHeader(context)),
          SliverToBoxAdapter(child: _buildLiveNow(context, ref)),
          SliverToBoxAdapter(child: _buildRecentEpisodes(context, ref)),
          SliverToBoxAdapter(child: _buildExplore(context)),
          const SliverToBoxAdapter(child: SizedBox(height: 100)),
        ],
      ),
    );
  }

  Widget _buildHeader(BuildContext context) {
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final surface = dark ? lyoSurfaceDark : lyoSurfaceLight;

    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                _greeting(),
                style: TextStyle(
                  fontSize: lyoCaption,
                  color: textSub,
                  fontWeight: FontWeight.w500,
                ),
              ),
              const Text(
                'lyo',
                style: TextStyle(
                  fontSize: lyoH1,
                  fontWeight: FontWeight.w800,
                  letterSpacing: -0.4,
                  color: lyoAccent,
                ),
              ),
            ],
          ),
          Row(
            children: [
              Container(
                width: 38,
                height: 38,
                decoration: BoxDecoration(
                  color: surface,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Icon(Icons.notifications_none,
                    size: 18, color: dark ? lyoSubDark : lyoSubLight),
              ),
              const SizedBox(width: lyoGapS),
              Container(
                width: 38,
                height: 38,
                decoration: BoxDecoration(
                  color: lyoAccent,
                  borderRadius: BorderRadius.circular(19),
                ),
                alignment: Alignment.center,
                child: const Text(
                  'JD',
                  style: TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                    color: Colors.white,
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildLiveNow(BuildContext context, WidgetRef ref) {
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final notifier = ref.read(homeNotifierProvider.notifier);
    final asyncStreams = ref.watch(liveStreamsProvider);

    Widget listContent = asyncStreams.when(
      loading: () => const Center(
        child: CircularProgressIndicator(color: lyoAccent, strokeWidth: 2),
      ),
      error: (e, _) => Center(
        child: Text(
          'Could not load streams',
          style: TextStyle(color: textSub, fontSize: lyoCaption),
        ),
      ),
      data: (streams) {
        if (streams.isEmpty) {
          return Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.radio,
                    size: 32, color: lyoAccent.withValues(alpha: 0.35)),
                const SizedBox(height: 8),
                Text(
                  'No live streams right now',
                  style: TextStyle(color: textSub, fontSize: lyoCaption),
                ),
              ],
            ),
          );
        }
        return ListView.separated(
          scrollDirection: Axis.horizontal,
          padding: const EdgeInsets.symmetric(horizontal: 20),
          itemCount: streams.length,
          separatorBuilder: (context, index) => const SizedBox(width: 12),
          itemBuilder: (context, i) {
            final s = streams[i];
            final show = LyoLiveShow(
              id: s.id,
              title: s.title,
              host: s.description?.isNotEmpty == true
                  ? s.description!
                  : 'Live broadcast',
              listeners: 0,
              colors: _streamColors(s.id),
            );
            return _LiveShowCard(
              show: show,
              onTap: () {
                notifier.openLiveStream(s);
                context.push('/player/${s.id}');
              },
            );
          },
        );
      },
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 20, 8, 14),
          child: Row(
            children: [
              Container(
                width: 8,
                height: 8,
                decoration: BoxDecoration(
                  color: lyoAccent,
                  borderRadius: BorderRadius.circular(4),
                  boxShadow: const [
                    BoxShadow(
                      color: Color(0x66E8B84A),
                      blurRadius: 6,
                      spreadRadius: 2,
                    ),
                  ],
                ),
              ),
              const SizedBox(width: lyoGapS),
              Text(
                'Live Now',
                style: TextStyle(
                  fontSize: lyoH2,
                  fontWeight: FontWeight.w700,
                  color: textPrimary,
                ),
              ),
              const Spacer(),
              IconButton(
                icon: Icon(Icons.refresh,
                    size: 18, color: dark ? lyoSubDark : lyoSubLight),
                tooltip: 'Refresh',
                visualDensity: VisualDensity.compact,
                onPressed: () => ref.invalidate(liveStreamsProvider),
              ),
            ],
          ),
        ),
        SizedBox(height: 170, child: listContent),
      ],
    );
  }

  Widget _buildRecentEpisodes(BuildContext context, WidgetRef ref) {
    final notifier = ref.read(homeNotifierProvider.notifier);
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final border = dark ? lyoBorderDark : lyoBorderLight;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 24, 20, 14),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                'Recent Episodes',
                style: TextStyle(
                  fontSize: lyoH2,
                  fontWeight: FontWeight.w700,
                  color: textPrimary,
                ),
              ),
              TextButton(
                onPressed: () {},
                style: TextButton.styleFrom(
                  foregroundColor: lyoAccent,
                  padding: EdgeInsets.zero,
                  minimumSize: Size.zero,
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                ),
                child: const Text(
                  'See all',
                  style: TextStyle(
                    fontSize: lyoCaption,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ],
          ),
        ),
        ListView.separated(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          padding: const EdgeInsets.symmetric(horizontal: 20),
          itemCount: _episodes.length,
          separatorBuilder: (_, _) =>
              Divider(color: border, height: 1),
          itemBuilder: (context, i) => _EpisodeTile(
            episode: _episodes[i],
            textPrimary: textPrimary,
            textSub: textSub,
            onTap: () {
              notifier.openRecordedPlayer(_episodes[i]);
              context.push('/recorded-player/${_episodes[i].id}');
            },
          ),
        ),
      ],
    );
  }

  Widget _buildExplore(BuildContext context) {
    final textPrimary = dark ? lyoTextDark : lyoTextLight;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 24, 20, 14),
          child: Text(
            'Explore',
            style: TextStyle(
              fontSize: lyoH2,
              fontWeight: FontWeight.w700,
              color: textPrimary,
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20),
          child: GridView.count(
            crossAxisCount: 2,
            crossAxisSpacing: 10,
            mainAxisSpacing: 10,
            childAspectRatio: 2.1,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            children: _categories
                .map((c) => _CategoryCard(category: c))
                .toList(),
          ),
        ),
      ],
    );
  }
}

// ── Live show card ────────────────────────────────────────────────────────────

class _LiveShowCard extends StatelessWidget {
  const _LiveShowCard({required this.show, required this.onTap});
  final LyoLiveShow show;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        width: 160,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(16),
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: show.colors,
          ),
        ),
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Container(
                    padding: const EdgeInsets.symmetric(
                        horizontal: 7, vertical: 3),
                    decoration: BoxDecoration(
                      color: lyoAccent,
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: const Text(
                      'LIVE',
                      style: TextStyle(
                        fontSize: lyoTiny,
                        fontWeight: FontWeight.w700,
                        color: Colors.white,
                        letterSpacing: 0.5,
                      ),
                    ),
                  ),
                  const LiveEQWidget(isPlaying: true),
                ],
              ),
              const Spacer(),
              Text(
                show.title,
                style: const TextStyle(
                  fontSize: lyoBody2,
                  fontWeight: FontWeight.w700,
                  color: Colors.white,
                  height: 1.2,
                ),
              ),
              const SizedBox(height: 3),
              Text(
                show.host,
                style: TextStyle(
                  fontSize: lyoSmall,
                  color: Colors.white.withValues(alpha: 0.6),
                ),
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  Icon(
                    Icons.visibility_outlined,
                    size: 12,
                    color: Colors.white.withValues(alpha: 0.5),
                  ),
                  const SizedBox(width: 4),
                  Text(
                    _fmt(show.listeners),
                    style: TextStyle(
                      fontSize: lyoSmall,
                      color: Colors.white.withValues(alpha: 0.5),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Episode tile ──────────────────────────────────────────────────────────────

class _EpisodeTile extends StatelessWidget {
  const _EpisodeTile({
    required this.episode,
    required this.textPrimary,
    required this.textSub,
    required this.onTap,
  });

  final LyoEpisode episode;
  final Color textPrimary;
  final Color textSub;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 10),
        child: Row(
          children: [
            LyoArtworkTile(
              size: 52,
              radius: 10,
              color1: episode.colors[0],
              color2: episode.colors[1],
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    episode.title,
                    style: TextStyle(
                      fontSize: lyoBody2,
                      fontWeight: FontWeight.w600,
                      color: textPrimary,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 3),
                  Text(
                    '${episode.show} · ${episode.duration}',
                    style: TextStyle(
                      fontSize: lyoBody2 - 2,
                      color: textSub,
                    ),
                  ),
                ],
              ),
            ),
            Icon(Icons.more_vert, size: 18, color: textSub),
          ],
        ),
      ),
    );
  }
}

// ── Category card ─────────────────────────────────────────────────────────────

class _CategoryCard extends StatelessWidget {
  const _CategoryCard({required this.category});
  final LyoCategory category;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(14),
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: category.colors,
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Icon(category.icon, size: 18, color: lyoAccent),
            Text(
              category.label,
              style: const TextStyle(
                fontSize: lyoCaption,
                fontWeight: FontWeight.w700,
                color: Colors.white,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ── Stub body for Browse / Search / Profile tabs ──────────────────────────────

class _StubBody extends StatelessWidget {
  const _StubBody({required this.label, required this.icon, required this.dark});
  final String label;
  final IconData icon;
  final bool dark;

  @override
  Widget build(BuildContext context) {
    final textSub = dark ? lyoSubDark : lyoSubLight;
    return SafeArea(
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 48, color: lyoAccent.withValues(alpha: 0.4)),
            const SizedBox(height: lyoGapM),
            Text(
              '$label — coming soon',
              style: TextStyle(color: textSub, fontSize: lyoBody1),
            ),
          ],
        ),
      ),
    );
  }
}
