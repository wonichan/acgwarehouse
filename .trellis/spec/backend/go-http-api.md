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
- `UserResponse` 必须包含：`id, username, role, created_at, nickname, favorite_tags, bio, public_profile, email_notifications, sync_collections, points`，不得暴露 `password_hash`。
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
- 触发：新增/修改多收藏夹能力，涉及 HTTP API、owner/visibility 权限、SQLite `collection` / `collection_item` 表、收藏夹封面、`image.favorite_count` 去重聚合和 `image_event` 行为流水。
- 目标：用户可管理多个命名收藏夹；公开收藏夹可被任何人浏览；私有收藏夹仅 owner 可见；列表/详情响应能直接渲染封面与图片卡片；图片收藏数按去重用户数维护。

### 2. Signatures
- `GET /api/v1/collections`，需 `Auth`，返回当前用户收藏夹列表。
- `POST /api/v1/collections`，需 `Auth`，Request：`{"name": <string>, "visibility": "private"|"public"}`。
- `GET /api/v1/collections/:id`，公开收藏夹任意访问；私有仅 owner。
- `PUT /api/v1/collections/:id`，需 `Auth + owner`，Request：`{"name": <string>, "visibility": "private"|"public", "cover_image_id"?: <int64>}`。
- `DELETE /api/v1/collections/:id`，需 `Auth + owner`。
- `POST /api/v1/collections/:id/items`，需 owner，Request：`{"image_id": <int64>}`。
- `DELETE /api/v1/collections/:id/items/:imageId`，需 owner。
- DB：`collection(id,user_id,name,visibility,cover_image_id nullable,created_at,updated_at)`；`collection_item(collection_id,image_id,created_at)`，同夹同图唯一；`image.favorite_count` 为去重用户数；`image_event(type='favorite', value=1|-1)` 记录收藏/取消收藏事件。

### 3. Contracts
- `visibility` 缺省为 `private`；只允许 `private` / `public`。
- `CollectionResponse` 必须包含 `id,user_id,name,visibility,created_at,cover_image_id,cover_image_url,items`。
- `items[]` 必须包含 `collection_id,image_id,created_at,image`；`image` 使用 `ImageResponse` 字段集：`id,cos_key,filename,url,size,last_modified,width,height,category,avg_score,rating_count,favorite_count,view_count,created_at`。
- `cover_image_id=0` 表示未显式设置封面；`cover_image_url` 必须 fallback 到第一张可展示 `items[].image.url`；空收藏夹返回空字符串。
- `PUT /collections/:id` 省略 `cover_image_id` 时不得修改已有封面；显式 `cover_image_id:0` 清空封面；`cover_image_id>0` 设置封面。
- 设置封面时，目标图片必须属于该收藏夹且是 active 图片；否则返回参数错误。
- 查询收藏夹时必须过滤软删除/不可展示图片条目：不要返回 `image` 为空的 `items[]`，封面 fallback 也必须跳过这些条目。
- 同一收藏夹内同一图片只能存在一条 `collection_item`；重复加入应幂等，不重复增加 `favorite_count`。
- `favorite_count` 统计某图片被多少个不同用户收藏，而不是被多少个收藏夹收藏。
- 用户删除收藏夹时，必须按被删除 item 的图片逐一重算该 owner 是否仍收藏该图；只有 owner 对该图无剩余收藏时才递减 `favorite_count` 并发 `favorite value=-1` 事件。
- 删除收藏夹、删除 items、更新 `favorite_count`、写 favorite event 必须在同一写事务内完成。
- handler 只接收/返回 `dto`，service 使用 `do.Collection` / `do.CollectionItem`，repository 内部使用 `po`；禁止 `po` 穿透 HTTP 响应。

### 4. Validation & Error Matrix
- 未登录管理收藏夹或 item -> HTTP 401 / Code 40101。
- 非法 `:id` / `:imageId` / body 绑定失败 / 非法 visibility -> HTTP 400 / Code 40001。
- `cover_image_id` 不属于该收藏夹或对应图片不可展示 -> HTTP 400 / Code 40001。
- 收藏夹不存在，或私有收藏夹被非 owner 读取 -> HTTP 404 / Code 40401（避免泄露私有资源存在性）。
- 非 owner 更新/删除/添加/移除 item -> HTTP 403 / Code 40301。
- DB 事务、聚合重算或事件写入失败 -> HTTP 500 / Code 50001，由 handler 统一记录日志。

