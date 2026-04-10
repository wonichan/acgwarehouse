# Service层日志补充计划

## 日志规范

**使用方式**: `log.Printf("key=value key2=value2 ...")`
**import**: 标准库 `"log"`
**格式原则**: 入口记入参、成功记结果、错误记详情（含error）

---

## 1. admin_service.go (14个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `GetSummary` | error | 获取task counts失败 | 中 |
| `GetTaskPlatformOverview` | entry | 开始获取概览 | 低 |
| | error | ListBatches失败 | 高 |
| `ClearTaskQueue` | entry | 清空任务队列 | 高 |
| | exit | 清空结果: cancelled_count=N | 高 |
| | error | 查询/取消任务失败 | 高 |
| `CancelTaskBatch` | entry | batch_id=N | 高 |
| | exit | 取消结果: cancelled_count=N | 高 |
| | error | 查询/取消失败 | 高 |
| `CancelTask` | entry | task_id=N | 高 |
| | exit | 取消结果 | 高 |
| | error | 查询/取消失败 | 高 |
| `GetJobs` | - | 纯查询跳过 | 无 |
| `GetTaskBatches` | - | 纯查询跳过 | 无 |
| `GetTasks` | - | 纯查询跳过 | 无 |
| `TriggerScan` | entry | 触发手动扫描 | 高 |
| | exit | job_id=N | 高 |
| | error | 添加job失败 | 高 |
| `RetryFailedJobs` | entry | 开始重试所有失败job | 高 |
| | exit | 重试总结果: retried_count=N | 高 |
| | error | listRetryableBatches失败 | 高 |
| `RetryFailedBatchTasks` | entry | batch_id=N | 高 |
| | exit | 重试结果: retried_count=N new_batch_id=N | 高 |
| | error | 批次/任务查询失败 | 高 |
| `RetryFailedTask` | entry | task_id=N | 高 |
| | exit | 重试结果 | 高 |
| | error | 查询/验证失败 | 高 |
| `PauseBackgroundTasks` | entry | 暂停后台任务 | 高 |
| `ResumeBackgroundTasks` | entry | 恢复后台任务 | 高 |
| `IsBackgroundRunning` | - | 纯getter跳过 | 无 |

---

## 2. task_platform_service.go (6个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `PlanBatch` | entry | source_type=N summary_label=N total_images=N | 高 |
| | exit | batch_id=N created_tasks=N skipped_tasks=N | 高 |
| | error | 创建batch/查询现有task失败 | 高 |
| `RefreshBatchStatus` | error | 刷新状态失败 | 中 |
| `QueueTask` | entry | task_type=N job_type=N | 高 |
| | exit | job_id=N | 高 |
| | error | 保存job/更新task失败 | 高 |
| `MarkJobRunning` | entry | job_id=N task_id=N | 高 |
| | error | 更新状态失败 | 高 |
| `MarkJobCompleted` | entry | job_id=N task_id=N | 高 |
| | error | 更新状态失败 | 高 |
| `MarkJobFailed` | entry | job_id=N task_id=N error_summary=X | 高 |
| | error | 更新状态失败 | 高 |

---

## 3. batch_service.go (5个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `BatchAddTags` | entry | image_count=N tag_count=N | 高 |
| | exit | success_count=N skipped=N | 高 |
| | error | 验证tag/保存图片失败 | 高 |
| `BatchRemoveTags` | entry | image_count=N tag_count=N | 高 |
| | exit | removed_count=N | 高 |
| | error | 删除关联失败 | 高 |
| `BatchMoveToCollection` | entry | image_count=N collection_id=N | 高 |
| | exit | moved_count=N | 高 |
| | error | 查询/移动失败 | 高 |
| `BatchRemoveFromCollection` | entry | image_count=N collection_id=N | 高 |
| | exit | removed_count=N cover_updated=bool | 高 |
| | error | 查询/移除失败 | 高 |
| `BatchDeleteImages` | entry | image_count=N | 高 |
| | exit | deleted_count=N | 高 |
| | error | 查询tag/删除图片失败 | 高 |

---

## 4. collection_service.go (11个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `CreateCollection` | entry | name=X | 高 |
| | exit | collection_id=N | 高 |
| | error | 查询重名/保存失败 | 高 |
| `GetCollection` | - | 纯查询跳过 | 无 |
| `ListCollections` | - | 纯查询跳过 | 无 |
| `UpdateCollection` | entry | id=N name=X | 高 |
| | exit | 更新成功 | 高 |
| | error | 查询/验证/更新失败 | 高 |
| `DeleteCollection` | entry | collection_id=N | 高 |
| | exit | 删除成功 | 高 |
| | error | 查询/删除失败 | 高 |
| `AddImageToCollection` | entry | collection_id=N image_id=N | 高 |
| | exit | 添加成功, cover_updated=bool | 高 |
| | error | 查询/添加/更新封面失败 | 高 |
| `RemoveImageFromCollection` | entry | collection_id=N image_id=N | 高 |
| | exit | 移除成功, cover_updated=bool | 高 |
| | error | 查询/移除/更新封面失败 | 高 |
| `SetCoverImage` | entry | collection_id=N image_id=N | 中 |
| | error | 更新封面失败 | 高 |
| `AutoUpdateCover` | entry | collection_id=N | 中 |
| | exit | 封面已更新/已清除 | 中 |
| | error | 查询/更新失败 | 高 |
| `GetCollectionImages` | - | 纯查询跳过 | 无 |
| `CountCollections` | - | 纯查询跳过 | 无 |

