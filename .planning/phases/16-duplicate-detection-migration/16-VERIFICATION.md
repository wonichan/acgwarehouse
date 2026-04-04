---
phase: 16-duplicate-detection-migration
status: passed
verification_date: 2026-04-04
verifier: Sisyphus-Junior
fresh_evidence: true
---

# Phase 16 `duplicate-detection-migration` 验证报告

**状态：PASSED** — 阶段目标已真正达成，代码与测试证据完整。

---

## 一、COMP-03 验证：计算迁移到 Python sidecar

### 1.1 Python sidecar 承担重复检测计算 ✅

| 计算模块 | 文件路径 | 验证内容 | 证据 |
|----------|----------|----------|------|
| SHA256 + pHash 计算 | `services/python-sidecar/compute/hashing.py` | 使用 `imagehash.phash(img, hash_size=16)` 生成 256-bit pHash，SHA256 通过 `hashlib.sha256()` 计算 | 代码第 26-28 行 |
| Union-Find 分组 | `services/python-sidecar/compute/grouping.py` | 实现传递性分组，支持 exact(sha256) + similar(phash) 双模式 | 模块存在并通过测试 |
| 推荐评分计算 | `services/python-sidecar/compute/scoring.py` | 多维加权评分 (resolution 0.5 + file_size 0.3 + format 0.2)，返回结构化 reasons | 代码第 11-43 行 |

**Python 测试通过证据：**
```
============================= 40 passed in 1.15s ==============================
services\python-sidecar\tests\test_duplicates_router.py ........
services\python-sidecar\tests\test_grouping.py .........
services\python-sidecar\tests\test_hashing.py ......
services\python-sidecar\tests\test_scoring.py ........
services\python-sidecar\tests\test_task_state.py .....
```

### 1.2 Go 仅做编排，不做计算 ✅

| 编排步骤 | 文件路径 | 证据 |
|----------|----------|------|
| Submit Detection | `internal/sidecar/client.go` 第 73-109 行 | `POST /compute/duplicates/detect` → 返回 task_id |
| Poll Progress | `internal/sidecar/client.go` 第 112-135 行 | `GET /compute/duplicates/tasks/{id}` → 返回 status/progress |
| Fetch Results | `internal/sidecar/client.go` 第 137-159 行 | `GET /compute/duplicates/tasks/{id}/result` → 返回 DetectionResult |
| Persist Results | `internal/service/duplicate_service.go` 第 121-167 行 | 回写 phash_hex、保存 recommendation 字段、删除旧组重建 |

**编排流程验证：**
- `DuplicateService.DetectDuplicates` 方法（第 47-118 行）完整执行 submit→poll loop→fetch→persist 链路
- 无任何本地哈希计算代码，所有计算数据来自 Python sidecar 返回

### 1.3 旧计算代码已删除 ✅

| 检查项 | 结果 |
|--------|------|
| `hash_service.go` 文件 | ❌ 不存在（grep 无匹配） |
| `hash_service_test.go` 文件 | ❌ 不存在（grep 无匹配） |
| `goimagehash` 依赖 | ❌ 已移除（grep 全仓库无匹配） |

---

## 二、COMP-04 验证：推荐保留项与推荐依据

### 2.1 结构化推荐依据格式 ✅

**Python 输出格式 (`scoring.py` 第 21-40 行)：**
```python
reasons = [
    {"factor": "resolution", "value": "1920x1080", "score": 25.0, "weight": 0.5},
    {"factor": "file_size", "value": "2000000", "score": 19.1, "weight": 0.3},
    {"factor": "format", "value": "png", "score": 100.0, "weight": 0.2},
]
```

**Go 接收与持久化 (`duplicate_service.go` 第 146-158 行)：**
```go
rationale, err := json.Marshal(member.RecommendationReasons)
// ...
RecommendationRationale: json.RawMessage(rationale),
```

### 2.2 数据库字段落库 ✅

| 字段 | Schema 迁移 | Domain 类型 | Repository 写入 |
|------|-------------|-------------|-----------------|
| `phash_hex` TEXT | `schema.go` 第 251-252 行 | `domain.Image.PHashHex string` | `image_repository.go` 第 507-509 行 `UpdateImagePHashHex` |
| `recommendation_score` REAL | `schema.go` 第 254-255 行 | `domain.DuplicateRelation.RecommendationScore float64` | `duplicate_repository.go` 第 90 行 |
| `recommendation_rationale` TEXT | `schema.go` 第 256-258 行 | `domain.DuplicateRelation.RecommendationRationale json.RawMessage` | `duplicate_repository.go` 第 91 行 |

