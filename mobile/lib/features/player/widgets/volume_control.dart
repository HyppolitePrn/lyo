import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/theme/lyo_tokens.dart';
import '../services/volume_controller.dart';

IconData _iconFor(double level) {
  if (level == 0) {
    return Icons.volume_off;
  }
  if (level < 0.5) {
    return Icons.volume_down;
  }
  return Icons.volume_up;
}

/// Speaker toggle plus a slider driving [VolumeController]. Shared by both
/// player screens so live and recorded playback expose the same control.
class VolumeControl extends StatelessWidget {
  const VolumeControl({super.key});

  @override
  Widget build(BuildContext context) {
    final volume = context.watch<VolumeController>();
    final dark = Theme.of(context).brightness == Brightness.dark;
    final textSub = dark ? lyoSubDark : lyoSubLight;
    final percent = (volume.level * 100).round();

    return Row(
      children: [
        IconButton(
          tooltip: volume.isMuted ? 'Unmute' : 'Mute',
          onPressed: context.read<VolumeController>().toggleMute,
          icon: Icon(_iconFor(volume.level), color: textSub, size: 22),
        ),
        Expanded(
          child: SliderTheme(
            data: SliderTheme.of(context).copyWith(
              trackHeight: 3,
              thumbShape: const RoundSliderThumbShape(enabledThumbRadius: 6),
            ),
            child: Slider(
              value: volume.level,
              label: '$percent%',
              semanticFormatterCallback: (v) => '${(v * 100).round()}% volume',
              activeColor: lyoAccent,
              inactiveColor: dark ? lyoBorderDark : lyoBorderLight,
              onChanged: context.read<VolumeController>().setLevel,
            ),
          ),
        ),
      ],
    );
  }
}
