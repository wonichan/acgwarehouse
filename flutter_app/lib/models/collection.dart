class Collection {
  final int id;
  final String name;
  final String? description;
  final int? coverImageId;
  final int imageCount;
  final DateTime createdAt;
  final DateTime updatedAt;

  const Collection({
    required this.id,
    required this.name,
    this.description,
    this.coverImageId,
    required this.imageCount,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Collection.fromJson(Map<String, dynamic> json) {
    return Collection(
      id: json['id'] as int,
      name: json['name'] as String,
      description: json['description'] as String?,
      coverImageId: json['cover_image_id'] as int?,
      imageCount: json['image_count'] as int? ?? 0,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'description': description,
      'cover_image_id': coverImageId,
      'image_count': imageCount,
      'created_at': createdAt.toIso8601String(),
      'updated_at': updatedAt.toIso8601String(),
    };
  }

  Collection copyWith({
    String? name,
    String? description,
    int? coverImageId,
    int? imageCount,
  }) {
    return Collection(
      id: id,
      name: name ?? this.name,
      description: description ?? this.description,
      coverImageId: coverImageId ?? this.coverImageId,
      imageCount: imageCount ?? this.imageCount,
      createdAt: createdAt,
      updatedAt: DateTime.now(),
    );
  }
}