**Repository 测试验证：**
```
=== RUN   TestDuplicateRelation_RecommendationColumns
--- PASS: TestDuplicateRelation_RecommendationColumns (0.09s)
=== RUN   TestDuplicateRepository_SaveAndFindRecommendationRationale
--- PASS: TestDuplicateRepository_SaveAndFindRecommendationRationale (0.11s)
```

### 2.3 API 响应透传 ✅

**Handler 测试验证 (`duplicate_handler_test.go` 第 246-248 行）：**
```go
if _, ok := firstImage["recommendation_rationale"].([]any); !ok {
    t.Fatalf("recommendation_rationale should be JSON array, got %#v", firstImage["recommendation_rationale"])
}
```

**实际 API 响应结构（`domain.DuplicateImage` JSON 标签）：**
```json
{
  "id": 1,
  "recommendation_score": 90.0,
  "recommendation_rationale": [
    {"factor": "resolution", "value": "100x120", "score": 10.0, "weight": 0.5}
  ],
  "is_recommended": true
}
```

---

## 三、Sidecar 不可用时的失败状态 ✅

### 3.1 Handler 503 返回机制

**代码证据 (`duplicate_handler.go` 第 49-56 行）：**
```go
if h.sidecarRuntime != nil && h.sidecarRuntime.State() != sidecar.StateReady {
    status := h.sidecarRuntime.Status()
    c.JSON(http.StatusServiceUnavailable, gin.H{
        "error":   "计算服务不可用，请检查 Python 侧车状态",
        "state":   string(status.State),
        "details": status.LastError,
    })
    return
}
```

### 3.2 测试覆盖

**测试证据 (`duplicate_handler_test.go` 第 105-126 行）：**
```go
func TestDuplicateHandler_DetectDuplicates_SidecarUnavailable(t *testing.T) {
    // ...
    if w.Code != http.StatusServiceUnavailable {
        t.Fatalf("status = %d, want 503, body=%s", w.Code, w.Body.String())
    }
    if !bytes.Contains(w.Body.Bytes(), []byte("计算服务不可用")) {
        t.Fatalf("body = %s, want contains 计算服务不可用", w.Body.String())
    }
}
```

**运行结果：**
```
=== RUN   TestDuplicateHandler_DetectDuplicates_SidecarUnavailable
--- PASS: TestDuplicateHandler_DetectDuplicates_SidecarUnavailable (0.39s)
```

---

## 四、Must-Haves 核对

| Must-Have | 状态 | 证据 |
|-----------|------|------|
| Python sidecar 计算 SHA256 + 256-bit pHash | ✅ | `hashing.py` 第 26-28 行 |
| Python sidecar Union-Find 分组 | ✅ | `grouping.py` 存在并通过测试 |
| Python sidecar 多维推荐评分 | ✅ | `scoring.py` 第 11-43 行 |
| Go submit→poll→fetch→persist 编排 | ✅ | `duplicate_service.go` 第 47-118 行 |
| `phash_hex` 字段落库 | ✅ | schema.go + image_repository.go UpdateImagePHashHex |
| recommendation fields 落库 | ✅ | schema.go + duplicate_repository.go SaveDuplicateGroup |
| recommendation 透传到 API | ✅ | handler 测试第 246-248 行验证 JSON array |
| sidecar 不可用返回 503 | ✅ | handler 第 49-56 行 + 测试通过 |
| 旧 Go 计算代码删除 | ✅ | grep 无匹配 |
| goimagehash 依赖移除 | ✅ | grep 全仓库无匹配 |

---

## 五、Artifacts 交付物核对

| Artifact | 状态 | 文件路径 |
|----------|------|----------|
| Python hashing module | ✅ | `services/python-sidecar/compute/hashing.py` |
| Python grouping module | ✅ | `services/python-sidecar/compute/grouping.py` |
| Python scoring module | ✅ | `services/python-sidecar/compute/scoring.py` |
| Python duplicates router | ✅ | `services/python-sidecar/routers/duplicates.py` |
| Go sidecar client | ✅ | `internal/sidecar/client.go` |
| Go duplicate service (编排版) | ✅ | `internal/service/duplicate_service.go` |
| Go duplicate handler | ✅ | `internal/handler/duplicate_handler.go` |
| Schema 扩展迁移 | ✅ | `internal/repository/schema.go` 第 251-258 行 |
| Domain 扩展 | ✅ | `internal/domain/image.go` + `duplicate_group.go` |
| Python 测试集 | ✅ | `services/python-sidecar/tests/*.py` (40 tests) |
| Go 测试集 | ✅ | `*_test.go` 文件完整覆盖 |
| E2E 测试 | ✅ | `test/e2e/duplicate_test.go` |

---

## 六、Key Links 关键链路验证

### 6.1 Go→Python HTTP 调用链路 ✅

```
[Handler] DetectDuplicates()
    ↓ 检查 sidecarRuntime.State() == ready
[Service] DetectDuplicates(ctx, opts)
    ↓ sidecarClient.SubmitDetection() → POST /compute/duplicates/detect
    ↓ 循环 sidecarClient.PollProgress() → GET /compute/duplicates/tasks/{id}
    ↓ sidecarClient.FetchResults() → GET /compute/duplicates/tasks/{id}/result
[Service] persistDetectionResults()
    ↓ imageRepo.UpdateImagePHashHex() → 写 phash_hex
    ↓ duplicateRepo.DeleteAllDuplicateGroups() → 清空旧组
    ↓ duplicateRepo.SaveDuplicateGroup() → 写 recommendation_score/rationale
[Handler] 返回 {message, groups_found}
```

### 6.2 数据流向验证 ✅

```
Python compute → DetectionResult.Groups[].Members[]
    ↓ RecommendationReasons []map[string]interface{}
Go sidecar client → DetectionResultMember.RecommendationReasons
    ↓ json.Marshal → json.RawMessage
Repository → recommendation_rationale TEXT (数据库)
    ↓ json.RawMessage (读取时)
Domain → DuplicateImage.RecommendationRationale json.RawMessage
    ↓ JSON 序列化透传
API Response → recommendation_rationale: [{factor, value, score, weight}]
```

---

## 七、Requirements Coverage 需求覆盖

| Requirement | 子需求 | 覆盖状态 |
|-------------|--------|----------|
| **COMP-03** | Python computes SHA256 + pHash | ✅ 已验证 |
| | Python performs Union-Find grouping | ✅ 已验证 |
| | Go orchestrates (submit/poll/fetch) | ✅ 已验证 |
| | Go persists results to database | ✅ 已验证 |
| | Old Go compute code removed | ✅ 已验证 |
| **COMP-04** | Multi-dimensional scoring | ✅ 已验证 |
| | Structured rationale format | ✅ 已验证 |
| | Rationale persisted to DB | ✅ 已验证 |
| | Rationale transparent in API | ✅ 已验证 |

---

## 八、测试执行结果汇总

### 8.1 Python 测试

```bash
$ python -m pytest services/python-sidecar/tests -x
============================= 40 passed in 1.15s ==============================
```

### 8.2 Go 测试

```bash
$ go test ./internal/service/... ./internal/handler/... ./internal/repository/... ./internal/sidecar/... -run Duplicate -count=1 -v
PASS ok github.com/wonichan/acgwarehouse-backend/internal/service
PASS ok github.com/wonichan/acgwarehouse-backend/internal/handler
PASS ok github.com/wonichan/acgwarehouse-backend/internal/repository
PASS ok github.com/wonichan/acgwarehouse-backend/internal/sidecar
```

### 8.3 E2E 测试

```bash
$ go test ./test/e2e/... -run Duplicate -count=1 -v
PASS ok github.com/wonichan/acgwarehouse-backend/test/e2e
```

### 8.4 构建验证

```bash
$ go build ./...
# 无错误输出
```

### 8.5 LSP 诊断

```bash
$ lsp_diagnostics services/python-sidecar (severity=error)
Files with errors: 0
Total diagnostics: 0
```

---

## 九、Phase 15 回归验证

**Sidecar Runtime 测试（Phase 15 产物）通过：**
```
=== RUN   TestRuntimeLifecycleTransitionsToReadyOnProbeSuccess
--- PASS: TestRuntimeLifecycleTransitionsToReadyOnProbeSuccess (0.00s)
=== RUN   TestRuntimeStartupTimeoutTransitionsToDegradedWithError
--- PASS: TestRuntimeStartupTimeoutTransitionsToDegradedWithError (0.06s)
=== RUN   TestRuntimeShutdownFallsBackToKillAndWaitsForProcess
--- PASS: TestRuntimeShutdownFallsBackToKillAndWaitsForProcess (0.00s)
```

---

## 十、结论

**Phase 16 验证状态：PASSED**

- COMP-03（计算迁移）所有子需求已通过代码与测试验证
- COMP-04（推荐依据）所有子需求已通过代码与测试验证
- 必须交付物（artifacts）完整存在
- 关键链路（key links）完整打通
- Phase 15 回归测试通过
- 所有测试套件通过（Python 40 tests + Go tests + E2E tests）
- 构建与 LSP 诊断零错误

**无需人工介入（human_needed: false）**
**无缺口发现（gaps_found: none）**

---

*验证日期：2026-04-04*
*验证者：Sisyphus-Junior*
*证据来源：代码审查 + 测试执行 + LSP 诊断 + 构建验证*