import '../../../core/api/api_client.dart';

class User {
  const User({
    required this.id,
    required this.username,
    required this.email,
    required this.role,
    required this.favoriteTrackIds,
    required this.favoriteStreamIds,
    required this.favoritePlaylistIds,
    required this.createdAt,
    required this.updatedAt,
  });

  final String id;
  final String username;
  final String email;
  final String role;
  final List<String> favoriteTrackIds;
  final List<String> favoriteStreamIds;
  final List<String> favoritePlaylistIds;
  final DateTime createdAt;
  final DateTime updatedAt;

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: json['id'] as String,
      username: json['username'] as String,
      email: json['email'] as String,
      role: json['role'] as String,
      favoriteTrackIds: (json['favorite_track_ids'] as List).cast<String>(),
      favoriteStreamIds: (json['favorite_stream_ids'] as List).cast<String>(),
      favoritePlaylistIds:
          (json['favorite_playlist_ids'] as List).cast<String>(),
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }
}

class UserService {
  const UserService(this._api);
  final ApiClient _api;

  Future<User> getMe(String token) async {
    final data = await _api.get('/users/me', token: token);
    return User.fromJson(data as Map<String, dynamic>);
  }
}
