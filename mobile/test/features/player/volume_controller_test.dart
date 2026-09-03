import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/features/player/services/volume_controller.dart';

void main() {
  late VolumeController volume;
  var notifications = 0;

  setUp(() {
    notifications = 0;
    volume = VolumeController()..addListener(() => notifications++);
  });

  test('starts at full volume, unmuted', () {
    expect(volume.level, 1.0);
    expect(volume.isMuted, isFalse);
  });

  test('setLevel notifies listeners', () {
    volume.setLevel(0.3);
    expect(volume.level, 0.3);
    expect(notifications, 1);
  });

  test('setLevel clamps out-of-range values', () {
    volume.setLevel(1.8);
    expect(volume.level, 1.0);
    volume.setLevel(-0.5);
    expect(volume.level, 0.0);
  });

  test('setting the same level does not notify', () {
    volume.setLevel(1.0);
    expect(notifications, 0);
  });

  test('mute drops to silence and unmute restores the previous level', () {
    volume.setLevel(0.4);
    volume.toggleMute();
    expect(volume.level, 0.0);
    expect(volume.isMuted, isTrue);

    volume.toggleMute();
    expect(volume.level, 0.4);
    expect(volume.isMuted, isFalse);
  });

  test('unmuting from a level of zero falls back to full volume', () {
    // Dragging the slider to 0 mutes without recording a level to come back
    // to, so the speaker button must still produce audible sound.
    volume.setLevel(0);
    expect(volume.isMuted, isTrue);

    volume.toggleMute();
    expect(volume.level, 1.0);
  });
}
