import '../../track/models/track_model.dart';

class Playlist {
  const Playlist({
    required this.id,
    required this.ownerId,
    required this.title,
    required this.trackIds,
    required this.isPublic,
    required this.createdAt,
    required this.updatedAt,
    this.description,
    this.tracks,
  });

  final String id;
  final String ownerId;
  final String title;
  final String? description;
  final List<String> trackIds;
  final bool isPublic;
  final DateTime createdAt;
  final DateTime updatedAt;
  // Only populated by GetPlaylist; null for list/create/update responses.
  final List<Track>? tracks;

  factory Playlist.fromJson(Map<String, dynamic> json) {
    return Playlist(
      id: json['id'] as String,
      ownerId: json['owner_id'] as String,
      title: json['title'] as String,
      description: json['description'] as String?,
      trackIds: (json['track_ids'] as List).cast<String>(),
      isPublic: json['is_public'] as bool,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
      tracks: (json['tracks'] as List?)
          ?.cast<Map<String, dynamic>>()
          .map(Track.fromJson)
          .toList(),
    );
  }
}
