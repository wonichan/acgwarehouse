# Quick Task 32: 修复标签重命名后FTS索引未同步的bug

**日期:** 2026-03-20
**提交:** a64474c

## 问题描述

用户报告：AI 生成标签 "开叉旗袍" → 确认标签 → 标签管理中重命名为 "旗袍" → 搜索 "旗袍" 找不到图片。

## 根本原因

当标签重命名时，`tags` 表中的 `preferred_label` 更新了，但 `images_fts` 全文索引中的 `tags` 字段没有同步更新，导致搜索时使用新标签名找不到图片。

## 修复内容

### 1. 代码修复

**`internal/repository/image_tag_repository.go`:**
- 添加 `SyncFTSForTag(ctx context.Context, tagID int64) error` 方法
- 该方法查找所有使用该标签的图片，并重新生成其 FTS 索引

**`internal/handler/tag_handler.go`:**
- 修改 `UpdateTag` 函数
- 当 `preferred_label` 改变时，调用 `SyncFTSForTag` 同步 FTS 索引

### 2. 数据修复

运行临时脚本修复数据库中的脏数据：
- 标签 ID 130（原 "开叉旗袍"，现 "旗袍"）
- 关联图片 ID 594 的 FTS 索引已更新

## 验证

- 所有相关测试通过
- 代码编译成功
- 数据库脏数据已修复

## 影响范围

- 标签管理功能
- 全文搜索功能