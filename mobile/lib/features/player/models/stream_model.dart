class LiveStream {
  const LiveStream({
    required this.id,
    required this.broadcasterId,
    required this.title,
    this.description,
    required this.status,
    required this.startedAt,
    this.endedAt,
  });

  final String id;
  final String broadcasterId;
  final String title;
  final String? description;
  final String status;
  final DateTime startedAt;
  final DateTime? endedAt;

  bool get isLive => status == 'live';

  factory LiveStream.fromJson(Map<String, dynamic> json) {
    return LiveStream(
      id: json['id'] as String,
      broadcasterId: json['broadcaster_id'] as String,
      title: json['title'] as String,
      description: json['description'] as String?,
      status: json['status'] as String,
      startedAt: DateTime.parse(json['started_at'] as String),
      endedAt: json['ended_at'] != null
          ? DateTime.parse(json['ended_at'] as String)
          : null,
    );
  }
}
