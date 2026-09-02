import 'package:flutter/material.dart';

import '../../../core/theme/lyo_tokens.dart';

class FavoriteButton extends StatelessWidget {
  const FavoriteButton({
    required this.isFavorited,
    required this.onTap,
    super.key,
    this.size = 22,
  });

  final bool isFavorited;
  final VoidCallback onTap;
  final double size;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.all(6),
        child: Icon(
          isFavorited ? Icons.favorite : Icons.favorite_border,
          size: size,
          color: isFavorited ? lyoAccent : lyoSubDark,
        ),
      ),
    );
  }
}
