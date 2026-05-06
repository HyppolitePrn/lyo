import 'package:flutter/material.dart';

import '../../../core/theme/lyo_tokens.dart';

class AuthErrorBanner extends StatelessWidget {
  const AuthErrorBanner({required this.message, super.key});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: const Color(0x1AE05A5A),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: lyoError.withValues(alpha: 0.3)),
      ),
      child: Row(
        children: [
          const Icon(Icons.error_outline, color: lyoError, size: 18),
          const SizedBox(width: lyoGapS),
          Expanded(
            child: Text(
              message,
              style: const TextStyle(color: lyoError, fontSize: lyoBody2),
            ),
          ),
        ],
      ),
    );
  }
}
