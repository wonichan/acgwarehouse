# Plan 005: Admin Dashboard Chinese Localization

## Overview

将后台管理界面从英文本地化为中文，提升中文用户的使用体验。

## Files to Modify

- `web/admin/index.html` - 静态 UI 文本
- `web/admin/app.js` - 动态提示和状态文本

## Translation Tasks

### Task 1: Localize index.html

**Target:** `web/admin/index.html`

**Translations:**

| English | Chinese |
|---------|---------|
| ACGWarehouse Admin Dashboard | ACGWarehouse 管理后台 |
| ACGWarehouse Admin | ACGWarehouse 管理后台 |
| Refresh | 刷新 |
| Logout | 退出登录 |
| Service Status | 服务状态 |
| Health | 健康状态 |
| Loading... | 加载中... |
| Environment | 运行环境 |
| Task Queue | 任务队列 |
| Total | 总计 |
| Ready | 待处理 |
| Running | 运行中 |
| Finished | 已完成 |
| Failed | 失败 |
| Pause Queue | 暂停队列 |
| Resume Queue | 恢复队列 |
| Retry Failed | 重试失败任务 |
| Library Scale | 图库规模 |
| Total Images | 图片总数 |
| Total Tags | 标签总数 |
| Collections | 收藏夹 |
| Configuration | 系统配置 |
| AI API Key | AI API 密钥 |
| COS Storage | COS 存储 |
| Admin Username | 管理员账号 |
| Trigger Scan | 触发扫描 |
| Recent Jobs | 最近任务 |
| ID | 任务ID |
| Type | 类型 |
| Status | 状态 |
| Progress | 进度 |
| Created | 创建时间 |
| Error | 错误信息 |
| Recent Errors | 最近错误 |
| No errors to display | 暂无错误 |

**Additional changes:**
- Update `<html lang="en">` to `<html lang="zh-CN">`

### Task 2: Localize app.js

**Target:** `web/admin/app.js`

**Translations:**

| English | Chinese |
|---------|---------|
| Authentication required. Please login. | 需要身份验证，请先登录 |
| Summary load error | 加载概览数据失败 |
| Error | 错误 |
| Jobs load error | 加载任务列表失败 |
| Failed to load jobs | 加载任务失败 |
| unknown | 未知 |
| healthy | 正常 |
| unhealthy | 异常 |
| Configured | 已配置 |
| Not set | 未设置 |
| (none) | (无) |
| No jobs found | 暂无任务 |
| Job #{id} | 任务 #{id} |
| Job queue paused | 任务队列已暂停 |
| Failed to pause queue | 暂停队列失败 |
| Job queue resumed | 任务队列已恢复 |
| Failed to resume queue | 恢复队列失败 |
| Failed jobs queued for retry | 失败任务已加入重试队列 |
| Failed to retry jobs | 重试任务失败 |
| Scan triggered successfully | 扫描任务已触发 |
| Failed to trigger scan | 触发扫描失败 |

## Verification Steps

1. 启动服务: `go run cmd/server/main.go`
2. 访问: http://localhost:8080/admin
3. 验证所有文本已显示为中文
4. 验证按钮操作后的提示消息为中文
5. 验证状态标签（健康/异常等）显示正确

## Success Criteria

- [ ] 所有静态文本已翻译为中文
- [ ] 所有动态提示消息已翻译为中文
- [ ] HTML lang 属性已更新为 zh-CN
- [ ] 界面显示正常，无乱码
- [ ] 功能操作正常，提示消息正确
