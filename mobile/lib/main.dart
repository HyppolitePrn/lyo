import 'package:flutter/material.dart';

void main() {
  runApp(const LyoApp());
}

class LyoApp extends StatelessWidget {
  const LyoApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Lyo',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF6C63FF)),
        useMaterial3: true,
      ),
      home: const Scaffold(
        body: Center(
          child: Text('Lyo — StreamPulse'),
        ),
      ),
    );
  }
}
