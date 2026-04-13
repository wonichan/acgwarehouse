import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/models/tag_governance.dart';

void main() {
  group('Tag Model', () {
    test('fromJson correctly parses JSON', () {
      final json = {
        'id': 1,
        'preferred_label': 'Test Tag',
        'slug': 'test-tag',
        'primary_category': 'Test Category',
        'review_state': 'confirmed',
        'trust_score': 0.85,
        'usage_count': 42,
        'created_at': '2024-01-15T10:30:00Z',
      };

      final tag = Tag.fromJson(json);

      expect(tag.id, 1);
      expect(tag.preferredLabel, 'Test Tag');
      expect(tag.slug, 'test-tag');
      expect(tag.primaryCategory, 'Test Category');
      expect(tag.reviewState, 'confirmed');
      expect(tag.trustScore, 0.85);
      expect(tag.usageCount, 42);
      expect(tag.createdAt, DateTime.parse('2024-01-15T10:30:00Z'));
      expect(tag.level, null);
      expect(tag.parentId, null);
    });

    test('fromJson correctly parses hierarchy fields', () {
      final json = {
        'id': 100,
        'preferred_label': 'Sub Tag',
        'slug': 'sub-tag',
        'review_state': 'confirmed',
        'trust_score': 0.9,
        'usage_count': 5,
        'created_at': '2024-01-15T10:30:00Z',
        'level': 'parent',
        'parent_id': 1,
      };

      final tag = Tag.fromJson(json);
      expect(tag.level, 'parent');
      expect(tag.parentId, 1);
    });

    test('fromJson handles optional primary_category', () {
      final json = {
        'id': 2,
        'preferred_label': 'No Category Tag',
        'slug': 'no-category-tag',
        'primary_category': null,
        'review_state': 'pending',
        'trust_score': 0.5,
        'usage_count': 10,
        'created_at': '2024-01-16T08:00:00Z',
      };

      final tag = Tag.fromJson(json);

      expect(tag.id, 2);
      expect(tag.primaryCategory, null);
    });

    test('fromJson parses trust_score as int', () {
      final json = {
        'id': 3,
        'preferred_label': 'Integer Score Tag',
        'slug': 'integer-score-tag',
        'primary_category': null,
        'review_state': 'rejected',
        'trust_score': 1,
        'usage_count': 5,
        'created_at': '2024-01-17T12:00:00Z',
      };

      final tag = Tag.fromJson(json);

      expect(tag.trustScore, 1.0);
    });

    test('toJson correctly serializes to JSON', () {
      final tag = Tag(
        id: 1,
        preferredLabel: 'Test Tag',
        slug: 'test-tag',
        primaryCategory: 'Test Category',
        reviewState: 'confirmed',
        trustScore: 0.85,
        usageCount: 42,
        createdAt: DateTime.parse('2024-01-15T10:30:00Z'),
      );

      final json = tag.toJson();

      expect(json['id'], 1);
      expect(json['preferred_label'], 'Test Tag');
      expect(json['slug'], 'test-tag');
      expect(json['primary_category'], 'Test Category');
      expect(json['review_state'], 'confirmed');
      expect(json['trust_score'], 0.85);
      expect(json['usage_count'], 42);
      expect(json['created_at'], '2024-01-15T10:30:00.000Z');
      expect(json.containsKey('level'), false);
      expect(json.containsKey('parent_id'), false);
    });

    test('toJson correctly serializes hierarchy fields', () {
      final tag = Tag(
        id: 1,
        preferredLabel: 'Test Tag',
        slug: 'test-tag',
        reviewState: 'confirmed',
        trustScore: 0.85,
        usageCount: 42,
        createdAt: DateTime.parse('2024-01-15T10:30:00Z'),
        level: 'child',
        parentId: 99,
      );

      final json = tag.toJson();
      expect(json['level'], 'child');
      expect(json['parent_id'], 99);
    });

    test('toJson handles null primary_category', () {
      final tag = Tag(
        id: 2,
        preferredLabel: 'No Category',
        slug: 'no-category',
        primaryCategory: null,
        reviewState: 'pending',
        trustScore: 0.5,
        usageCount: 10,
        createdAt: DateTime.parse('2024-01-16T08:00:00Z'),
      );

      final json = tag.toJson();

      expect(json['primary_category'], null);
    });

    test('copyWith creates a copy with updated values', () {
      final tag = Tag(
        id: 1,
        preferredLabel: 'Original',
        slug: 'original',
        primaryCategory: 'Category',
        reviewState: 'pending',
        trustScore: 0.5,
        usageCount: 10,
        createdAt: DateTime.parse('2024-01-15T10:30:00Z'),
      );

      final updatedTag = tag.copyWith(
        reviewState: 'confirmed',
        usageCount: 11,
        level: 'root',
      );

      expect(updatedTag.id, tag.id);
      expect(updatedTag.preferredLabel, tag.preferredLabel);
      expect(updatedTag.reviewState, 'confirmed');
      expect(updatedTag.usageCount, 11);
      expect(updatedTag.level, 'root');
    });
  });

  group('TagStatistics and TagGovernanceRow', () {
    test('TagStatistics.fromJson handles hierarchy stats', () {
      final json = {
        'tag_id': 10,
        'label': 'Test Stat',
        'usage_count': 100,
        'level': 'root',
        'parent_id': null,
        'direct_usage_count': 20,
        'tree_usage_count': 100,
        'direct_pending_count': 5,
        'tree_pending_count': 15,
      };

      final stat = TagStatistics.fromJson(json);
      expect(stat.tagId, 10);
      expect(stat.label, 'Test Stat');
      expect(stat.level, 'root');
      expect(stat.parentId, null);
      expect(stat.directUsageCount, 20);
      expect(stat.treeUsageCount, 100);
      expect(stat.directPendingCount, 5);
      expect(stat.treePendingCount, 15);
    });

    test('TagGovernanceRow.fromJson handles hierarchy stats', () {
      final json = {
        'tag_id': 11,
        'preferred_label': 'Gov Tag',
        'usage_count': 50,
        'level': 'parent',
        'parent_id': 10,
        'direct_usage_count': 30,
        'tree_usage_count': 50,
        'direct_confirmed_count': 20,
        'tree_confirmed_count': 40,
      };

      final row = TagGovernanceRow.fromJson(json);
      expect(row.tagId, 11);
      expect(row.preferredLabel, 'Gov Tag');
      expect(row.level, 'parent');
      expect(row.parentId, 10);
      expect(row.directUsageCount, 30);
      expect(row.treeUsageCount, 50);
      expect(row.directConfirmedCount, 20);
      expect(row.treeConfirmedCount, 40);
    });
  });
}
