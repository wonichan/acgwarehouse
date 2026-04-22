# 缩略图访问地址兼容 MinIO / COS 设计

- 日期：2026-04-22
- 主题：修正缩略图访问地址解析逻辑，使前端在 MinIO 与腾讯云 COS 场景下都能稳定展示缩略图

## 目标

现有前端通过运行时配置中的 `thumbnail_base_url` 与数据库中的缩略图字段拼接最终访问地址。该方案本身支持“绝对 URL 直出、相对路径拼接”的双形态，但后端当前只按 MinIO 配置生成 `thumbnail_base_url`，导致使用 COS 作为缩略图存储后端时，运行时基址解析规则失真。

本次目标是：

1. 让后端按 `thumbnail_storage_provider` 统一生成正确的 `thumbnail_base_url`。
2. 保持前端的 URL 解析规则简单稳定，不在前端识别存储厂商。
3. 兼容数据库中既可能存在绝对 URL、也可能存在相对路径的历史数据，但仅限这些字段仍与**当前部署的存储 provider**语义一致。
4. 保证 AI 打标链路拿到的图片地址也遵循同一套解析规则，而不是仅修复前端展示。
5. 通过后端与 Flutter 测试封印行为，避免再次退化为只兼容 MinIO。

## 已确认规则

1. `thumbnail_storage_provider` 是当前缩略图存储后端的唯一选择开关，合法值至少包括 `minio` 与 `cos`。
   - provider 比较规则应先 `TrimSpace` 再转为小写；未命中已知值时按未知 provider 处理。
2. `internal/service/minio_service.go` 当前上传成功后返回相对路径（例如 `acg/thumbnails/20260421/example-large.jpg`）。
3. `internal/service/cos_service.go` 当前上传成功后返回绝对 URL（例如 `https://bucket.cos.ap-shanghai.myqcloud.com/thumbnails/example-large.jpg`）。
4. 前端 `ApiConfig.resolveThumbnailUrl()` 已具备以下规则：
   - 字段为空：返回空。
   - 字段为绝对 URL：原样返回。
   - 字段为相对路径且 `thumbnailBaseUrl` 非空：进行拼接。
   - 字段为相对路径且 `thumbnailBaseUrl` 为空：返回原值。
5. 当前根因不在前端拼接器本身，而在后端 `ResolveThumbnailBaseURL(cfg)` 仅基于 MinIO 配置产出基址。
6. AI 打标链路也依赖缩略图地址：`ResolveAITagImagePath(image, thumbnailBaseURL)` 会优先使用 `image.ThumbnailLargeUrl`，并通过 `ResolveThumbnailURL()` 解析为最终可访问地址；只有缩略图缺失时才回退到原图 `image.Path`。
7. `internal/handler/ai_tag_handler.go`、`internal/service/ai_backfill_service.go`、`internal/service/ai_tag_auto_scheduler.go` 都会先调用 `ResolveThumbnailBaseURL(...)`，再把解析后的路径写入 AI 任务 payload。

## 当前实现与约束

### 后端

- `internal/service/thumbnail_path.go`
  - `BuildThumbnailBaseURL(endpoint, useSSL)` 目前仅适用于 MinIO 风格 endpoint。
  - `ResolveThumbnailBaseURL(cfg)` 当前无视 `thumbnail_storage_provider`，直接读取 `cfg.Minio.Endpoint`。
- `internal/app/app.go`
  - 应用启动时调用 `service.ResolveThumbnailBaseURL(a.config)`，将结果写入 runtime manifest。
- `internal/app/runtime_manifest.go`
  - `BuildRuntimeManifestPayload()` 已支持 `thumbnail_base_url` 字段。

### 前端

- `flutter_app/lib/bootstrap/runtime_manifest_loader.dart`
  - 已支持读取 `go.thumbnail_base_url`。
- `flutter_app/lib/providers/config_provider.dart`
  - 已持有 `thumbnailBaseUrl`，并通过 `resolveThumbnailUrl()` 提供统一解析。
- `flutter_app/lib/config/api_config.dart`
  - `resolveThumbnailUrl()` 已支持绝对 URL 原样返回与相对路径拼接。

### 约束