### 5. Good/Base/Bad Cases
- Good：用户 A 将图片 1 加入第一个收藏夹，`favorite_count +1` 且发 `favorite value=1`。
- Good：用户 A 又将图片 1 加入第二个收藏夹，`favorite_count` 不变，不重复发 +1 事件。
- Good：`GET /collections` 返回 `cover_image_url`，前端列表页无需二次请求即可显示收藏夹封面。
- Good：`PUT /collections/:id` 省略 `cover_image_id` 只改名称/可见性，不清空已设置封面。
- Good：`PUT /collections/:id` 传 `cover_image_id:0` 清空显式封面，响应继续 fallback 第一张可展示图片。
- Good：某收藏条目图片被软删除后，`items[]` 不返回该条目，封面 fallback 选择后续 active 图片。
- Base：公开收藏夹未登录可读；私有收藏夹非 owner 读取按不存在处理。
- Bad：按 `collection_item` 行数维护 `favorite_count`，导致同一用户多收藏夹收藏同图时重复计数。
- Bad：把 `cover_image_id` 省略和 `cover_image_id:0` 都解析为 0，导致普通更新误清空封面。
- Bad：`Preload("Items.Image")` 后仍返回 `Image.ID==0` 的 item，导致前端出现无图卡片或封面 URL 为空。

### 6. Tests Required
- Repository：同夹同图唯一；同用户多夹收藏同图只计数一次；删除 item / 删除 collection 时只在最后一份收藏消失后递减；favorite event value=1/-1。
- Repository：`cover_image_id` 持久化；省略 cover 字段更新时保留已有值；显式 0 清空；预加载时不可展示图片不进入领域 `Items`。
- Service：visibility 默认值与非法值；非 owner 管理返回 forbidden；public/private 读取边界；封面必须属于收藏夹；封面 URL fallback 跳过不可展示图片。
- Route smoke：未登录管理 401；非 owner 管理 403；公开收藏夹可读；私有收藏夹非 owner/游客不可读；`GET /collections/:id` 返回 `items[].image.url`；非法 cover 返回 400。
- Browser/E2E：登录后列表页显示封面且不显示 ID/Owner/imageID；详情页列出图片并可设封面；未登录可读公开详情且不显示设封面按钮。

### 7. Wrong vs Correct

#### Wrong
```go
// 收藏数按 item 行数递增，重复收藏同图会重复计数。
if err := tx.Create(&po.CollectionItem{CollectionID: cid, ImageID: imageID}).Error; err == nil {
    tx.Model(&po.Image{}).Where("id = ?", imageID).
        UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1"))
}
```

#### Wrong
```go
// 省略 cover_image_id 的普通更新会误清空封面。
collection.CoverImageID = derefInt64(input.CoverImageID)
if collection.CoverImageID > 0 {
    stored.CoverImageID = &collection.CoverImageID
} else {
    stored.CoverImageID = nil
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

#### Correct
```go
// 用显式 set 标志区分"字段省略"和"清空封面"。
coverImageIDSet := input.CoverImageID != nil
if coverImageIDSet {
    collection.CoverImageID = *input.CoverImageID
}
if collection.CoverImageIDSet {
    stored.CoverImageID = nil
    if collection.CoverImageID > 0 {
        stored.CoverImageID = &collection.CoverImageID
    }
}
```

## 10. 场景：用户每日签到 API

### 1. Scope / Trigger
- 触发：新增每日自动签到与积分累计能力，涉及 HTTP API、service 签到逻辑、SQLite `check_in` 表、`user.points` 字段、前端签到日历组件。
- 目标：用户每日首次访问个人中心时自动签到获得 10 积分；个人中心展示签到日历与累计积分；不支持补签。

### 2. Signatures
- `GET /api/v1/users/me`，需 `Auth`——在返回用户信息前**触发幂等自动签到**（best-effort 副作用）。
- `GET /api/v1/users/me/check-ins?year=&month=`，需 `Auth`——返回指定月份签到日期列表与累计积分。
- DB：`check_in(id,user_id,check_in_date,points_awarded,created_at)`，`(user_id,check_in_date)` 复合唯一索引保证同日幂等；`user.points int64 not null default 0` 存累计积分。
- Service：`CheckInService.CheckInToday(ctx, userID)` 计算亚洲/上海时区当日日期，调 repository 原子签到；`CheckInService.ListMonthly(ctx, userID, year, month)` 查月度记录 + 用户积分。

### 3. Contracts
- 签到日期以**亚洲/上海时区（UTC+8）**计算"今日"：`time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")`。
- `check_in_date` 字符串格式 `YYYY-MM-DD`，与 `daily_recommendation.date` 一致。
- `CheckInToday` 必须在**单事务**内完成：查 `(user_id, date)` 是否存在 → 不存在则 INSERT check_in 记录 + `UPDATE user SET points = points + 10 WHERE id = ?`；存在则跳过。返回 `(checkedIn bool, err error)`。
- `(user_id, check_in_date)` 复合唯一索引是幂等最终防线：并发请求至多一条 INSERT 成功，冲突时返回 `checkedIn=false`。
- `GET /users/me` 中的自动签到是 **best-effort**：签到失败仅 `logger.Warn` 记录，**不得**导致 `/users/me` 请求失败。
- `UserResponse.points` 必须反映签到后的最新累计积分。
- `GET /users/me/check-ins` 缺省 `year`/`month` 时回退到当前 CST 年/月。
- `MonthlyCheckInsResponse`：`{dates: []string, total_points: int64}`，`dates` 按日期升序。
- 每次签到固定发放 `10` 积分（service 常量 `pointsPerDay`）；不支持补签，不提供手动签到按钮。

