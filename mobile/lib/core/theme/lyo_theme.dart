import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import 'lyo_tokens.dart';

ThemeData lyoTheme(Brightness brightness) {
  final isDark = brightness == Brightness.dark;

  final bg = isDark ? lyoBgDark : lyoBgLight;
  final surface = isDark ? lyoSurfaceDark : lyoSurfaceLight;
  final border = isDark ? lyoBorderDark : lyoBorderLight;
  final text = isDark ? lyoTextDark : lyoTextLight;
  final sub = isDark ? lyoSubDark : lyoSubLight;

  final base = ThemeData(brightness: brightness);

  return ThemeData(
    brightness: brightness,
    scaffoldBackgroundColor: bg,
    colorScheme: ColorScheme(
      brightness: brightness,
      primary: lyoAccent,
      onPrimary: Colors.white,
      secondary: lyoAccent,
      onSecondary: Colors.white,
      error: lyoError,
      onError: Colors.white,
      surface: surface,
      onSurface: text,
    ),
    textTheme: GoogleFonts.plusJakartaSansTextTheme(base.textTheme).apply(
      bodyColor: text,
      displayColor: text,
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: surface,
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(lyoRadiusInput),
        borderSide: BorderSide.none,
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(lyoRadiusInput),
        borderSide: BorderSide.none,
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(lyoRadiusInput),
        borderSide: const BorderSide(color: lyoAccent, width: 1.5),
      ),
      errorBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(lyoRadiusInput),
        borderSide: const BorderSide(color: lyoError, width: 1.5),
      ),
      focusedErrorBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(lyoRadiusInput),
        borderSide: const BorderSide(color: lyoError, width: 1.5),
      ),
      contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
      hintStyle: TextStyle(color: sub, fontSize: lyoBody1),
      labelStyle: TextStyle(
        color: sub,
        fontSize: lyoCaption,
        fontWeight: FontWeight.w500,
      ),
      errorStyle: const TextStyle(color: lyoError, fontSize: lyoCaption),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: lyoAccent,
        foregroundColor: Colors.white,
        minimumSize: const Size(double.infinity, 56),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(lyoRadiusBtn),
        ),
        textStyle: GoogleFonts.plusJakartaSans(
          fontSize: 16,
          fontWeight: FontWeight.w700,
        ),
        elevation: 0,
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: text,
        minimumSize: const Size(double.infinity, 56),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(lyoRadiusBtn),
        ),
        side: BorderSide(color: border, width: 1.5),
        textStyle: GoogleFonts.plusJakartaSans(
          fontSize: 16,
          fontWeight: FontWeight.w600,
        ),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: lyoAccent,
        textStyle: GoogleFonts.plusJakartaSans(
          fontSize: lyoCaption,
          fontWeight: FontWeight.w600,
        ),
      ),
    ),
    checkboxTheme: CheckboxThemeData(
      fillColor: WidgetStateProperty.resolveWith(
        (states) => states.contains(WidgetState.selected) ? lyoAccent : null,
      ),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
    ),
    dividerColor: border,
  );
}
