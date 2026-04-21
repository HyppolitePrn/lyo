import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../providers/auth_notifier.dart';

class SplashScreen extends ConsumerWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      backgroundColor: lyoBgDark,
      body: Stack(
        children: [
          // Radial gradient overlays
          Positioned.fill(
            child: DecoratedBox(
              decoration: const BoxDecoration(
                gradient: RadialGradient(
                  center: Alignment(0.2, -0.4),
                  radius: 0.8,
                  colors: [Color(0x33E8B84A), Colors.transparent],
                ),
              ),
            ),
          ),
          Positioned.fill(
            child: DecoratedBox(
              decoration: const BoxDecoration(
                gradient: RadialGradient(
                  center: Alignment(-0.6, 0.6),
                  radius: 0.6,
                  colors: [Color(0x1AE8B84A), Colors.transparent],
                ),
              ),
            ),
          ),
          SafeArea(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Expanded(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: lyoPadH),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const SizedBox(height: lyoGapXXL),
                        _LogoRow(),
                        const SizedBox(height: lyoGapXXL),
                        _ArtworkCollage(),
                        const Spacer(),
                        _HeroCopy(),
                        const SizedBox(height: lyoGapXL),
                      ],
                    ),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(
                    lyoPadH,
                    0,
                    lyoPadH,
                    lyoGapXXL,
                  ),
                  child: _CtaBlock(),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _LogoRow extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 40,
          height: 40,
          decoration: BoxDecoration(
            color: lyoAccent,
            borderRadius: BorderRadius.circular(12),
          ),
          child: const Icon(Icons.radio, size: 20, color: Colors.white),
        ),
        const SizedBox(width: lyoGapS),
        const Text(
          'lyo',
          style: TextStyle(
            color: lyoTextDark,
            fontSize: 26,
            fontWeight: FontWeight.w800,
            letterSpacing: -0.5,
          ),
        ),
      ],
    );
  }
}

class _ArtworkCollage extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 210,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          Positioned(
            left: 0,
            top: 20,
            child: _ArtCard(
              width: 120,
              height: 120,
              radius: 18,
              angle: -6 * math.pi / 180,
              gradient: const LinearGradient(
                colors: [Color(0xFF1A2A3D), Color(0xFF0D4F6B)],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              label: 'LIVE',
            ),
          ),
          Positioned(
            left: 0,
            right: 0,
            top: 0,
            child: Center(
              child: _ArtCard(
                width: 140,
                height: 140,
                radius: 18,
                angle: 2 * math.pi / 180,
                gradient: const LinearGradient(
                  colors: [Color(0xFF3D1A0A), Color(0xFF7A3520)],
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                ),
                label: 'EP',
              ),
            ),
          ),
          Positioned(
            right: 0,
            top: 30,
            child: _ArtCard(
              width: 110,
              height: 110,
              radius: 18,
              angle: 5 * math.pi / 180,
              gradient: const LinearGradient(
                colors: [Color(0xFF1A1A3D), Color(0xFF3D2070)],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              label: 'POD',
            ),
          ),
          Positioned(
            bottom: 0,
            left: 20,
            child: _ArtCard(
              width: 90,
              height: 90,
              radius: 14,
              angle: -3 * math.pi / 180,
              gradient: const LinearGradient(
                colors: [Color(0xFF1A3D1A), Color(0xFF1F6B2E)],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
            ),
          ),
          Positioned(
            bottom: 10,
            right: 10,
            child: _ArtCard(
              width: 80,
              height: 80,
              radius: 14,
              angle: 4 * math.pi / 180,
              gradient: const LinearGradient(
                colors: [Color(0xFF3D2D1A), Color(0xFF6B4A20)],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ArtCard extends StatelessWidget {
  const _ArtCard({
    required this.width,
    required this.height,
    required this.radius,
    required this.angle,
    required this.gradient,
    this.label,
  });

  final double width;
  final double height;
  final double radius;
  final double angle;
  final LinearGradient gradient;
  final String? label;

  @override
  Widget build(BuildContext context) {
    return Transform.rotate(
      angle: angle,
      child: Container(
        width: width,
        height: height,
        decoration: BoxDecoration(
          gradient: gradient,
          borderRadius: BorderRadius.circular(radius),
        ),
        alignment: Alignment.center,
        child: label != null
            ? Text(
                label!,
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 13,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 1.5,
                ),
              )
            : null,
      ),
    );
  }
}

class _HeroCopy extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return const Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Listen live.\nHear everything.',
          style: TextStyle(
            color: lyoTextDark,
            fontSize: lyoDisplay,
            fontWeight: FontWeight.w800,
            letterSpacing: -0.8,
            height: 1.15,
          ),
        ),
        SizedBox(height: lyoGapM),
        Text(
          'Tune in to live radio stations or catch up on recorded episodes — all in one place.',
          style: TextStyle(
            color: lyoSubDark,
            fontSize: lyoBody1,
            height: 1.6,
          ),
        ),
      ],
    );
  }
}

class _CtaBlock extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        DecoratedBox(
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(lyoRadiusBtn),
            boxShadow: const [lyoCtaGlow],
          ),
          child: ElevatedButton(
            onPressed: () => context.push('/register'),
            child: const Text('Get Started'),
          ),
        ),
        const SizedBox(height: lyoGapM),
        OutlinedButton(
          style: OutlinedButton.styleFrom(
            side: const BorderSide(color: lyoBorderDark, width: 1.5),
            foregroundColor: lyoTextDark,
          ),
          onPressed: () => context.push('/login'),
          child: const Text('Sign In'),
        ),
        const SizedBox(height: lyoGapS),
        TextButton(
          onPressed: () {
            ref.read(authNotifierProvider.notifier).continueAnonymously();
            context.go('/home');
          },
          child: const Text(
            'Browse without account',
            style: TextStyle(
              color: lyoSubDark,
              fontSize: lyoBody2,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
      ],
    );
  }
}
