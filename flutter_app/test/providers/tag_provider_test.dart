import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:gallery/models/tag.dart';
import 'package:gallery/models/tag_governance.dart';
import 'package:gallery/providers/tag_provider.dart';
import 'package:gallery/services/tag_service.dart';
import 'package:mocktail/mocktail.dart';

class MockTagService extends Mock implements TagService {}

void main() {
  group('TagProvider', () {
    late TagProvider tagProvider;
    late MockTagService mockTagService;

    setUp(() {
      mockTagService = MockTagService();
      tagProvider = TagProvider(mockTagService);
    });

    group('Tag Selection', () {
      test('toggleTag adds tag to selection', () {
        tagProvider.toggleTag(1);
        expect(tagProvider.selectedTagIds.contains(1), true);
      });

      test('toggleTag removes tag from selection if already selected', () {
        tagProvider.toggleTag(1);
        tagProvider.toggleTag(1);
        expect(tagProvider.selectedTagIds.contains(1), false);
      });

      test('clearSelection removes all selected tags', () {
        tagProvider.toggleTag(1);
        tagProvider.toggleTag(2);
        tagProvider.clearSelection();
        expect(tagProvider.selectedTagIds.isEmpty, true);
      });

      test('selectedTags returns tags matching selected IDs', () async {
        final tags = [
          Tag(
            id: 1,
            preferredLabel: 'Tag 1',
            slug: 'tag-1',
            reviewState: 'confirmed',
            trustScore: 0.8,
            usageCount: 10,
            createdAt: DateTime.now(),
          ),
          Tag(
            id: 2,
            preferredLabel: 'Tag 2',
            slug: 'tag-2',
            reviewState: 'confirmed',
            trustScore: 0.7,
            usageCount: 5,
            createdAt: DateTime.now(),
          ),
        ];

        when(() => mockTagService.fetchTags()).thenAnswer((_) async => tags);

        await tagProvider.loadTags();
        tagProvider.toggleTag(1);

        expect(tagProvider.selectedTags.length, 1);
        expect(tagProvider.selectedTags.first.preferredLabel, 'Tag 1');
      });
    });

    group('Loading Tags', () {
      test('loadTags fetches and stores tags', () async {
        final tags = [
          Tag(
            id: 1,
            preferredLabel: 'Test Tag',
            slug: 'test-tag',
            reviewState: 'confirmed',
            trustScore: 0.8,
            usageCount: 10,
            createdAt: DateTime.now(),
          ),
        ];

        when(() => mockTagService.fetchTags()).thenAnswer((_) async => tags);

        await tagProvider.loadTags();

        expect(tagProvider.allTags.length, 1);
        expect(tagProvider.filteredTags.length, 1);
        expect(tagProvider.isLoading, false);
        expect(tagProvider.error, null);
      });

      test('loadTags sets error on failure', () async {
        when(
          () => mockTagService.fetchTags(),
        ).thenThrow(Exception('Network error'));

        await tagProvider.loadTags();

        expect(tagProvider.allTags.isEmpty, true);
        expect(tagProvider.error, isNotNull);
        expect(tagProvider.isLoading, false);
      });

      test('searchTags filters tags', () async {
        final tags = [
          Tag(
            id: 1,
            preferredLabel: 'Anime Character',
            slug: 'anime-character',
            reviewState: 'confirmed',
            trustScore: 0.8,
            usageCount: 100,
            createdAt: DateTime.now(),
          ),
        ];

        when(
          () => mockTagService.searchTags('anime'),
        ).thenAnswer((_) async => tags);

        await tagProvider.searchTags('anime');

        expect(tagProvider.filteredTags.length, 1);
        expect(
          tagProvider.filteredTags.first.preferredLabel,
          'Anime Character',
        );
      });

      test('searchTags with empty query shows all tags', () async {
        final tags = [
          Tag(
            id: 1,
            preferredLabel: 'Tag 1',
            slug: 'tag-1',
            reviewState: 'confirmed',
            trustScore: 0.8,
            usageCount: 10,
            createdAt: DateTime.now(),
          ),
          Tag(
            id: 2,
            preferredLabel: 'Tag 2',
            slug: 'tag-2',
            reviewState: 'confirmed',
            trustScore: 0.7,
            usageCount: 5,
            createdAt: DateTime.now(),
          ),
        ];

        when(() => mockTagService.fetchTags()).thenAnswer((_) async => tags);

        await tagProvider.loadTags();
        await tagProvider.searchTags('');

        expect(tagProvider.filteredTags.length, 2);
      });
    });

    group('Image Tag Operations', () {
      test('loadImageTags fetches image tags', () async {
        final Map<String, List<Tag>> imageTags = {
          'confirmed': [
            Tag(
              id: 1,
              preferredLabel: 'Confirmed Tag',
              slug: 'confirmed',
              reviewState: 'confirmed',
              trustScore: 0.9,
              usageCount: 100,
              createdAt: DateTime.now(),
            ),
          ],
          'pending': [],
          'rejected': [],
        };

        when(
          () => mockTagService.getImageTags(123),
        ).thenAnswer((_) async => imageTags);

        await tagProvider.loadImageTags(123);

        expect(tagProvider.imageTags['confirmed']!.length, 1);
        expect(tagProvider.isLoadingImageTags, false);
      });

      test('loadImageTags ignores stale results from older requests', () async {
        final firstRequest = Completer<Map<String, List<Tag>>>();
        final secondRequest = Completer<Map<String, List<Tag>>>();

        when(
          () => mockTagService.getImageTags(1),
        ).thenAnswer((_) => firstRequest.future);
        when(
          () => mockTagService.getImageTags(2),
        ).thenAnswer((_) => secondRequest.future);

        final firstLoad = tagProvider.loadImageTags(1);
        final secondLoad = tagProvider.loadImageTags(2);

        secondRequest.complete({
          'confirmed': [
            Tag(
              id: 2,
              preferredLabel: 'Current Tag',
              slug: 'current-tag',
              reviewState: 'confirmed',
              trustScore: 0.9,
              usageCount: 3,
              createdAt: DateTime.now(),
            ),
          ],
          'pending': [],
          'rejected': [],
        });
        await secondLoad;

        firstRequest.complete({
          'confirmed': [
            Tag(
              id: 1,
              preferredLabel: 'Stale Tag',
              slug: 'stale-tag',
              reviewState: 'confirmed',
              trustScore: 0.5,
              usageCount: 1,
              createdAt: DateTime.now(),
            ),
          ],
          'pending': [],
          'rejected': [],
        });
        await firstLoad;

        expect(
          tagProvider.imageTags['confirmed']!.single.preferredLabel,
          'Current Tag',
        );
        expect(tagProvider.isLoadingImageTags, false);
      });

      test('confirmImageTag moves tag from pending to confirmed', () async {
        final pendingTag = Tag(
          id: 1,
          preferredLabel: 'Pending Tag',
          slug: 'pending',
          reviewState: 'pending',
          trustScore: 0.7,
          usageCount: 5,
          createdAt: DateTime.now(),
        );

        // Set up initial state
        tagProvider = TagProvider(mockTagService);
        when(() => mockTagService.confirmTag(123, 1)).thenAnswer((_) async {});

        // Manually set image tags
        tagProvider.imageTags['pending']!.add(pendingTag);

        await tagProvider.confirmImageTag(123, 1);

        expect(tagProvider.imageTags['pending']!.isEmpty, true);
        expect(tagProvider.imageTags['confirmed']!.length, 1);
        expect(
          tagProvider.imageTags['confirmed']!.first.reviewState,
          'confirmed',
        );
      });

      test('rejectImageTag moves tag from pending to rejected', () async {
        final pendingTag = Tag(
          id: 1,
          preferredLabel: 'Pending Tag',
          slug: 'pending',
          reviewState: 'pending',
          trustScore: 0.7,
          usageCount: 5,
          createdAt: DateTime.now(),
        );

        when(() => mockTagService.rejectTag(123, 1)).thenAnswer((_) async {});

        tagProvider.imageTags['pending']!.add(pendingTag);

        await tagProvider.rejectImageTag(123, 1);

        expect(tagProvider.imageTags['pending']!.isEmpty, true);
        expect(tagProvider.imageTags['rejected']!.length, 1);
        expect(
          tagProvider.imageTags['rejected']!.first.reviewState,
          'rejected',
        );
      });

      test('removeImageTag removes tag from all lists', () async {
        final tag = Tag(
          id: 1,
          preferredLabel: 'Tag',
          slug: 'tag',
          reviewState: 'confirmed',
          trustScore: 0.8,
          usageCount: 10,
          createdAt: DateTime.now(),
        );

        when(
          () => mockTagService.removeImageTag(123, 1),
        ).thenAnswer((_) async {});

        tagProvider.imageTags['confirmed']!.add(tag);

        await tagProvider.removeImageTag(123, 1);

        expect(tagProvider.imageTags['confirmed']!.isEmpty, true);
      });

      test('addImageTag adds tag to confirmed list', () async {
        final newTag = Tag(
          id: 1,
          preferredLabel: 'New Tag',
          slug: 'new-tag',
          reviewState: 'confirmed',
          trustScore: 0.8,
          usageCount: 1,
          createdAt: DateTime.now(),
        );

        when(
          () => mockTagService.addImageTag(123, tagLabel: 'New Tag'),
        ).thenAnswer((_) async => newTag);

        await tagProvider.addImageTag(123, tagLabel: 'New Tag');

        expect(tagProvider.imageTags['confirmed']!.length, 1);
        expect(
          tagProvider.imageTags['confirmed']!.first.preferredLabel,
          'New Tag',
        );
      });

      test('triggerAITags returns job id', () async {
        when(
          () => mockTagService.triggerAITags(123),
        ).thenAnswer((_) async => 999);

        final jobId = await tagProvider.triggerAITags(123);

        expect(jobId, 999);
      });

      test('getAITagStatus returns status', () async {
        final status = {'status': 'processing', 'progress': 0.5};

        when(
          () => mockTagService.getAITagStatus(123),
        ).thenAnswer((_) async => status);

        final result = await tagProvider.getAITagStatus(123);

        expect(result['status'], 'processing');
        expect(result['progress'], 0.5);
      });
    });

    group('Error Handling', () {
      test('clearError resets error state', () {
        tagProvider.loadTags(); // This will fail with mock
        tagProvider.clearError();
        expect(tagProvider.error, null);
      });

      test('confirmImageTag sets error on failure', () async {
        when(
          () => mockTagService.confirmTag(123, 1),
        ).thenThrow(Exception('Failed'));

        expect(() => tagProvider.confirmImageTag(123, 1), throwsException);
      });
    });

    group('Governance Workspace', () {
      final governanceRow = TagGovernanceRow(
        tagId: 101,
        preferredLabel: 'anime-girl',
        primaryCategory: 'character',
        aliases: const ['waifu'],
        usageCount: 42,
        pendingCount: 3,
        confirmedCount: 37,
        rejectedCount: 2,
        aiCount: 30,
        manualCount: 12,
        affectedImageCount: 42,
        canDelete: false,
      );

      test('loadGovernanceTags fetches and stores governance rows', () async {
        when(
          () => mockTagService.fetchGovernanceTags(search: 'anime'),
        ).thenAnswer((_) async => [governanceRow]);

        await tagProvider.loadGovernanceTags(search: 'anime');

        expect(tagProvider.governanceRows, hasLength(1));
        expect(tagProvider.governanceRows.first.tagId, 101);
      });

      test(
        'toggleGovernanceSelection and clearGovernanceSelection manage selected IDs',
        () {
          tagProvider.toggleGovernanceSelection(101);
          tagProvider.toggleGovernanceSelection(102);
          expect(tagProvider.selectedGovernanceIds, {101, 102});

          tagProvider.toggleGovernanceSelection(101);
          expect(tagProvider.selectedGovernanceIds, {102});

          tagProvider.clearGovernanceSelection();
          expect(tagProvider.selectedGovernanceIds, isEmpty);
        },
      );

      test(
        'setActiveMergeSource and clearActiveMergeSource track merge source',
        () {
          tagProvider.setActiveMergeSource(governanceRow);
          expect(tagProvider.activeMergeSource?.tagId, 101);

          tagProvider.clearActiveMergeSource();
          expect(tagProvider.activeMergeSource, isNull);
        },
      );

      test('loadDeletePreview stores preview state', () async {
        when(() => mockTagService.fetchDeletePreview(101)).thenAnswer(
          (_) async => const TagDeletePreview(
            tagId: 101,
            preferredLabel: 'anime-girl',
            affectedImageCount: 42,
            canDelete: false,
            blockingReason: 'merge_or_reclassify_required',
          ),
        );

        await tagProvider.loadDeletePreview(101);

        expect(tagProvider.deletePreview?.tagId, 101);
        expect(tagProvider.deletePreview?.canDelete, false);
      });

      test(
        'applyPrimaryCategoryToSelection aggregates failures and refreshes rows',
        () async {
          when(
            () => mockTagService.fetchGovernanceTags(
              search: any(named: 'search'),
            ),
          ).thenAnswer((_) async => [governanceRow]);
          when(
            () => mockTagService.updateTag(101, primaryCategory: 'character'),
          ).thenAnswer((_) async => _buildTag(101, 'anime-girl'));
          when(
            () => mockTagService.updateTag(102, primaryCategory: 'character'),
          ).thenThrow(Exception('blocked'));

          tagProvider.toggleGovernanceSelection(101);
          tagProvider.toggleGovernanceSelection(102);

          final result = await tagProvider.applyPrimaryCategoryToSelection(
            'character',
          );

          expect(result.deletedTagIds, isEmpty);
          expect(result.failures, hasLength(1));
          expect(result.failures.first.tagId, 102);
        },
      );

      test(
        'addAliasToSelection aggregates failures and refreshes rows',
        () async {
          when(
            () => mockTagService.fetchGovernanceTags(
              search: any(named: 'search'),
            ),
          ).thenAnswer((_) async => [governanceRow]);
          when(
            () => mockTagService.addTagAlias(101, 'heroine', 'synonym'),
          ).thenAnswer((_) async {});
          when(
            () => mockTagService.addTagAlias(102, 'heroine', 'synonym'),
          ).thenThrow(Exception('invalid alias'));

          tagProvider.toggleGovernanceSelection(101);
          tagProvider.toggleGovernanceSelection(102);

          final result = await tagProvider.addAliasToSelection('heroine');

          expect(result.failures, hasLength(1));
          expect(result.failures.first.tagId, 102);
        },
      );

      test(
        'cleanupSelectedUnusedTags keeps batch result and refreshes rows',
        () async {
          when(
            () => mockTagService.fetchGovernanceTags(
              search: any(named: 'search'),
            ),
          ).thenAnswer((_) async => [governanceRow]);
          when(() => mockTagService.batchCleanupTags([101, 102])).thenAnswer(
            (_) async => const TagGovernanceBatchResult(
              deletedTagIds: [101],
              failures: [
                TagGovernanceFailure(
                  tagId: 102,
                  preferredLabel: 'used-tag',
                  message: 'tag is still in use',
                ),
              ],
            ),
          );

          tagProvider.toggleGovernanceSelection(101);
          tagProvider.toggleGovernanceSelection(102);
          final result = await tagProvider.cleanupSelectedUnusedTags();

          expect(result.deletedTagIds, [101]);
          expect(tagProvider.lastBatchResult?.failures, hasLength(1));
        },
      );

      test(
        'mergeSelectionInto uses active merge source and selected ids',
        () async {
          when(
            () => mockTagService.fetchGovernanceTags(
              search: any(named: 'search'),
            ),
          ).thenAnswer((_) async => [governanceRow]);
          when(
            () => mockTagService.mergeTagInto(101, 999),
          ).thenAnswer((_) async {});
          when(
            () => mockTagService.mergeTagInto(102, 999),
          ).thenThrow(Exception('cannot merge'));

          tagProvider.setActiveMergeSource(governanceRow);
          tagProvider.toggleGovernanceSelection(101);
          tagProvider.toggleGovernanceSelection(102);

          final result = await tagProvider.mergeSelectionInto(999);

          expect(result.failures, hasLength(1));
          expect(result.failures.first.tagId, 102);
          expect(tagProvider.selectedGovernanceIds, {101, 102});
        },
      );
    });
  });
}

Tag _buildTag(int id, String label) {
  return Tag(
    id: id,
    preferredLabel: label,
    slug: label,
    reviewState: 'confirmed',
    trustScore: 1,
    usageCount: 1,
    createdAt: DateTime.fromMillisecondsSinceEpoch(0),
  );
}
