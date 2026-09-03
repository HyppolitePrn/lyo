import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import '../../../core/features/feature_flags_provider.dart';
import '../../../core/theme/lyo_tokens.dart';
import '../../auth/providers/auth_notifier.dart';

String initialsFrom(String? username) {
  final trimmed = username?.trim() ?? '';
  if (trimmed.isEmpty) {
    return '?';
  }
  final letters = trimmed.length >= 2 ? trimmed.substring(0, 2) : trimmed;
  return letters.toUpperCase();
}

class ProfileScreen extends StatelessWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthNotifier>();
    final dark = Theme.of(context).brightness == Brightness.dark;

    if (!auth.isAuthenticated) {
      return _SignedOutBody(dark: dark);
    }

    return _ProfileBody(
      auth: auth,
      dark: dark,
      onSignOut: () {
        context.read<AuthNotifier>().signOut();
        context.go('/login');
      },
    );
  }
}

// ── Signed-out state ─────────────────────────────────────────────────────────

class _SignedOutBody extends StatelessWidget {
  const _SignedOutBody({required this.dark});
  final bool dark;

  @override
  Widget build(BuildContext context) {
    final textSub = dark ? lyoSubDark : lyoSubLight;
    return SafeArea(
      child: Center(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: lyoPadH),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ExcludeSemantics(
                child: Icon(
                  Icons.person_outline,
                  size: 48,
                  color: lyoAccent.withValues(alpha: 0.4),
                ),
              ),
              const SizedBox(height: lyoGapM),
              Text(
                'Connecte-toi pour voir ton profil',
                textAlign: TextAlign.center,
                style: TextStyle(color: textSub, fontSize: lyoBody1),
              ),
              const SizedBox(height: lyoGapL),
              ElevatedButton(
                onPressed: () => context.go('/login'),
                child: const Text('Se connecter'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ── Signed-in state ───────────────────────────────────────────────────────────

class _ProfileBody extends StatelessWidget {
  const _ProfileBody({
    required this.auth,
    required this.dark,
    required this.onSignOut,
  });

  final AuthNotifier auth;
  final bool dark;
  final VoidCallback onSignOut;

  @override
  Widget build(BuildContext context) {
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final flags = context.watch<FeatureFlags>();

    return SafeArea(
      child: ListView(
        padding: const EdgeInsets.fromLTRB(
          lyoPadHMain,
          lyoGapXL,
          lyoPadHMain,
          lyoGapXXL,
        ),
        children: [
          Center(
            child: Semantics(
              label:
                  '${auth.username ?? 'Unknown user'}, '
                  '${auth.email ?? 'no email'}',
              excludeSemantics: true,
              child: Column(
                children: [
                  Container(
                    width: 72,
                    height: 72,
                    decoration: const BoxDecoration(
                      color: lyoAccent,
                      shape: BoxShape.circle,
                    ),
                    alignment: Alignment.center,
                    child: Text(
                      initialsFrom(auth.username),
                      style: const TextStyle(
                        fontSize: 26,
                        fontWeight: FontWeight.w700,
                        color: Colors.white,
                      ),
                    ),
                  ),
                  const SizedBox(height: lyoGapM),
                  Text(
                    auth.username ?? '—',
                    style: TextStyle(
                      fontSize: lyoH2,
                      fontWeight: FontWeight.w700,
                      color: textPrimary,
                    ),
                  ),
                  const SizedBox(height: lyoGapXS),
                  Text(
                    auth.email ?? '—',
                    style: TextStyle(fontSize: lyoBody2, color: textSub),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: lyoGapXXL),
          _InfoTile(
            icon: Icons.badge_outlined,
            label: 'Rôle',
            value: auth.role ?? '—',
            dark: dark,
          ),
          if (flags.isEnabled('playlists')) ...[
            const SizedBox(height: lyoGapM),
            _NavTile(
              icon: Icons.queue_music_outlined,
              label: 'My Playlists',
              dark: dark,
              onTap: () => context.push('/playlists'),
            ),
          ],
          if (flags.isEnabled('favorites')) ...[
            const SizedBox(height: lyoGapM),
            _NavTile(
              icon: Icons.favorite_border,
              label: 'Favorites',
              dark: dark,
              onTap: () => context.push('/favorites'),
            ),
          ],
          const SizedBox(height: lyoGapXXL),
          OutlinedButton.icon(
            style: OutlinedButton.styleFrom(
              minimumSize: const Size(double.infinity, 52),
              foregroundColor: lyoError,
              side: const BorderSide(color: lyoError),
            ),
            onPressed: onSignOut,
            icon: const Icon(Icons.logout),
            label: const Text('Se déconnecter'),
          ),
        ],
      ),
    );
  }
}

class _NavTile extends StatelessWidget {
  const _NavTile({
    required this.icon,
    required this.label,
    required this.dark,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final bool dark;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final surface = dark ? lyoSurfaceDark : lyoSurfaceLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final border = dark ? lyoBorderDark : lyoBorderLight;

    return Semantics(
      button: true,
      label: label,
      excludeSemantics: true,
      onTap: onTap,
      child: GestureDetector(
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.all(lyoGapL),
          decoration: BoxDecoration(
            color: surface,
            borderRadius: BorderRadius.circular(lyoRadiusCard),
            border: Border.all(color: border),
          ),
          child: Row(
            children: [
              Icon(icon, size: 18, color: lyoAccent),
              const SizedBox(width: lyoGapM),
              Text(
                label,
                style: TextStyle(
                  color: textPrimary,
                  fontWeight: FontWeight.w600,
                  fontSize: lyoBody2,
                ),
              ),
              const Spacer(),
              Icon(Icons.chevron_right, color: textSub),
            ],
          ),
        ),
      ),
    );
  }
}

class _InfoTile extends StatelessWidget {
  const _InfoTile({
    required this.icon,
    required this.label,
    required this.value,
    required this.dark,
  });

  final IconData icon;
  final String label;
  final String value;
  final bool dark;

  @override
  Widget build(BuildContext context) {
    final surface = dark ? lyoSurfaceDark : lyoSurfaceLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final textPrimary = dark ? lyoTextDark : lyoTextLight;
    final border = dark ? lyoBorderDark : lyoBorderLight;

    return Semantics(
      label: '$label : $value',
      excludeSemantics: true,
      child: Container(
        padding: const EdgeInsets.all(lyoGapL),
        decoration: BoxDecoration(
          color: surface,
          borderRadius: BorderRadius.circular(lyoRadiusCard),
          border: Border.all(color: border),
        ),
        child: Row(
          children: [
            Icon(icon, size: 18, color: textSub),
            const SizedBox(width: lyoGapM),
            Text(
              label,
              style: TextStyle(color: textSub, fontSize: lyoBody2),
            ),
            const Spacer(),
            Text(
              value,
              style: TextStyle(
                color: textPrimary,
                fontWeight: FontWeight.w600,
                fontSize: lyoBody2,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