1. 不应把 COS / MinIO 的识别逻辑扩散到前端多个页面或组件。
2. 不应强制立即迁移历史数据库字段值；若存在跨 provider 留下的旧相对路径，本次不承诺自动修复。
3. 不应修改已工作的绝对 URL 行为；绝对 URL 必须优先原样输出。
4. 运行时 manifest 允许 `thumbnail_base_url` 为空，但不得输出非法 URL。
5. AI payload 中的图片地址必须与前端展示使用同一套解析契约，避免两套规则分裂。

## 方案比较

### 方案一：最小补丁，按 provider 生成不同基址

在后端 `ResolveThumbnailBaseURL(cfg)` 中加入 provider 分流：

- `minio` → 继续使用 `BuildThumbnailBaseURL(cfg.Minio.Endpoint, cfg.Minio.UseSSL)`
- `cos` → 对 `cfg.COS.BucketURL` 做绝对 `http/https` URL 校验后再规范化输出

优点：

1. 改动最小。
2. 不需要触碰前端主逻辑。
3. 能立刻修复当前 COS 场景。

缺点：

1. 规则仍主要体现在一个 helper 中，语义说明较少。
2. 需要补足测试，否则后续容易回退。

### 方案二：前端增强识别逻辑

在前端根据 URL 形态、域名或 provider 增加更多兼容判断。

优点：

1. 后端改动少。

缺点：

1. 错误地把存储语义下沉到前端。
2. 需要在多个前端链路保持一致，维护成本高。
3. 一旦后端存储规则继续变化，前端将持续背负兼容债务。

### 方案三：统一运行时法典（推荐）

以后端为唯一基址裁定者：

1. `ResolveThumbnailBaseURL(cfg)` 按 `thumbnail_storage_provider` 生成合法基址。
2. 前端只保留“绝对 URL 直出 / 相对路径拼接”两条规则。
3. AI 打标入口也统一依赖同一个后端解析规则生成图片地址。
4. 历史数据允许同时存在绝对 URL 与相对路径，不要求本次迁移；但跨 provider 的旧相对路径不在本次自动兼容范围内。
5. 用后端与 Flutter 测试共同锁定该契约。

优点：

1. 职责边界清晰。
2. 兼容历史数据。
3. 改动集中，风险可控。
4. 最符合当前项目的 runtime manifest 架构。

缺点：

1. 需要补强测试覆盖。

## 结论

采用**方案三：统一运行时法典**。

## 推荐设计

## 架构与组件边界

### 1. 后端配置层

继续由 `config.Config` 持有：

- `ThumbnailStorageProvider`
- `COS.BucketURL`
- `Minio.Endpoint`
- `Minio.UseSSL`

配置层仅负责提供原始配置，不负责拼接访问 URL。

### 2. 后端缩略图路径服务层

`internal/service/thumbnail_path.go` 作为唯一地址法典入口，负责：

1. 解析当前 provider 对应的缩略图基址。
2. 保留现有 `ResolveThumbnailURL(baseURL, raw)` 的“绝对优先、相对拼接”规则。
3. 为前端展示与 AI 打标共同提供同一套地址解析契约。
4. 对非法或缺失配置返回空字符串，而不是输出伪造 URL。

建议演进为以下职责：

- `BuildThumbnailBaseURL(endpoint string, useSSL bool) string`：保留 MinIO endpoint 规范化。
- `ResolveThumbnailBaseURL(cfg *config.Config) string`：按 provider 选择 MinIO 或 COS，并只在 provider 配置可解析为合法绝对 URL 时输出基址。

其中：

- MinIO 允许使用现有 endpoint 形态（裸 `host:port` 或已带 `http/https` 的地址），统一通过 `BuildThumbnailBaseURL()` 规范化为绝对 URL。
- COS 仅接受可解析的绝对 `http/https` URL。

### 3. Runtime manifest 层

`internal/app/app.go` 继续在启动时写入：

- `base_url`
- `thumbnail_base_url`

但 `thumbnail_base_url` 的值必须由新的 `ResolveThumbnailBaseURL(cfg)` 统一裁定。

### 4. 前端运行时配置层

前端保持现状：

