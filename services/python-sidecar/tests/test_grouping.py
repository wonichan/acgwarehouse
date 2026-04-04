"""Grouping module tests."""

# pyright: reportMissingImports=false

from compute.grouping import UnionFind, group_duplicates, hamming_distance


def test_union_find_fresh_find_returns_self():
    uf = UnionFind(3)

    assert uf.find(0) == 0


def test_union_find_union_merges_two_nodes():
    uf = UnionFind(3)
    uf.union(0, 1)

    assert uf.find(0) == uf.find(1)


def test_union_find_transitivity_holds():
    uf = UnionFind(4)
    uf.union(0, 1)
    uf.union(1, 2)

    assert uf.find(0) == uf.find(2)


def test_union_find_groups_returns_only_multi_member_groups():
    uf = UnionFind(4)
    uf.union(0, 1)

    groups = uf.groups()

    assert len(groups) == 1
    assert sorted(list(groups.values())[0]) == [0, 1]


def test_group_duplicates_identical_phashes_returns_one_group():
    hashes = [
        {"image_id": 1, "sha256": "a" * 64, "phash": "0" * 64},
        {"image_id": 2, "sha256": "b" * 64, "phash": "0" * 64},
    ]

    groups = group_duplicates(hashes, threshold=0)

    assert len(groups) == 1
    assert groups[0]["member_indices"] == [0, 1]


def test_group_duplicates_dissimilar_hashes_returns_no_group():
    hashes = [
        {"image_id": 1, "sha256": "a" * 64, "phash": "0" * 64},
        {"image_id": 2, "sha256": "b" * 64, "phash": "f" * 64},
    ]

    groups = group_duplicates(hashes, threshold=3)

    assert groups == []


def test_group_duplicates_mixed_similarity_groups_correctly():
    hashes = [
        {"image_id": 1, "sha256": "a" * 64, "phash": "0" * 64},
        {"image_id": 2, "sha256": "b" * 64, "phash": "0" * 64},
        {"image_id": 3, "sha256": "c" * 64, "phash": "f" * 64},
    ]

    groups = group_duplicates(hashes, threshold=0)

    assert len(groups) == 1
    assert groups[0]["member_indices"] == [0, 1]


def test_hamming_distance_same_hash_is_zero():
    assert hamming_distance("0" * 64, "0" * 64) == 0


def test_hamming_distance_known_difference_matches_expected():
    assert hamming_distance("0" * 64, "f" * 64) == 256
