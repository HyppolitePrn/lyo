import 'package:flutter/material.dart';

class LyoArtworkTile extends StatelessWidget {
  const LyoArtworkTile({
    required this.size, required this.radius, required this.color1, required this.color2, super.key,
  });

  final double size;
  final double radius;
  final Color color1;
  final Color color2;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(radius),
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [color1, color2],
        ),
      ),
    );
  }
}