- `RuntimeManifestLoader` 负责读取 manifest 中的 `thumbnail_base_url`
- `ConfigProvider` 负责保存 `thumbnailBaseUrl`
- `ApiConfig.resolveThumbnailUrl()` 负责最终解析

前端不得新增任何与 `thumbnail_storage_provider` 相关的判断。

### 5. AI 图片来源层

`internal/service/ai_image_source.go` 继续作为 AI 输入地址总入口：

1. 若 `ThumbnailLargeUrl` 存在，则调用 `ResolveThumbnailURL(thumbnailBaseURL, image.ThumbnailLargeUrl)`。
2. 若 `ThumbnailLargeUrl` 不存在，则回退到 `image.Path`。
3. `ai_tag_handler`、`ai_backfill_service`、`ai_tag_auto_scheduler` 三条链路都必须复用该入口，不得各自拼接 URL。

这里需要明确：本次统一的是 **`ThumbnailLargeUrl` 的解析契约**。当 AI 链路回退到 `image.Path` 时，保持现有值原样透传，不套用 `thumbnail_base_url`。
`image.Path` 的可访问性不是本次设计要统一的新契约；若某些部署中的 `image.Path` 只是本地存储路径或相对路径，该问题仍属于独立议题，不由本次缩略图基址修复一并解决。

## 数据流设计

### 1. 上传阶段

- MinIO 上传后，数据库继续写入相对路径。
- COS 上传后，数据库当前仍写入绝对 URL。

本次不要求统一上传返回值形态，只要求运行时解析规则对两者都有效。

### 2. 应用启动阶段

应用启动时：

1. 读取配置文件。
2. 调用 `ResolveThumbnailBaseURL(cfg)`。
3. 若 provider 为 `minio`，生成 MinIO 可访问基址。
4. 若 provider 为 `cos`，生成规范化后的 `cfg.COS.BucketURL`。
5. 将结果写入 `runtime-manifest.json` 的 `go.thumbnail_base_url`。

### 3. 前端展示阶段

界面读取缩略图字段后，统一遵循：

1. 为空 → 不展示。
2. 为绝对 URL → 直接访问。
3. 为相对路径且存在 `thumbnailBaseUrl` → 拼接后访问。
4. 为相对路径且 `thumbnailBaseUrl` 为空 → 保留原值，由展示失败暴露配置问题。

### 4. AI 打标阶段

AI 打标相关链路统一遵循：

1. 先调用 `ResolveThumbnailBaseURL(cfg)` 得到当前部署对应的基址。
2. 再调用 `ResolveAITagImagePath(image, thumbnailBaseURL)` 生成最终传给 AI 服务商的 `path`。
3. 若 `ThumbnailLargeUrl` 为绝对 URL，则直接透传给 AI。
4. 若 `ThumbnailLargeUrl` 为相对路径，则按当前 provider 对应的基址拼接。
5. 若无缩略图，则回退到 `image.Path`。
6. 若 `ThumbnailLargeUrl` 为相对路径但 `thumbnailBaseURL` 为空，则与前端契约保持一致：返回原值、不猜测 provider、不伪造 URL，并由日志暴露配置问题。

这意味着 `thumbnail_base_url` 的正确性不再只是 UI 可见性问题，而是 AI 输入契约问题。

## 错误处理

1. 未知 `thumbnail_storage_provider`：
   - `ResolveThumbnailBaseURL(cfg)` 返回空字符串。
   - 可记录告警日志，但不生成伪造 URL。
   - `ResolveThumbnailBaseURL(nil)` 也必须返回空字符串。
2. `cfg.COS.BucketURL` 为空或非法：
   - 返回空字符串。
   - 仅当其可解析为绝对 `http/https` URL 时才允许写入 runtime manifest 或用于 AI 拼接。
3. `cfg.Minio.Endpoint` 为空：
   - 返回空字符串。
   - 若 `BuildThumbnailBaseURL()` 无法得到可用绝对 URL，也返回空字符串。
4. 数据库字段为绝对 URL：
   - 永远优先原样返回，不做二次拼接。
5. 数据库字段为相对路径但缺少基址：
   - 前端保留原值；问题由配置与日志暴露，而不是前端猜测修复。
   - AI 链路同样保留原值；问题由配置与日志暴露，而不是在任务投递时伪造可访问地址。
   - `thumbnailBaseURL` 非法时，也按同样规则降级为“保留原值 + 暴露日志”。
