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

## 4. 场景：用户账户资料 / 偏好 / 密码 API

### 1. Scope / Trigger
- 触发：登录用户需要读取/更新个人资料、偏好设置和登录密码，涉及 HTTP API、service 校验、SQLite `user` 表和前端账户中心。
- 目标：注册/登录/当前用户/资料保存/偏好保存/密码修改形成真实闭环，刷新页面后资料仍由后端恢复。

### 2. Signatures
- `POST /api/v1/users/register`，Request：`{"username": string, "password": string}`。
- `POST /api/v1/users/login`，Request：`{"username": string, "password": string}`，Response：`{"token": string}`。
- `GET /api/v1/users/me`，需 `Auth`。
- `PUT /api/v1/users/me`，需 `Auth`，Request：`{"nickname": string, "favorite_tags": string, "bio": string, "public_profile": bool, "email_notifications": bool, "sync_collections": bool}`。
- `PUT /api/v1/users/password`，需 `Auth`，Request：`{"old_password": string, "new_password": string}`。
- DB：`user` 必须持久化 `nickname/favorite_tags/bio/public_profile/email_notifications/sync_collections/password_hash`。

### 3. Contracts
- `UserResponse` 必须包含：`id, username, role, created_at, nickname, favorite_tags, bio, public_profile, email_notifications, sync_collections`，不得暴露 `password_hash`。
- 新注册用户默认：`nickname=username`，`public_profile=true`，`email_notifications=true`，`sync_collections=true`。
- profile 字段是面向中文/日文/韩文用户的字符限制，不能在 DTO 上用 `vd:"len($)"` 这类字节长度校验拦截；handler 只 bind，service/domain 用 `utf8.RuneCountInString()` 校验字符数。
- service 更新 profile 前必须 `strings.TrimSpace`；repository 只保存已规范化后的字段。
- 密码修改必须先校验旧密码 bcrypt hash，再写入新 bcrypt hash；成功后当前 JWT 会话可继续使用。

### 4. Validation & Error Matrix
- 未登录调用 `/users/me` / `/users/password` -> HTTP 401 / Code 40101。
- `username` 长度不在 3-32 或 `password/new_password` 少于 6 -> HTTP 400 / Code 40001。
- `nickname` trim 后为空或超过 20 个字符 -> HTTP 400 / Code 40001。
- `favorite_tags` 超过 120 个字符、`bio` 超过 200 个字符 -> HTTP 400 / Code 40001。
- 旧密码不匹配 -> HTTP 401 / Code 40101。
- 用户不存在 -> HTTP 404 / Code 40401。

### 5. Good/Base/Bad Cases
- Good：20 个 CJK 字符的 nickname 可通过 handler bind，并由 service 按 rune 计数接受。
- Good：`PUT /users/me` 保存 `email_notifications=false` 后，下一次 `GET /users/me` 返回 false。
- Good：密码修改后，旧密码登录失败，新密码登录成功。
- Base：空 `favorite_tags` / `bio` 合法；空 nickname 不合法。
- Bad：DTO 使用 `vd:"len($) <= 20"` 校验 nickname，导致 20 个中文字符因字节数超过 20 被错误拒绝。

### 6. Tests Required
- Service：profile trim、CJK 字符边界、nickname/tags/bio 超限、旧密码错误、新密码过短、密码 hash 更新。
- Repository：profile/preference 字段持久化、password_hash 持久化。
- Route smoke：`GET /users/me` 返回扩展字段；`PUT /users/me` 支持 CJK 边界；`PUT /users/password` 成功/旧密码错误/新密码过短。
- Browser/E2E：注册自动登录、保存资料、保存偏好、修改密码、退出后用新密码登录并恢复资料。

### 7. Wrong vs Correct

#### Wrong
```go
type UserProfileUpdateRequest struct {
    Nickname string `json:"nickname" vd:"len($) <= 20"`
}
```

#### Correct
```go
type UserProfileUpdateRequest struct {
    Nickname string `json:"nickname"`
}

if profile.Nickname == "" || utf8.RuneCountInString(profile.Nickname) > maxNicknameLength {
    return do.User{}, pkgerrors.WithMessage(ErrInvalidUserInput, "validate nickname")
}
```

