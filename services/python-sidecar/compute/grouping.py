import imagehash


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


def hamming_distance(hash1: str, hash2: str) -> int:
    return imagehash.hex_to_hash(hash1) - imagehash.hex_to_hash(hash2)


def group_duplicates(hashes: list[dict], threshold: int) -> list[dict]:
    groups: list[dict] = []
    group_id = 1
    exact_groups: dict[str, list[int]] = {}
    exact_member_indices: set[int] = set()

    for index, item in enumerate(hashes):
        sha256 = item.get("sha256")
        if sha256:
            exact_groups.setdefault(sha256, []).append(index)

    for members in exact_groups.values():
        if len(members) > 1:
            sorted_members = sorted(members)
            groups.append(
                {
                    "group_id": group_id,
                    "member_indices": sorted_members,
                    "type": "exact",
                }
            )
            exact_member_indices.update(sorted_members)
            group_id += 1

    remaining_indices = [
        index
        for index, item in enumerate(hashes)
        if index not in exact_member_indices and item.get("phash")
    ]

    if len(remaining_indices) < 2:
        return groups

    uf = UnionFind(len(remaining_indices))
    for i in range(len(remaining_indices)):
        for j in range(i + 1, len(remaining_indices)):
            left = hashes[remaining_indices[i]]
            right = hashes[remaining_indices[j]]
            distance = hamming_distance(left["phash"], right["phash"])
            if distance <= threshold:
                uf.union(i, j)

    for members in uf.groups().values():
        original_indices = sorted(remaining_indices[member] for member in members)
        groups.append(
            {
                "group_id": group_id,
                "member_indices": original_indices,
                "type": "similar",
            }
        )
        group_id += 1

    return groups
