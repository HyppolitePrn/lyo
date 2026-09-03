import 'dart:io';

import 'package:flutter/foundation.dart';

import '../../../core/api/api_client.dart';
import '../models/track_model.dart';
import '../services/track_service.dart';

enum UploadTrackStatus { idle, uploading, success, error }

class UploadTrackNotifier extends ChangeNotifier {
  UploadTrackNotifier({TrackService? trackService})
    : _svc = trackService ?? const TrackService(ApiClient());

  final TrackService _svc;

  UploadTrackStatus status = UploadTrackStatus.idle;
  File? selectedFile;
  Track? uploadedTrack;
  String? error;

  void selectFile(File file) {
    selectedFile = file;
    error = null;
    notifyListeners();
  }

  void clearFile() {
    selectedFile = null;
    notifyListeners();
  }

  Future<void> upload({
    required String title,
    required String token,
    String? artist,
  }) async {
    final file = selectedFile;
    if (file == null) {
      error = 'Pick an audio file first.';
      notifyListeners();
      return;
    }

    status = UploadTrackStatus.uploading;
    error = null;
    notifyListeners();

    try {
      final track = await _svc.uploadTrack(
        file: file,
        title: title,
        artist: artist,
        token: token,
      );
      status = UploadTrackStatus.success;
      uploadedTrack = track;
      notifyListeners();
    } on ApiException catch (e) {
      status = UploadTrackStatus.error;
      error = e.message;
      notifyListeners();
    } catch (_) {
      status = UploadTrackStatus.error;
      error = 'Upload failed. Try again.';
      notifyListeners();
    }
  }

  void reset() {
    status = UploadTrackStatus.idle;
    selectedFile = null;
    uploadedTrack = null;
    error = null;
    notifyListeners();
  }
}