## 5. 场景：图片查询 / 详情 / 搜索 / 软删除 API

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
- `GET /api/v1/images/:id` 的本次响应必须立即体现本次浏览：`data.image.view_count` 应为读取到的持久化快照值 `+1`，但不得因此绕过 `ViewBuffer` 改成每次同步写库。
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
- Good：`GET /api/v1/images/:id` 返回的 `data.image.view_count` 立即包含本次详情浏览，同时只通过 `ViewBuffer` 记录事件，最终异步累加 DB。
- Base：`tag` 参数和 `sort=tag` 在标签模块前保持兼容，不提前创建标签表。
- Bad：详情缺少占位字段，导致前端在 Phase 04-06 前后响应结构变化。
- Bad：详情接口只返回记录前快照的 `view_count`，导致用户点击进入详情后当前页面看不到本次浏览。
- Bad：为让 `view_count` 立即变化而在 detail 请求路径同步更新 DB，绕过浏览事件缓冲器。

### 6. Tests Required
- Repository：filename 过滤/计数、soft delete hide、restore show、事件批量写入并累加 `view_count`。
- Service：detail 占位字段、view event 记录、detail 响应 `view_count` 立即包含本次浏览、search 按索引顺序返回、search total 保持索引总数。
- Infra search：分页只返回当前页 ID，但 `total` 保持全部命中数。
- Route smoke（有测试框架时）：公开路由 200，管理员路由 401/403/200 分支。

### 7. Wrong vs Correct

#### Wrong
```go
// infra 不能反向依赖 service 查询类型。
import "github.com/yachiyo/acgwarehouse/internal/service"
```

#### Wrong
```go
// 记录 view 后仍用旧快照构造响应，当前详情页不会立即显示本次浏览。
if err := views.RecordView(ctx, event); err != nil {
    return dto.ImageDetailResponse{}, err
}
return newDetailResponse(image)
```

#### Correct
```go
// 持久化仍交给 ViewBuffer；本次响应只补展示增量。
if err := views.RecordView(ctx, event); err != nil {
    return dto.ImageDetailResponse{}, err
}
image.ViewCount++
return newDetailResponse(image)
```

## 6. 场景：图片评分 API

### 1. Scope / Trigger
- 触发：新增用户对图片评分能力，涉及 HTTP API、service 校验、SQLite `rating` 表、`image` 冗余聚合字段和 `image_event` 行为流水。
- 目标：登录用户可对单张图片提交 0-100 整数评分；重复评分覆盖；前端可立即获得最新均分与评分人数。

### 2. Signatures
- API：`PUT /api/v1/images/:id/rating`，需 `Auth`。
- Request：`{"score": <int>}`。
- Response：`{"image_id": <int64>, "user_id": <int64>, "score": <int>, "avg_score": <float64>, "rating_count": <int64>, "updated_at": <RFC3339>}`。
- DB：`rating(user_id,image_id,score,created_at,updated_at)`，主键/唯一键为 `(user_id,image_id)`；`image.avg_score` / `image.rating_count` 为冗余聚合；`image_event(type='rating', value=score)` 记录评分事件。

### 3. Contracts
- `score` 必须为整数且范围 `0 <= score <= 100`；0 和 100 均为合法边界值。
- 同一 `(user_id,image_id)` 重复提交评分时覆盖原 `score`，不得新增第二条 rating。
- rating upsert、`image.avg_score` / `image.rating_count` 重算、rating 事件写入必须在同一写事务中完成。
- handler 只接收/返回 `dto`，service 使用 `do.Rating`，repository 内部使用 `po.Rating`；禁止 `po` 穿透 HTTP 响应。

### 4. Validation & Error Matrix
- 未登录或 JWT 非法 -> HTTP 401 / Code 40101。
- 非法 `:id` -> HTTP 400 / Code 40001。
- `score < 0` 或 `score > 100` / body 绑定失败 -> HTTP 400 / Code 40001。
- 图片不存在或已软删除 -> HTTP 404 / Code 40401。
- DB 事务、聚合重算或事件写入失败 -> HTTP 500 / Code 50001，由 handler 统一记录日志。

### 5. Good/Base/Bad Cases
- Good：用户 A 对图片 1 评分 80 后，`rating_count=1`、`avg_score=80`，并产生一条 `rating` 事件。
- Good：用户 A 再次对图片 1 评分 60 后，rating 表仍只有一条 `(A,1)`，`rating_count` 不增加，`avg_score=60`。
- Base：用户 A 评分 0、用户 B 评分 100 后，`rating_count=2`、`avg_score=50`。
- Bad：先 insert rating 再在事务外更新 image 聚合，导致聚合与评分表短暂不一致。

### 6. Tests Required
- Repository：首次 upsert 创建 rating、重复 upsert 覆盖、多人评分均分/人数正确、写入 rating 事件。
- Service：0/100 边界合法；-1/101 返回 `ErrInvalidRatingInput`；重复评分不重复计数。
- Route smoke：未登录返回 401；非法 score 返回 400；合法请求返回最新 `avg_score` / `rating_count`。