### 4. Validation & Error Matrix
- 未登录调用 `/users/me` / `/users/me/check-ins` -> HTTP 401 / Code 40101。
- 签到事务失败（DB 异常）-> `/users/me` 仍返回 200（best-effort），错误仅写日志；`/users/me/check-ins` 返回 HTTP 500 / Code 50001。
- `year`/`month` 参数非法或缺省 -> 回退当前 CST 年/月，不返回 400。

### 5. Good/Base/Bad Cases
- Good：用户当日首次 `GET /users/me` -> `points` 增加 10，`check_in` 表新增一行。
- Good：同日再次 `GET /users/me` -> `points` 不变，`check_in` 表无新增。
- Good：跨日（CST 00:00 后）首次 `GET /users/me` -> `points` 再加 10。
- Good：`GET /users/me/check-ins?year=2026&month=7` 返回当月所有签到日期升序排列。
- Base：新注册用户 `points=0`，首次访问后 `points=10`。
- Bad：签到与积分更新不在同一事务，导致 check_in 有记录但 points 未增加。
- Bad：自动签到失败导致 `/users/me` 返回 500，用户无法查看个人资料。
- Bad：用 UTC 计算"今日"，导致中国用户 UTC 16:00（CST 00:00）后签到日期错位。

### 6. Tests Required
- Repository（真实 SQLite）：首签加分、同日幂等不加分、跨日累加、`ListByMonth` 升序且只返回当月、唯一索引防并发。
- Service（mock repo）：`CheckInToday` 首签返回 `PointsAwarded=10`、重复返回 `PointsAwarded=0`、`ListMonthly` 返回日期列表 + 用户积分。
- Route smoke：`/users/me/check-ins` 需 Auth；缺省参数回退当前月。
- 前端：`CheckInCalendar` 日历网格渲染、月份切换、积分展示、loading skeleton、`prefers-reduced-motion`。

### 7. Wrong vs Correct

#### Wrong
```go
// 签到与积分更新分开执行，不在同一事务，可能不一致。
func (r *CheckInRepository) CheckInToday(ctx context.Context, userID int64, date string, points int) (bool, error) {
    var existing po.CheckIn
    if err := r.db.Where("user_id = ? AND check_in_date = ?", userID, date).First(&existing).Error; err == nil {
        return false, nil
    }
    r.db.Create(&po.CheckIn{UserID: userID, CheckInDate: date, PointsAwarded: points})
    r.db.Model(&po.User{}).Where("id = ?", userID).UpdateColumn("points", gorm.Expr("points + ?", points))
    return true, nil
}
```

#### Wrong
```go
// 自动签到失败导致 /users/me 整体失败，用户无法查看资料。
func (h *UserHandler) Me(c context.Context, ctx *app.RequestContext) {
    if _, err := h.checkInService.CheckInToday(c, id); err != nil {
        Fail(c, ctx, consts.StatusInternalServerError, apperrors.CodeInternal, "签到失败", err)
        return
    }
    // ...
}
```

