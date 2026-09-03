import 'package:flutter/foundation.dart';

/// App-level playback volume, shared by the live and the recorded player so a
/// listener sets it once and it carries over between the two.
///
/// This scales Lyo's own output only — the device's master volume is left
/// alone, so turning a stream down never silences the rest of the phone.
class VolumeController extends ChangeNotifier {
  static const double _defaultLevel = 1.0;

  double _level = _defaultLevel;
  // Where the slider was before a mute, so unmuting restores it instead of
  // jumping back to full blast.
  double _levelBeforeMute = _defaultLevel;

  /// 0.0 (silent) to 1.0 (full), the range `just_audio` expects.
  double get level => _level;

  bool get isMuted => _level == 0;

  void setLevel(double value) {
    final clamped = value.clamp(0.0, 1.0);
    if (clamped == _level) {
      return;
    }
    _level = clamped;
    notifyListeners();
  }

  void toggleMute() {
    if (isMuted) {
      setLevel(_levelBeforeMute == 0 ? _defaultLevel : _levelBeforeMute);
      return;
    }
    _levelBeforeMute = _level;
    setLevel(0);
  }
}