### 7. Wrong vs Correct

#### Wrong
```go
// handler 直接构造 po.Rating 并写库，且聚合在事务外更新。
func (h *RatingHandler) Rate(c context.Context, ctx *app.RequestContext) {
    rating := po.Rating{ImageID: id, UserID: userID, Score: input.Score}
    db.Create(&rating)
    db.Model(&po.Image{}).Update("avg_score", input.Score)
}
```

#### Correct
```go
// handler 只做边界解析，事务与聚合由 repository 经 service 触发。
result, err := h.ratingService.Upsert(c, do.Rating{
    ImageID: imageID,
    UserID:  userID,
    Score:   input.Score,
})
```

## 7. 场景：收藏夹 API

### 1. Scope / Trigger
- 触发：新增多收藏夹能力，涉及 HTTP API、owner/visibility 权限、SQLite `collection` / `collection_item` 表、`image.favorite_count` 去重聚合和 `image_event` 行为流水。
- 目标：用户可管理多个命名收藏夹；公开收藏夹可被任何人浏览；私有收藏夹仅 owner 可见；图片收藏数按去重用户数维护。

### 2. Signatures
- `GET /api/v1/collections`，需 `Auth`，返回当前用户收藏夹列表。
- `POST /api/v1/collections`，需 `Auth`，Request：`{"name": <string>, "visibility": "private"|"public"}`。
- `GET /api/v1/collections/:id`，公开收藏夹任意访问；私有仅 owner。
- `PUT /api/v1/collections/:id` / `DELETE /api/v1/collections/:id`，需 `Auth` 且 owner。
- `POST /api/v1/collections/:id/items`，需 owner，Request：`{"image_id": <int64>}`。
- `DELETE /api/v1/collections/:id/items/:imageId`，需 owner。
- DB：`collection(id,user_id,name,visibility,created_at,updated_at)`；`collection_item(collection_id,image_id,created_at)`，同夹同图唯一；`image.favorite_count` 为去重用户数；`image_event(type='favorite', value=1|-1)` 记录收藏/取消收藏事件。

### 3. Contracts
- `visibility` 缺省为 `private`；只允许 `private` / `public`。
- 同一收藏夹内同一图片只能存在一条 `collection_item`；重复加入应幂等，不重复增加 `favorite_count`。
- `favorite_count` 统计某图片被多少个不同用户收藏，而不是被多少个收藏夹收藏。
- 用户删除收藏夹时，必须按被删除 item 的图片逐一重算该 owner 是否仍收藏该图；只有 owner 对该图无剩余收藏时才递减 `favorite_count` 并发 `favorite value=-1` 事件。
- 删除收藏夹、删除 items、更新 `favorite_count`、写 favorite event 必须在同一写事务内完成。
- handler 只接收/返回 `dto`，service 使用 `do.Collection` / `do.CollectionItem`，repository 内部使用 `po`；禁止 `po` 穿透 HTTP 响应。

### 4. Validation & Error Matrix
- 未登录管理收藏夹或 item -> HTTP 401 / Code 40101。
- 非法 `:id` / `:imageId` / body 绑定失败 / 非法 visibility -> HTTP 400 / Code 40001。
- 收藏夹不存在，或私有收藏夹被非 owner 读取 -> HTTP 404 / Code 40401（避免泄露私有资源存在性）。
- 非 owner 更新/删除/添加/移除 item -> HTTP 403 / Code 40301。
- DB 事务、聚合重算或事件写入失败 -> HTTP 500 / Code 50001，由 handler 统一记录日志。

### 5. Good/Base/Bad Cases
- Good：用户 A 将图片 1 加入第一个收藏夹，`favorite_count +1` 且发 `favorite value=1`。
- Good：用户 A 又将图片 1 加入第二个收藏夹，`favorite_count` 不变，不重复发 +1 事件。
- Good：用户 A 删除第一个收藏夹但第二个收藏夹仍含图片 1，`favorite_count` 不变。
- Good：用户 A 删除最后一个包含图片 1 的收藏夹，`favorite_count -1` 且发 `favorite value=-1`。
- Base：公开收藏夹未登录可读；私有收藏夹非 owner 读取按不存在处理。
- Bad：按 `collection_item` 行数维护 `favorite_count`，导致同一用户多收藏夹收藏同图时重复计数。

