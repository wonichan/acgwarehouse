---
phase: 04-duplicate-detection-search
plan: 01
subsystem: duplicate-detection
tags: [hash, phash, duplicate, detection, api]
dependencies:
  requires: []
  provides: [duplicate-detection-backend]
  affects: [internal/service, internal/repository, internal/handler, internal/domain]
tech-stack:
  added:
    - github.com/corona10/goimagehash v1.1.0
  patterns:
    - Union-Find algorithm for transitive grouping
    - TDD (Test-Driven Development)
key-files:
  created:
    - internal/service/hash_service.go
    - internal/service/hash_service_test.go
    - internal/service/duplicate_service.go
    - internal/service/duplicate_service_test.go
    - internal/repository/duplicate_repository.go
    - internal/repository/duplicate_repository_test.go
    - internal/handler/duplicate_handler.go
    - internal/handler/duplicate_handler_test.go
    - internal/domain/duplicate_group.go
  modified:
    - internal/repository/schema.go
    - internal/handler/routes.go
    - go.mod
    - go.sum
decisions:
  - Use Union-Find algorithm for transitive duplicate grouping
  - Auto-select highest resolution image as recommended
  - Support both SHA256 file hash and pHash for detection
  - Threshold parameter for similarity detection (default 10, max 64)
metrics:
  duration: ~30 minutes
  completed_date: 2026-03-17
  tasks_completed: 4
  files_created: 9
  files_modified: 4
  test_coverage: All components have comprehensive tests
---

# Phase 04 Plan 01: Duplicate Detection Backend Summary

## 一句话总结

实现了完整的重复检测后端层，包括 SHA256 文件哈希检测完全相同图片、pHash 感知哈希检测相似图片、Union-Find 传递性分组算法和 RESTful API 端点。

## 完成的功能

### Task 1: 哈希计算服务
- **SHA256 文件哈希**：计算文件的 SHA256 哈希值，用于检测完全相同的图片
- **pHash 感知哈希**：使用 goimagehash 库计算图片的感知哈希，用于检测相似图片
- **汉明距离计算**：计算两个哈希值之间的汉明距离，用于相似度比较

### Task 2: 数据模型和仓储
- **DuplicateGroup**：重复组模型，包含推荐图片 ID 和相似度阈值
- **DuplicateRelation**：图片与重复组的关系模型，包含文件哈希和汉明距离
- **DuplicateRepository**：完整的 CRUD 操作，支持按图片 ID 查询分组

### Task 3: 重复检测服务
- **Union-Find 算法**：实现传递性分组（A~B, B~C → {A,B,C}）
- **双重检测**：同时支持文件哈希匹配和 pHash 相似度检测
- **自动推荐**：选择分辨率最高的图片作为推荐保留

### Task 4: API 端点
- **POST /api/v1/duplicates/detect**：触发重复检测
- **GET /api/v1/duplicates**：获取重复组列表（分页）
- **GET /api/v1/duplicates/:id**：获取单个重复组详情
- **DELETE /api/v1/duplicates/:id**：删除重复组记录

## 技术亮点

### Union-Find 传递性分组
```go
type UnionFind struct {
    parent []int
    rank   []int
}

// 带路径压缩的查找
func (uf *UnionFind) Find(x int) int {
    if uf.parent[x] != x {
        uf.parent[x] = uf.Find(uf.parent[x])
    }
    return uf.parent[x]
}

// 按秩合并
func (uf *UnionFind) Union(x, y int) {
    px, py := uf.Find(x), uf.Find(y)
    if px == py {
        return
    }
    if uf.rank[px] < uf.rank[py] {
        px, py = py, px
    }
    uf.parent[py] = px
    if uf.rank[px] == uf.rank[py] {
        uf.rank[px]++
    }
}
```

### 并发哈希计算
使用 sync.WaitGroup 并发计算所有图片的哈希值，提高检测效率。

## 偏差记录

### 无偏差
计划执行完全按照预期，没有发现需要修复的 bug 或需要调整的设计。

## 测试覆盖

| 组件 | 测试数量 | 状态 |
|------|---------|------|
| HashService | 5 | ✅ 全部通过 |
| DuplicateRepository | 6 | ✅ 全部通过 |
| DuplicateService | 7 | ✅ 全部通过 |
| DuplicateHandler | 7 | ✅ 全部通过 |
| Union-Find | 1 | ✅ 全部通过 |

## 文件清单

### 新增文件 (9)
- `internal/service/hash_service.go` - 哈希计算服务
- `internal/service/hash_service_test.go` - 哈希服务测试
- `internal/service/duplicate_service.go` - 重复检测服务
- `internal/service/duplicate_service_test.go` - 检测服务测试
- `internal/repository/duplicate_repository.go` - 重复组仓储
- `internal/repository/duplicate_repository_test.go` - 仓储测试
- `internal/handler/duplicate_handler.go` - API 处理器
- `internal/handler/duplicate_handler_test.go` - API 测试
- `internal/domain/duplicate_group.go` - 数据模型

### 修改文件 (4)
- `internal/repository/schema.go` - 添加 duplicate_groups 和 duplicate_relations 表
- `internal/handler/routes.go` - 注册重复检测 API 路由
- `go.mod` - 添加 goimagehash 依赖
- `go.sum` - 依赖校验和

## 下一步

Phase 04 Plan 02 将实现：
- 图片搜索功能
- 与重复检测的集成
- 前端界面支持

## Self-Check: PASSED

- [x] 所有文件已创建/修改
- [x] 所有测试通过
- [x] 构建成功
- [x] 代码已提交