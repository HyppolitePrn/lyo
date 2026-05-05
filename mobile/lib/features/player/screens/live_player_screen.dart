import 'package:flutter/material.dart';

import '../../../core/theme/lyo_tokens.dart';

class LivePlayerScreen extends StatelessWidget {
  const LivePlayerScreen({super.key, required this.showId});
  final String showId;

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
          'Live Player',
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
            Icon(Icons.radio, size: 64, color: lyoAccent.withValues(alpha: 0.4)),
            const SizedBox(height: lyoGapM),
            Text(
              'Live Player — coming soon',
              style: TextStyle(color: textSub, fontSize: lyoBody1),
            ),
          ],
        ),
      ),
    );
  }
}