### 6. Tests Required
- Repository：同夹同图唯一；同用户多夹收藏同图只计数一次；删除 item / 删除 collection 时只在最后一份收藏消失后递减；favorite event value=1/-1。
- Service：visibility 默认值与非法值；非 owner 管理返回 forbidden；public/private 读取边界。
- Route smoke：未登录管理 401；非 owner 管理 403；公开收藏夹可读；私有收藏夹非 owner/游客不可读。

### 7. Wrong vs Correct

#### Wrong
```go
// 收藏数按 item 行数递增，重复收藏同图会重复计数。
if err := tx.Create(&po.CollectionItem{CollectionID: cid, ImageID: imageID}).Error; err == nil {
    tx.Model(&po.Image{}).Where("id = ?", imageID).
        UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1"))
}
```

#### Correct
```go
// 先判断该用户此前是否已收藏该图；只有首次用户收藏才递增。
hadFavorite, err := userHasFavoriteImage(ctx, tx, ownerID, imageID)
if err != nil { return err }
created, err := createCollectionItem(ctx, tx, collectionID, imageID)
if err != nil { return err }
if created && !hadFavorite {
    return incrementFavoriteCountAndEvent(ctx, tx, imageID, ownerID, 1)
}
```

## 8. 场景：热榜 API

### 1. Scope / Trigger
- 触发：新增日/周/月热榜能力，涉及定时 job、`ranking` 缓存表、`image_event` 聚合、贝叶斯评分与热度公式、HTTP 查询接口。
- 目标：前端可通过缓存的 ranking 表快速获得按综合热度排序的图片列表；定时 job 负责重算并持久化 ranking。

### 2. Signatures
- `GET /api/v1/rankings?period=day|week|month&page=&size=`，公开无 Auth。
- Response：`{total,page,size,list:[{rank,image_id,cos_key,filename,url,size,width,height,category,avg_score,rating_count,favorite_count,view_count,created_at}]}`。
- DB：`ranking(image_id,period,score,bayes_score,favorite_count,view_count,computed_at)`，按 `(period, score desc)` 排序缓存；`image_event(type=view|rating|favorite, value, user_id, image_id, created_at)` 作为源数据。
- Job：`internal/job/ranking_job.go`，默认 10min 调度，聚合 day/week/month 窗口，写 ranking，排除软删除。

### 3. Contracts
- `period` 必须为 `day` / `week` / `month`；其他值返回 HTTP 400 / Code 40001。
- 查询接口只读 `ranking` 缓存表并与 `image` join 返回活跃图片元数据；不在请求时实时重算。
- job 聚合规则：
  - `view` 计数不去重；`favorite` 按用户去重；`rating` 取窗口内评分事件 value。
  - 贝叶斯评分：`bayes = (C*m + sum_score)/(C+n)`，C 为可配置先验票数，m 为全局均分（默认 50）。
  - 热度：`score = w1*bayes + w2*log(1+fav) + w3*log(1+view)`，权重可配置。
  - 排除 `image.status=deleted`。
- handler 只接收 query param 并返回 `dto`；job/repository 使用 `do`/`po`；禁止 `po` 穿透 HTTP 响应。

### 4. Validation & Error Matrix
- 非法 `period` / 缺失 -> HTTP 400 / Code 40001。
- DB 读失败 -> HTTP 500 / Code 50001，由 handler 统一记录日志。

### 5. Good/Base/Bad Cases
- Good：job 运行后 `ranking` 表包含 day/week/month 各 top-N，且软删除图片不在其中。
- Good：`GET /rankings?period=day` 返回按 score 降序的图片列表，前端可渲染。
- Base：无事件数据时 ranking 表为空，接口返回空列表。
- Bad：查询接口实时聚合 image_event，导致高并发下响应变慢。

### 6. Tests Required
- Repository：day/week/month 窗口聚合、view 不去重、favorite 去重、贝叶斯与热度公式排序、软删除排除、缓存读写。
- Job：定时触发后 ranking 表被刷新。
- Route smoke：合法 period 返回 200；非法 period 返回 400。

### 7. Wrong vs Correct

#### Wrong
```go
// 查询接口实时聚合 image_event，导致 O(n) 扫描。
func (h *RankingHandler) List(c context.Context, ctx *app.RequestContext) {
    events := db.Where("created_at >= ?", window).Find(&po.ImageEvent{})
    // 在内存里聚合评分……
}
```

#### Correct
```go
// 查询接口只读缓存表；job 在后台定时重算。
func (h *RankingHandler) List(c context.Context, ctx *app.RequestContext) {
    rankings, total, err := h.rankingService.ListCached(c, period, page, size)
    if err != nil { /* 500 */ }
    Success(ctx, rankings)
}
```
