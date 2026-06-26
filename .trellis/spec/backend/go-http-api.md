# HTTP RESTful 路由与响应格式规范
AI 在 `handler/router/` 中定义 API 或设计 HTTP 交互时，必须完全符合以下契约。
## 1. 路由定义规范
- **大小写**：路由路径**全小写**。例如：`/api/v1/users`（禁止 `/api/Users`）。
- **单词分隔**：多个单词之间必须使用**中划线连字符 `-`**。例如：`/user-profiles`（禁止 `/user_profiles`）。
- **路径层级**：路径用 `/` 分割，总深度**严格控制在 5 层以内**。
- **版本控制**：URL 开头必须带有版本号，格式为 `v` + 数字。例如：`/v1/users`（禁止 `/users/v1`）。
## 2. 请求与响应格式约定
- **Content-Type**：必须强制为 `application/json`。
- **请求参数**：查询参数使用 `/users?page=1`；资源定位的路径参数使用 `/users/:id`。
- **Body 格式**：统一为标准 JSON 格式。**禁用 FormData**（普通文件上传除外）。
- **统一响应体结构**：所有的 HTTP 返回必须序列化为以下 Go 结构体：
```go

type Response struct {

    Code    int         `json:"code"`    // 业务状态码（非 HTTP 状态码）

    Data    interface{} `json:"data"`    // 成功时返回具体数据对象

    Msg     string      `json:"msg"`     // 错误时返回给客户端的提示消息

}

```
## 3. 状态码与业务 Code 映射矩阵
必须根据以下场景匹配对应的 HTTP 状态码和业务 Code：

| **业务场景**          | **HTTP 状态码** | **业务 Code 示例** | **Msg 规范说明**   |
| --------------------- | --------------- | ------------------ | ------------------ |
| **请求成功**          | 200             | 0                  | `""` (留空)        |
| **参数校验失败**      | 400             | 40001              | 具体的参数错误描述 |
| **认证失败/未登录**   | 401             | 40101              | 提示重新登录       |
| **权限不足/拒绝访问** | 403             | 40301              | 提示无权操作       |
| **目标数据不存在**    | 404             | 40401              | 例如："用户不存在" |
| **服务器内部错误**    | 500             | 50001              | 系统异常提示       |

## 4. 场景：图片查询 / 详情 / 搜索 / 软删除 API

### 1. Scope / Trigger
- 触发：新增图片公开查询、搜索、详情浏览事件、管理员软删除/恢复 API。
- 目标：前端可稳定消费图片元数据；标签/评分/收藏未实现前也保持响应字段稳定。

### 2. Signatures
- `GET /api/v1/images?filename=&tag=&sort=created_at|size|tag&order=asc|desc&page=&size=`。
- `GET /api/v1/images/:id`。
- `GET /api/v1/search?q=&page=&size=`。
- `DELETE /api/v1/images/:id`，需 `Auth + RequireAdmin`。
- `POST /api/v1/images/:id/restore`，需 `Auth + RequireAdmin`。
- DB：`image.status/deleted_at` 控制公开可见性；`image_event(image_id,user_id,type,value,created_at)` 记录浏览事件。

### 3. Contracts
- 所有列表响应必须使用 `{total,page,size,list}`，外层仍为 `Response{Code,Data,Msg}`。
- 图片列表项必须包含：`id, cos_key, filename, url, size, last_modified, width, height, category, avg_score, rating_count, favorite_count, view_count, created_at`。
- 详情响应必须包含：`image, tags, avg_score, rating_count, favorite_count, my_rating, is_collected, similar_images`。
- Phase 03 占位约定：标签/评分/收藏/相似图尚未落真实模块时，返回 `tags:[]`、`my_rating:null`、`is_collected:false`、`similar_images:[]`。
- `GET /api/v1/images/:id` 必须记录 `view` 事件；写入走缓冲器，关闭服务前必须 flush。
- 软删除图片不得出现在 list/detail/search；恢复后必须重新可见并重新写入搜索索引。

### 4. Validation & Error Matrix
- 非法 `:id` -> HTTP 400 / Code 40001。
- 图片不存在或已软删除 -> HTTP 404 / Code 40401。
- 未登录调用管理员路由 -> HTTP 401 / Code 40101。
- 非管理员调用管理员路由 -> HTTP 403 / Code 40301。
- 搜索索引失败或 DB 异常 -> HTTP 500 / Code 50001，由 handler 统一记录日志。

### 5. Good/Base/Bad Cases
- Good：`GET /images?filename=miku&page=1&size=20` 只返回 active 且 filename 匹配的图片，总数正确。
- Good：`DELETE /images/:id` 后该图片 detail 返回 404，search 不再返回；`POST /restore` 后重新出现。
- Base：`tag` 参数和 `sort=tag` 在标签模块前保持兼容，不提前创建标签表。
- Bad：详情缺少占位字段，导致前端在 Phase 04-06 前后响应结构变化。

### 6. Tests Required
- Repository：filename 过滤/计数、soft delete hide、restore show、事件批量写入并累加 `view_count`。
- Service：detail 占位字段、view event 记录、search 按索引顺序返回、search total 保持索引总数。
- Infra search：分页只返回当前页 ID，但 `total` 保持全部命中数。
- Route smoke（有测试框架时）：公开路由 200，管理员路由 401/403/200 分支。

### 7. Wrong vs Correct

#### Wrong
```go
// infra 不能反向依赖 service 查询类型。
import "github.com/yachiyo/acgwarehouse/internal/service"
```

#### Correct
```go
// 共享搜索契约放在 do 层，service 与 infra/search 都依赖更低层。
type ImageSearchQuery struct {
    Text string
    Page int
    Size int
}
```
