import 'dart:io';

import 'package:http/http.dart' as http;

import '../../../core/api/api_client.dart';
import '../models/track_model.dart';

class TrackService {
  const TrackService(this._api);
  final ApiClient _api;

  Future<List<Track>> listTracks({
    int page = 1,
    int limit = 20,
    String? broadcasterId,
    String? token,
  }) async {
    final query = {
      'page': '$page',
      'limit': '$limit',
      'broadcaster_id': ?broadcasterId,
    };
    final qs = Uri(queryParameters: query).query;
    final data = await _api.get('/tracks?$qs', token: token);
    final items = (data as Map<String, dynamic>)['items'] as List<dynamic>;
    return items.cast<Map<String, dynamic>>().map(Track.fromJson).toList();
  }

  Future<Track> getTrack(String id, {String? token}) async {
    final data = await _api.get('/tracks/$id', token: token);
    return Track.fromJson(data as Map<String, dynamic>);
  }

  Future<Track> createTrack({
    required String title,
    required String audioUrl,
    required String token,
    String? artist,
    int? durationSeconds,
  }) async {
    final data = await _api.post(
      '/tracks',
      {
        'title': title,
        'artist': ?artist,
        'audio_url': audioUrl,
        'duration_seconds': ?durationSeconds,
      },
      token: token,
    );
    return Track.fromJson(data);
  }

  Future<void> deleteTrack(String id, String token) async {
    await _api.delete('/tracks/$id', token: token);
  }

  // Presigned-upload flow: ask the backend for an S3 PUT URL, upload the raw
  // file bytes directly to S3, then record the metadata via createTrack.
  Future<Track> uploadTrack({
    required File file,
    required String title,
    required String token,
    String? artist,
    int? durationSeconds,
  }) async {
    final filename = file.path.split('/').last;
    final presign = await _api.post(
      '/tracks/upload-url',
      {'filename': filename},
      token: token,
    );
    final uploadUrl = presign['upload_url'] as String;
    final audioUrl = presign['audio_url'] as String;

    final bytes = await file.readAsBytes();
    final putResponse = await http
        .put(Uri.parse(uploadUrl), body: bytes)
        .timeout(const Duration(minutes: 5));
    if (putResponse.statusCode >= 400) {
      throw ApiException(
        putResponse.statusCode,
        'Upload failed (${putResponse.statusCode})',
      );
    }

    return createTrack(
      title: title,
      artist: artist,
      audioUrl: audioUrl,
      durationSeconds: durationSeconds,
      token: token,
    );
  }
}
