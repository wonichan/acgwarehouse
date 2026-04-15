class GalleryFilterState {
  final Set<int> exactTagIds;
  final Set<int> subtreeRootTagIds;
  final bool? hasTags;
  final bool? hasPendingTags;

  GalleryFilterState({
    Set<int>? exactTagIds,
    Set<int>? subtreeRootTagIds,
    this.hasTags,
    this.hasPendingTags,
  }) : exactTagIds = Set.unmodifiable(exactTagIds ?? const <int>{}),
       subtreeRootTagIds = Set.unmodifiable(subtreeRootTagIds ?? const <int>{});

  static const Object _unset = Object();

  bool get isEmpty =>
      exactTagIds.isEmpty &&
      subtreeRootTagIds.isEmpty &&
      hasTags == null &&
      hasPendingTags == null;

  GalleryFilterState copyWith({
    Set<int>? exactTagIds,
    Set<int>? subtreeRootTagIds,
    Object? hasTags = _unset,
    Object? hasPendingTags = _unset,
  }) {
    return GalleryFilterState(
      exactTagIds: exactTagIds ?? this.exactTagIds,
      subtreeRootTagIds: subtreeRootTagIds ?? this.subtreeRootTagIds,
      hasTags: identical(hasTags, _unset) ? this.hasTags : hasTags as bool?,
      hasPendingTags: identical(hasPendingTags, _unset)
          ? this.hasPendingTags
          : hasPendingTags as bool?,
    );
  }

  GalleryFilterState clear() {
    return GalleryFilterState();
  }

  GalleryFilterState normalized() {
    if (hasTags == false) {
      return GalleryFilterState(hasTags: false);
    }

    return GalleryFilterState(
      exactTagIds: exactTagIds,
      subtreeRootTagIds: subtreeRootTagIds,
      hasTags: hasTags,
      hasPendingTags: hasPendingTags,
    );
  }
}
