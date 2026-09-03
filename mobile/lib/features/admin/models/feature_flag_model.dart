class FeatureFlag {
  const FeatureFlag({
    required this.id,
    required this.name,
    required this.enabled,
    required this.description,
    required this.updatedAt,
  });

  final String id;
  final String name;
  final bool enabled;
  final String description;
  final DateTime updatedAt;

  factory FeatureFlag.fromJson(Map<String, dynamic> json) {
    return FeatureFlag(
      id: json['id'] as String,
      name: json['name'] as String,
      enabled: json['enabled'] as bool,
      description: json['description'] as String,
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  FeatureFlag copyWith({bool? enabled}) {
    return FeatureFlag(
      id: id,
      name: name,
      enabled: enabled ?? this.enabled,
      description: description,
      updatedAt: updatedAt,
    );
  }
}
