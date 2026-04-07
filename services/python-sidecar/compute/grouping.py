# pyright: reportMissingImports=false

from imagededup.methods import PHash


HashRecord = dict[str, object]
GroupRecord = dict[str, object]


_PHASHER = PHash(verbose=False)


class UnionFind:
    def __init__(self, n: int):
        self.parent = list(range(n))
        self.rank = [0] * n

    def find(self, x: int) -> int:
        if self.parent[x] != x:
            self.parent[x] = self.find(self.parent[x])
        return self.parent[x]

    def union(self, x: int, y: int) -> None:
        root_x = self.find(x)
        root_y = self.find(y)
        if root_x == root_y:
            return

        if self.rank[root_x] < self.rank[root_y]:
            root_x, root_y = root_y, root_x

        self.parent[root_y] = root_x
        if self.rank[root_x] == self.rank[root_y]:
            self.rank[root_x] += 1

    def groups(self) -> dict[int, list[int]]:
        group_map: dict[int, list[int]] = {}
        for index in range(len(self.parent)):
            root = self.find(index)
            group_map.setdefault(root, []).append(index)
        return {
            root: members for root, members in group_map.items() if len(members) > 1
        }


def _phash_to_int(hash_value: str) -> int:
    return int(hash_value, 16)


def hamming_distance(hash1: str, hash2: str) -> int:
    return (_phash_to_int(hash1) ^ _phash_to_int(hash2)).bit_count()


def group_duplicates(hashes: list[HashRecord], threshold: int) -> list[GroupRecord]:
    if len(hashes) < 2:
        return []

    uf = UnionFind(len(hashes))

    exact_groups: dict[str, list[int]] = {}
    for index, item in enumerate(hashes):
        sha256_obj = item.get("sha256")
        if isinstance(sha256_obj, str) and sha256_obj:
            exact_groups.setdefault(sha256_obj, []).append(index)

    for members in exact_groups.values():
        for i in range(1, len(members)):
            uf.union(members[0], members[i])

    normalized_threshold = max(0, threshold)
    encoding_map: dict[str, str] = {}
    for index, item in enumerate(hashes):
        phash_obj = item.get("phash")
        if not isinstance(phash_obj, str) or not phash_obj:
            continue
        if len(phash_obj) != 16:
            continue
        try:
            _phash_to_int(phash_obj)
        except ValueError:
            continue
        encoding_map[str(index)] = phash_obj

    if encoding_map:
        duplicate_map = _PHASHER.find_duplicates(
            encoding_map=encoding_map,
            max_distance_threshold=normalized_threshold,
            scores=True,
            num_dist_workers=0,
        )
        for source_key, matches in duplicate_map.items():
            source_index = int(source_key)
            for match in matches:
                if isinstance(match, tuple):
                    target_key = match[0]
                else:
                    target_key = match
                try:
                    target_index = int(target_key)
                except (TypeError, ValueError):
                    continue
                uf.union(source_index, target_index)

    groups: list[GroupRecord] = []
    group_id = 1
    for members in uf.groups().values():
        sorted_members = sorted(members)
        sha_values = {
            sha
            for idx in sorted_members
            for sha in [hashes[idx].get("sha256")]
            if isinstance(sha, str) and sha
        }
        group_type = "exact" if len(sha_values) == 1 and len(sha_values) > 0 else "similar"
        groups.append(
            {
                "group_id": group_id,
                "member_indices": sorted_members,
                "type": group_type,
            }
        )
        group_id += 1

    return groups