6. 若历史数据库中保留的是来自旧 provider 的相对路径：
   - 本次设计不尝试从路径本身推断 provider。
   - 是否迁移由后续专门方案处理。
7. runtime manifest 中的 `thumbnail_base_url` 允许为空，但不得用占位值、默认假地址或伪造 URL 填充。

## 测试策略

### 后端测试

建议新增或补强以下测试：

1. `internal/service/thumbnail_path*_test.go`
   - `ResolveThumbnailBaseURL` 在 `minio` 配置下返回 MinIO 基址。
   - `ResolveThumbnailBaseURL` 在 `cos` 配置下返回 COS bucket URL。
   - `ResolveThumbnailBaseURL` 对 provider 的大小写与空白做归一化处理。
   - `ResolveThumbnailBaseURL` 对 COS 尾部 `/` 做规范化。
   - `ResolveThumbnailBaseURL` 对非法 COS URL 返回空字符串。
   - `ResolveThumbnailBaseURL(nil)` 返回空字符串。
   - provider 未知时返回空字符串。
   - 缺失关键配置时返回空字符串。
   - MinIO endpoint 无法形成有效绝对 URL 时返回空字符串。
   - `ResolveThumbnailURL` 必测：绝对 URL 原样返回、相对路径正常拼接、空值返回空、空/非法 base URL 时原样返回相对路径。
2. `internal/app/runtime_manifest_test.go`
   - `BuildRuntimeManifestPayload` 接收并写出 COS 风格 `thumbnail_base_url`。
   - `thumbnail_base_url` 为空时保持为空，不生成伪造地址。
3. `internal/service/ai_image_source_test.go`
    - 验证绝对 URL 保持原样。
    - 验证相对路径在 COS base URL 下正确拼接。
    - 验证缩略图缺失时回退到 `image.Path`。
    - 验证相对路径在空/非法 `thumbnailBaseURL` 下原样返回。
4. AI 入口链路测试：
    - `internal/handler/ai_tag_handler_test.go` 覆盖相对路径缩略图在当前 provider 下写入正确 payload。
   - `internal/handler/ai_tag_handler_test.go` 也必须覆盖空/非法 base URL 下 payload 保留原值的降级行为。
   - `internal/service/ai_backfill_service*_test.go` 必须验证 payload 中的 `path` 与 `ResolveAITagImagePath` 契约一致，并覆盖空/非法 base URL 的降级行为。
   - `internal/service/ai_tag_auto_scheduler*_test.go` 必须验证 payload 中的 `path` 与 `ResolveAITagImagePath` 契约一致，并覆盖空/非法 base URL 的降级行为。

### 前端测试

建议补强以下测试：

1. `flutter_app/test/config/api_config_test.dart`
   - 绝对 URL 保持不变。
   - 相对路径与 MinIO base URL 正确拼接。
   - 相对路径与 COS base URL 正确拼接。
   - 相对路径在空 base URL 下返回原值。
2. `flutter_app/test/bootstrap/runtime_manifest_loader_test.dart`
   - 能读取 manifest 中的 COS `thumbnail_base_url`。
3. 若已有界面级测试依赖缩略图展示：
   - 验证 `ConfigProvider.resolveThumbnailUrl()` 在绝对 / 相对场景下的行为未回退。

## 非目标

本次不包含以下内容：

1. 将 COS 上传返回值改为相对路径。
2. 批量迁移数据库中既有的缩略图字段值。
3. 重构前端图片组件的渲染结构。
4. 引入新的缩略图访问接口。
5. 让 AI 打标入口自行识别 provider 或绕过统一解析函数。

## 实施摘要

最小闭环应集中在以下文件：

- `internal/service/thumbnail_path.go`
- 对应后端测试文件
- 如有必要的 `internal/app/runtime_manifest_test.go`
- `flutter_app/test/config/api_config_test.dart`
- `flutter_app/test/bootstrap/runtime_manifest_loader_test.dart`

核心裁定是：**后端按 provider 生成正确的 `thumbnail_base_url`，前端继续只识别 URL 形态，不识别厂商。**
