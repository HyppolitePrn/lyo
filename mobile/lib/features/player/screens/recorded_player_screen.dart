import 'package:flutter/material.dart';

import '../../../core/theme/lyo_tokens.dart';

class RecordedPlayerScreen extends StatelessWidget {
  const RecordedPlayerScreen({required this.episodeId, super.key});
  final String episodeId;

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final bg = dark ? lyoBgDark : lyoBgLight;
    final textSub = dark ? lyoSubDark : lyoSubLight;

    return Scaffold(
      backgroundColor: bg,
      appBar: AppBar(
        backgroundColor: bg,
        elevation: 0,
        iconTheme: IconThemeData(color: dark ? lyoTextDark : lyoTextLight),
        title: Text(
          'Episode Player',
          style: TextStyle(
            color: dark ? lyoTextDark : lyoTextLight,
            fontSize: lyoH1,
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.headphones,
              size: 64,
              color: lyoAccent.withValues(alpha: 0.4),
            ),
            const SizedBox(height: lyoGapM),
            Text(
              'Recorded Player — coming soon',
              style: TextStyle(color: textSub, fontSize: lyoBody1),
            ),
          ],
        ),
      ),
    );
  }
}
