# Quick Task 4: 添加image_imported处理器

**任务编号:** 4  
**任务描述:** 选择方案二：添加image_imported处理器来自动触发缩略图生成任务  
**创建日期:** 2026-03-19  
**状态:** Completed

---

## 任务清单

### 任务 1: 分析现有代码 ✅
**文件:**
- `cmd/server/main.go`
- `internal/worker/job_manager.go`
- `internal/service/scanner_service.go`

**动作:**
1. ✅ 已确认扫描导入时创建的是 `image_imported` 任务
2. ✅ 已确认 `thumbnail_generate` 处理器已注册但从未被创建任务
3. ✅ 已确定需要添加 `image_imported` 处理器来桥接两者

**完成标志:**
完成代码分析，确定方案

---

### 任务 2: 实现image_imported处理器 ✅
**文件:**
- `cmd/server/main.go` (已修改)

**动作:**
1. ✅ 添加 `encoding/json` import
2. ✅ 在 `thumbnail_generate` 处理器注册后，添加 `image_imported` 处理器
3. ✅ 处理器逻辑：
   - 解析 payload 获取 image_id 和 path
   - 创建 `thumbnail_generate` 任务
   - 添加到任务队列
4. ✅ 添加日志输出便于调试

**验证:**
- [x] 代码可以编译通过
- [x] 处理器正确注册
- [x] 错误处理完善

**完成标志:**
修改 `cmd/server/main.go`，添加 image_imported 处理器

---

## 修改详情

### 修改的文件

**cmd/server/main.go:**

1. 添加 import:
```go
"encoding/json"
```

2. 在缩略图处理器注册代码块中添加:
```go
// 注册 image_imported 处理器 - 自动触发缩略图生成任务
jobManager.RegisterHandler("image_imported", func(ctx context.Context, id int64, payload string) error {
    // 解析 payload 获取 image_id 和 path
    var p struct {
        ImageID int64  `json:"image_id"`
        Path    string `json:"path"`
    }
    if err := json.Unmarshal([]byte(payload), &p); err != nil {
        return fmt.Errorf("解析 image_imported payload 失败: %w", err)
    }
    
    // 创建缩略图生成任务
    thumbnailPayload, err := json.Marshal(map[string]interface{}{
        "image_id": p.ImageID,
        "path":     p.Path,
    })
    if err != nil {
        return err
    }
    
    // 添加到任务队列
    _, err = jobManager.AddJob(ctx, "thumbnail_generate", string(thumbnailPayload))
    if err != nil {
        return fmt.Errorf("添加缩略图生成任务失败: %w", err)
    }
    
    log.Printf("已为新导入的图片 %d 创建缩略图生成任务", p.ImageID)
    return nil
})
log.Printf("已注册 image_imported 处理器 - 将自动触发缩略图生成")
```

---

## 工作流程

修改后，当服务器启动时：

1. **扫描导入图片**
   ```powershell
   go run cmd/scan/main.go
   ```
   - 创建 `image_imported` 任务 → 数据库

2. **服务器启动**
   ```powershell
   go run cmd/server/main.go
   ```
   - 注册 `thumbnail_generate` 处理器
   - 注册 `image_imported` 处理器 ← 新增

3. **任务处理**
   - JobManager 从数据库加载 `image_imported` 任务
   - 调用 `image_imported` 处理器
   - 处理器创建 `thumbnail_generate` 任务
   - JobManager 处理 `thumbnail_generate` 任务
   - 生成缩略图并上传 COS

---

## 注意事项

1. **COS 配置**: 需要正确配置腾讯云 COS 才能上传缩略图
2. **任务队列**: 确保 JobManager 正常运行
3. **日志查看**: 启动服务器时会看到 "已注册 image_imported 处理器" 日志
4. **错误处理**: 处理器包含完善的错误处理和日志输出

---

## 计划元数据

```yaml
must_haves:
  truths:
    - image_imported 处理器已正确注册
    - 处理器能正确解析 payload
    - 处理器能创建 thumbnail_generate 任务
    - 代码可以编译通过
  artifacts:
    - cmd/server/main.go (已修改)
  key_links:
    - internal/worker/job_manager.go
    - internal/service/scanner_service.go
    - internal/worker/thumbnail_handler.go
```

---

*计划创建时间: 2026-03-19*