#### Correct
```go
// 签到与积分更新在同一事务内完成，唯一索引冲突时返回 false。
func (r *CheckInRepository) CheckInToday(ctx context.Context, userID int64, date string, pointsAwarded int) (bool, error) {
    checkedIn := false
    err := r.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        var existing po.CheckIn
        if err := tx.Where("user_id = ? AND check_in_date = ?", userID, date).First(&existing).Error; err == nil {
            return nil // 已签到
        } else if !errors.Is(err, gorm.ErrRecordNotFound) {
            return err
        }
        if err := tx.Create(&po.CheckIn{UserID: userID, CheckInDate: date, PointsAwarded: pointsAwarded}).Error; err != nil {
            if isUniqueConstraintError(err) { return nil } // 并发幂等
            return err
        }
        return tx.Model(&po.User{}).Where("id = ?", userID).
            UpdateColumn("points", gorm.Expr("points + ?", pointsAwarded)).Error
    })
    return checkedIn, err
}

// 自动签到 best-effort：失败仅 warn，不影响 /users/me 正常返回。
func (h *UserHandler) Me(c context.Context, ctx *app.RequestContext) {
    id, ok := requiredCurrentUserID(c, ctx)
    if !ok { return }
    if _, err := h.checkInService.CheckInToday(c, id); err != nil {
        logger.Warn(c, "auto check-in failed", zap.Error(err))
    }
    user, err := h.userService.CurrentUser(c, id)
    // ...
}
```

## 8. 场景：热榜 API

### 1. Scope / Trigger
- 触发：新增日/周/月热榜能力，涉及定时 job、`ranking` 缓存表、`image_event` 聚合、贝叶斯评分与热度公式、HTTP 查询接口。
- 目标：前端可通过缓存的 ranking 表快速获得按综合热度排序的图片列表；定时 job 负责重算并持久化 ranking。

### 2. Signatures
- `GET /api/v1/rankings?period=day|week|month&page=&size=`，公开无 Auth。
- `size` 省略时使用周期默认展示数量：`day=20`、`week=50`、`month=100`；显式传入 `size` 时尊重请求值。
- Response：`{total,page,size,list:[{rank,image_id,cos_key,filename,url,size,width,height,category,avg_score,rating_count,favorite_count,view_count,created_at}]}`。
- DB：`ranking(image_id,period,score,bayes_score,favorite_count,view_count,computed_at)`，按 `(period, score desc)` 排序缓存；`image_event(type=view|rating|favorite, value, user_id, image_id, created_at)` 作为源数据。
- Job：`internal/job/ranking_job.go`，默认 10min 调度，聚合 day/week/month 窗口，按周期默认展示数量裁剪后写 ranking，排除软删除。

### 3. Contracts
- `period` 必须为 `day` / `week` / `month`；其他值返回 HTTP 400 / Code 40001。
- 查询接口只读 `ranking` 缓存表并与 `image` join 返回活跃图片元数据；不在请求时实时重算。
- job 聚合规则：
  - `view` 计数不去重；`favorite` 按用户去重；`rating` 取窗口内评分事件 value。
  - 贝叶斯评分：`bayes = (C*m + sum_score)/(C+n)`，C 为可配置先验票数，m 为全局均分（默认 50）。
  - 热度：`score = w1*bayes + w2*log(1+fav) + w3*log(1+view)`，权重可配置。
  - 排除 `image.status=deleted`。
- handler 只接收 query param 并返回 `dto`；job/repository 使用 `do`/`po`；禁止 `po` 穿透 HTTP 响应。
- 周期默认展示数量必须集中在 `do.RankingPeriod` 领域层（例如 `DefaultSize()`）；handler 默认分页、job 缓存裁剪、前端请求数量都引用或镜像同一契约，禁止在各层散落互不一致的 magic number。

### 4. Validation & Error Matrix
- 非法 `period` / 缺失 -> HTTP 400 / Code 40001。
- DB 读失败 -> HTTP 500 / Code 50001，由 handler 统一记录日志。

### 5. Good/Base/Bad Cases
- Good：job 运行后 `ranking` 表包含 day/week/month 各 top-N，且软删除图片不在其中。
- Good：job 运行后 day/week/month 分别最多缓存 20/50/100 张；`GET /rankings?period=week` 省略 `size` 时响应 `data.size=50`。
- Good：`GET /rankings?period=day` 返回按 score 降序的图片列表，前端可渲染。
- Base：无事件数据时 ranking 表为空，接口返回空列表。
- Bad：查询接口实时聚合 image_event，导致高并发下响应变慢。

### 6. Tests Required
- Repository：day/week/month 窗口聚合、view 不去重、favorite 去重、贝叶斯与热度公式排序、软删除排除、缓存读写。
- Job：定时触发后 ranking 表被刷新，并断言 day/week/month 裁剪数量分别为 20/50/100。
- Route smoke：合法 period 返回 200；非法 period 返回 400；省略 `size` 时 day/week/month 响应 `size` 分别为 20/50/100。

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

## 9. 场景：每日随机推荐 API

