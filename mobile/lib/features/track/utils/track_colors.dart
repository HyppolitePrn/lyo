import 'package:flutter/material.dart';

const _trackPalettes = [
  [Color(0xFF0D1F3D), Color(0xFF1B4F8A)],
  [Color(0xFF1A0D3D), Color(0xFF4A2090)],
  [Color(0xFF1A2A0D), Color(0xFF3D6B1A)],
  [Color(0xFF3D1A0A), Color(0xFF6B3320)],
  [Color(0xFF0A1A3D), Color(0xFF203D7A)],
];

// Deterministic gradient for a track's artwork placeholder, keyed by its ID.
List<Color> trackColors(String id) {
  final idx = id.codeUnits.fold(0, (a, b) => a + b) % _trackPalettes.length;
  return _trackPalettes[idx];
}
