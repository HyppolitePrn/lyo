class Track {
  const Track({
    required this.id,
    required this.broadcasterId,
    required this.title,
    required this.audioUrl,
    required this.durationSeconds,
    required this.createdAt,
    this.artist,
  });

  final String id;
  final String broadcasterId;
  final String title;
  final String? artist;
  final String audioUrl;
  final int durationSeconds;
  final DateTime createdAt;

  factory Track.fromJson(Map<String, dynamic> json) {
    return Track(
      id: json['id'] as String,
      broadcasterId: json['broadcaster_id'] as String,
      title: json['title'] as String,
      artist: json['artist'] as String?,
      audioUrl: json['audio_url'] as String,
      durationSeconds: json['duration_seconds'] as int,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}