### 1. Scope / Trigger
- 触发：新增全站统一“今日每日推荐”，涉及公开 HTTP API、SQLite 持久化随机池、service 日期边界、前端首页展示。
- 目标：每天按北京时间自然日提供最多 10 张 active 图片；推荐来源是公平随机池，不复用热榜/社区精选算法。

### 2. Signatures
- API：`GET /api/v1/daily-recommendations`，公开无 Auth。
- Response：`{date,timezone,total,list:[ImageResponse...]}`，外层仍为 `Response{Code,Data,Msg}`。
- DB：
  - `daily_recommendation(date,image_id,position,cycle,created_at)`，`date + image_id` 唯一表示某日推荐项。
  - `daily_recommendation_pool(image_id,cycle,position,created_at)` 保存当前随机周期剩余池。
  - `daily_recommendation_state(key,cycle,updated_at)` 保存全局周期，`key='global'`。

### 3. Contracts
- `date` 使用 Asia/Shanghai 自然日，格式 `YYYY-MM-DD`；数据库时间戳仍存 UTC。
- `timezone` 固定返回 `Asia/Shanghai`。
- `total` 表示本次返回 `list` 长度，不表示全站图片总数。
- `list` 只包含 active 图片：`image.status='active' AND image.deleted_at IS NULL`。
- 同一 `date` 多次请求必须返回稳定结果；若已有当日结果中的图片被软删除，允许同日补齐，但不得重复返回当天已有 image_id。
- 新增或恢复的 active 图片进入当前 cycle 的剩余池；当剩余池不足以补齐当日数量时，开启下一 cycle 并重新洗牌 active 图片。
- active 图片少于 10 张时返回全部 active 图片，不重复填充。
- `/api/v1/rankings` 与 ranking job 不得因每日推荐改动而改变行为。

### 4. Validation & Error Matrix
- SQLite 读写失败 / AutoMigrate 缺表 / pool 维护失败 -> HTTP 500 / Code 50001，handler 统一记录日志。
- active 图片为空 -> HTTP 200 / Code 0，`total=0,list=[]`。
- 今日已有推荐但部分图片软删除 -> HTTP 200 / Code 0，过滤软删除并尝试补齐。
- 同日并发首次请求遇到唯一键冲突 -> 不消费 pool row；通过插入 `RowsAffected` 判断后再删除 pool row。

### 5. Good/Base/Bad Cases
- Good：`GET /daily-recommendations` 首次请求生成 10 张 active 图片，第二次同日请求返回相同顺序。
- Good：第 1 天返回 1-10，第 2 天返回 11-20；当 cycle 剩余不足 10 时，用下一 cycle 补齐但不复用同日已展示图片。
- Good：今日推荐项被软删除后再次请求，软删除项消失并由 pool 中下一张 active 图片补齐。
- Good：恢复一张 deleted 图片后，该图片进入当前 cycle 剩余池，有机会在下一次生成中出现。
- Base：图库只有 3 张 active 图片时返回 3 张且无重复。
- Bad：请求时直接 `ORDER BY RANDOM()`，导致同一天刷新频繁变化且无法保证长期公平。
- Bad：`ON CONFLICT DO NOTHING` 后仍删除 pool row，导致并发下消耗了候选但没有生成推荐行。

### 6. Tests Required
- Repository：同日稳定、跨日 cycle 推进、小池无重复、软删除过滤并补齐、恢复图片加入当前 pool、同日补齐不复用已有 row、AutoMigrate 创建表。
- Service：UTC 时间跨北京时间 00:00 时 date key 正确；COS URL 拼接与 `{date,timezone,total,list}` DTO 正确。
- Route smoke：`GET /daily-recommendations` 成功返回 200/envelope/shape；repository/service 错误返回 500。
- Regression：保留 `/rankings?period=week` 等既有 route 测试，确认热榜不受影响。

### 7. Wrong vs Correct

#### Wrong
```go
// 每次请求实时随机，刷新页面会变化，也无法保证每张图片长期有机会出现。
db.Order("RANDOM()").Limit(10).Find(&images)
```

#### Correct
```go
inserted, err := createTodayRow(ctx, tx, date, imageID, nextPosition, cycle, nowUTC)
if err != nil {
    return err
}
if inserted {
    return deletePoolRow(ctx, tx, imageID)
}
```

#### Wrong
```go
// 同日补齐新 cycle 时没有排除当日已展示图片，可能重复返回同一 image_id。
rows := listPoolRows(ctx, tx, cycle)
```

#### Correct
```go
rows := listPoolRows(ctx, tx, cycle, date) // excludes image_id already used for this date
```