---

## 5. search_service.go (3个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `Search` | entry | query_len=N tag_count=N sort_by=X limit=N | 高 |
| | exit | result_count=N total=N | 高 |
| | error | 搜索失败 | 高 |
| `SearchByFilename` | entry | pattern=X limit=N | 中 |
| | exit | result_count=N total=N | 高 |
| | error | 搜索失败 | 高 |
| `ViewerWindow` | entry | selected_index=N limit=N | 中 |
| | exit | window_size=N total=N | 高 |
| | error | 请求超出范围/查询失败 | 高 |

---

## 6. scanner_service.go (1个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `Scan` | entry | roots=N workers=N | 高 |
| | exit | total_files=N imported=N skipped=N failed=N duration=N | 高 |
| | mid | 开始任务规划(已有日志通过result) | 中 |
| | error | 文件系统遍历/任务规划/入队失败 | 高 |

---

## 7. watcher_service.go (2个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `Start` | entry | 启动文件监控, roots=N | 高 |
| | error | 添加递归watcher失败 | 高 |
| `Stop` | entry | 停止文件监控 | 高 |

---

## 8. cos_service.go (2个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `Upload` | entry | filename=X size=X | 高 |
| | exit | 上传成功, url=X | 高 |
| | error | 上传失败 (已有wrapped error) | 高 |
| `DeleteByURL` | entry | url=X | 中 |
| | exit | 删除成功 | 中 |
| | error | 解析/删除失败 (已有wrapped error) | 高 |

---

## 9. metadata_service.go (2个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `IsImage` | - | 纯判断跳过 | 无 |
| `ExtractMetadata` | entry | path=X | 中 |
| | exit | width=N height=N format=X size=N | 中 |
| | error | 文件操作/解码失败 | 高 |

---

## 10. tag_governance_service.go (3个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `MergeTags` | entry | image_id=N tag_count=N source=ai/manual | 高 |
| | exit | 合并完成, tag_count=N | 高 |
| | error | 查询observation/标签查找/保存失败 | 高 |
| `RemovePendingAITags` | entry | image_id=N | 中 |
| | exit | 移除完成 | 中 |
| | error | 查询/删除失败 | 高 |
| `RemoveRejectedAITags` | entry | image_id=N | 中 |
| | exit | 移除完成 | 中 |
| | error | 查询/删除失败 | 高 |

---

## 11. tag_admin_service.go (5个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `ListGovernanceTags` | - | 纯查询跳过 | 无 |
| `MergeTags` | entry | source_id=N target_id=N | 高 |
| | exit | 合并完成, migrated_images=N migrated_aliases=N | 高 |
| | error | 事务/查询/迁移失败 | 高 |
| `GetDeletePreview` | - | 纯查询跳过 | 无 |
| `CleanupUnusedTags` | entry | tag_count=N | 高 |
| | exit | deleted=N blocked=N failed=N | 高 |
| | error | 删除失败(每tag的error已记录) | 中 |

---

## 12. task_read_service.go (2个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `ListBatches` | - | 纯查询+数据转换跳过 | 无 |
| `ListTasks` | - | 纯查询+数据转换跳过 | 无 |

---

## 13. monitoring_event_bus.go (4个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `Subscribe` | - | 内部事件机制跳过 | 无 |
| `Broadcast` | - | 内部事件机制跳过 | 无 |
| `Start` | entry | 启动监控事件总线, interval=N | 高 |
| `Stop` | entry | 停止监控事件总线 | 高 |

---

## 14. log_stream_service.go (4个方法)

| 方法 | 日志点 | 日志内容 | 优先级 |
|------|--------|---------|--------|
| `Start` | entry | 启动日志流服务, path=X | 高 |
| | error | 初始化watcher失败 | 高 |
| `Stop` | entry | 停止日志流服务 | 高 |
| `Subscribe` | - | 内部订阅机制跳过 | 无 |
| `Broadcast` | - | 内部广播机制跳过 | 无 |

---

## 执行策略

### 分批委托并行执行

**Batch 1** (高优先级 - 核心业务):
- admin_service.go
- task_platform_service.go
- batch_service.go
- search_service.go

**Batch 2** (高优先级 - 数据管理):
- collection_service.go
- scanner_service.go
- watcher_service.go

**Batch 3** (中优先级 - 辅助服务):
- cos_service.go
- metadata_service.go
- tag_governance_service.go
- tag_admin_service.go

**Batch 4** (中优先级 - 内部服务):
- monitoring_event_bus.go
- log_stream_service.go

### 日志格式统一模板

```go
// 入口日志
log.Printf("[service] method_name started: key1=%v key2=%v", val1, val2)

// 成功日志
log.Printf("[service] method_name completed: result_key1=%v result_key2=%v duration=%v", val1, val2, duration)

// 错误日志
log.Printf("[service] method_name failed: key1=%v error=%v", val1, err)
```

### 注意事项

1. **不改变业务逻辑** - 仅添加log.Printf调用
2. **保持现有错误处理** - 不在日志后吞掉错误
3. **敏感信息脱敏** - 不记录密码、密钥等
4. **耗时记录** - 对可能耗时的操作(Scan, Search, Batch操作)记录duration
5. **原子事务类操作** - 在事务前后各记一条, 不在事务内部记
6. **循环内不记日志** - 批量操作只记总体结果, 避免日志爆炸
