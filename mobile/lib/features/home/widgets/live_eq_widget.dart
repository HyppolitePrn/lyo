import 'dart:async';
import 'dart:math';

import 'package:flutter/material.dart';

import '../../../core/theme/lyo_tokens.dart';

class LiveEQWidget extends StatefulWidget {
  const LiveEQWidget({super.key, required this.isPlaying});
  final bool isPlaying;

  @override
  State<LiveEQWidget> createState() => _LiveEQWidgetState();
}

class _LiveEQWidgetState extends State<LiveEQWidget>
    with WidgetsBindingObserver {
  final _rng = Random();
  List<double> _heights = [0.6, 0.3, 0.8, 0.4];
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    if (widget.isPlaying) _start();
  }

  @override
  void didUpdateWidget(LiveEQWidget old) {
    super.didUpdateWidget(old);
    if (widget.isPlaying && !old.isPlaying) {
      _start();
    } else if (!widget.isPlaying && old.isPlaying) {
      _stop();
    }
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState lifecycleState) {
    if (lifecycleState == AppLifecycleState.resumed && widget.isPlaying) {
      _start();
    } else if (lifecycleState != AppLifecycleState.resumed) {
      _stop();
    }
  }

  void _start() {
    if (_timer != null) return;
    _timer = Timer.periodic(const Duration(milliseconds: 160), (_) {
      if (mounted) {
        setState(() {
          _heights =
              List.generate(4, (_) => 0.2 + _rng.nextDouble() * 0.8);
        });
      }
    });
  }

  void _stop() {
    _timer?.cancel();
    _timer = null;
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _stop();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 20,
      height: 14,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.end,
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: _heights
            .map(
              (h) => AnimatedContainer(
                duration: const Duration(milliseconds: 140),
                curve: Curves.easeInOut,
                width: 3,
                height: h * 14,
                decoration: BoxDecoration(
                  color: lyoAccent,
                  borderRadius: BorderRadius.circular(1.5),
                ),
              ),
            )
            .toList(),
      ),
    );
  }
}
