# 血缘构建清理契约

## Scenario: BuildAll flow-scoped cleanup

### 1. Scope / Trigger

- Trigger: 修改 `lineage_node`、`lineage_relation` 的构建清理边界。
- 适用范围：`LineageBuild.BuildAll`、`NodeRepository.DeleteNodesNotInBuild`、`RelationRepository.DeleteRelationsNotInBuild`。
- 原则：`build_version` 只判定同一构建范围内的新旧，不得作为跨 flow 全局删除条件。

### 2. Signatures

```go
BuildAll(ctx context.Context, requests []*dto.FlowBuildRequest) errors.DexError

DeleteNodesNotInBuild(ctx context.Context, buildVersion string, flows []po.FlowVersionRef) (int64, error)

DeleteRelationsNotInBuild(ctx context.Context, buildVersion string, flows []po.FlowVersionRef) (int64, error)
```

### 3. Contracts

- `buildVersion`：本次构建批次号。
- `flows`：本次构建覆盖的 flow scope，按 `flow_type_id + version` 标识，不依赖 `flow_name`。
- `flow_versions`：节点或关系归属的 flow 列表。
- 清理只能影响 `flows` 覆盖范围内、且 `build_version != buildVersion` 的旧归属。
- 共享节点或关系仍属于其他 flow 时，只移除当前 flow 归属，不删除整条文档。
- 文档的 `flow_versions` 被移空后，方可删除该节点或关系。

### 4. Validation & Error Matrix

- `buildVersion == ""` -> 返回错误，禁止无版本清理。
- `flows` 为空 -> 返回 `0, nil`，禁止无范围清理。
- `flows` 中无有效 `flow_type_id + version` -> 返回 `0, nil`。
- Mongo 更新或删除失败 -> 原样向上返回错误。

### 5. Good/Base/Bad Cases

- Good: flow A 构建后再构建 flow B，flow A 的节点和关系仍存在。
- Base: 同一 flow 二次构建，旧图里消失的节点和关系被清理。
- Bad: `DeleteMany({"build_version": {"$ne": buildVersion}})` 全局删除，会误删其他 flow 的旧版本数据。

### 6. Tests Required

- `BuildAll` 单测需覆盖同一 flow 二次构建，断言旧节点和旧关系消失。
- `BuildAll` 单测需覆盖 flow A 后构建 flow B，断言 flow A 图仍可查。
- 共享节点/关系场景需断言当前 flow 归属被移除，但文档未被整条删除。
- 仓储接口签名改动后，mock 与所有调用点必须同步编译通过。

### 7. Wrong vs Correct

#### Wrong

```go
DeleteMany(ctx, bson.M{
	"is_active":     1,
	"build_version": bson.M{"$ne": buildVersion},
})
```

#### Correct

```go
UpdateMany(ctx, bson.M{
	"is_active":     1,
	"build_version": bson.M{"$ne": buildVersion},
	"flow_versions": bson.M{"$elemMatch": bson.M{"$or": flowConditions}},
}, bson.M{
	"$pull": bson.M{"flow_versions": bson.M{"$or": flowConditions}},
})

DeleteMany(ctx, bson.M{
	"is_active":     1,
	"build_version": bson.M{"$ne": buildVersion},
	"flow_versions": bson.M{"$size": 0},
})
```
