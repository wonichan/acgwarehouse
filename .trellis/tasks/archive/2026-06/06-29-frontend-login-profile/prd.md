# 补齐登录注册与用户个人中心

## Goal

补齐 ACGWarehouse 的登录、注册和用户个人中心端到端能力。前端 `frontend/vue-gallery` 需要参考 `frontend/example` 的账户中心样例完成可访问、可反馈、可响应式的界面；后端需要在现有注册、登录、`/users/me` 基础上补充个人资料、偏好设置和密码修改能力，确保用户个人中心中的可编辑内容真实持久化，而不是前端假保存。

## Background / Confirmed Facts

- 用户明确要求：前端用户登录功能和用户个人中心界面没有完整实现，需要对照 `frontend/example` 中的样例解决；若后端没有实现功能，则补充后端功能，确保登录注册和用户个人中心完整。
- 当前前端是 TypeScript + Vue 3 + Vite 项目，相关实现位于 `frontend/vue-gallery`。
- 当前后端是 Go + Hertz + GORM + SQLite，用户表由 `internal/infra/db/sqlite.go` 的 `AutoMigrate()` 自动迁移。
- 前端规范要求所有类型导入使用 `import type`。
- 后端规范要求 HTTP API 使用 `/api/v1/*`、统一 `{code,data,msg}` 响应、handler 只处理 DTO、service 使用 DO、repository 使用 PO，并保持 handler -> service -> ports -> repository 依赖方向。
- 当前后端用户 API 已有 `POST /api/v1/users/register`、`POST /api/v1/users/login`、`GET /api/v1/users/me`。
- 当前 `internal/model/po.User` 只有 `id`、`username`、`password_hash`、`role`、`created_at`；`internal/model/do.User` 和 `internal/model/dto.UserResponse` 也只表达这些公开字段。
- 当前 `internal/repository.UserRepository` 只有 `FindByUsername`、`FindByID`、`Create`；`internal/service.UserService` 只有 `Register`、`Login`、`CurrentUser`、`EnsureAdmin`。
- 当前 `frontend/vue-gallery/src/api/client.ts` 已有 `login(username, password)`、`register(username, password)`、`getCurrentUser()` API 封装。
- 当前 `frontend/vue-gallery/src/composables/useAuth.ts` 已有 `user`、`loading`、`error`、`isLoggedIn`、`isAdmin`、`login`、`register`、`logout`、`initAuth` 状态与方法。
- 当前 `frontend/vue-gallery/src/router/index.ts` 已有 `/account` 路由，页面组件为 `AccountPage.vue`。
- `frontend/example/DESIGN.md` 定义了 Colorful 设计系统：暖黄背景、白色面板、橙色主强调、深紫黑文本、8pt 间距、可访问表单与克制动效。
- `frontend/example/screens/account.html` 是账户中心样例，覆盖登录/注册标签页、资料保存、偏好开关、安全状态、最近活动、inline status、helper text、键盘可访问标签页和 toast 反馈。
- 当前 `frontend/vue-gallery/src/pages/AccountPage.vue` 已有登录/注册表单、资料/偏好/安全面板和退出按钮，但仍存在样例对照缺口：表单不是 `<form>` 提交流；标签页按钮缺少 `id` 但面板使用 `aria-labelledby`；登录/注册错误只通过 toast 暴露，缺少样例中的 inline status；资料、偏好、安全动作仍是占位文案或静态字段；最近活动面板缺失。
- 当前 `frontend/vue-gallery/src/components/AppHeader.vue` 右侧动作始终显示“登录”，未根据 `useAuth()` 登录状态显示用户入口或退出相关状态。

## Requirements

- R1. 登录功能必须可从前端页面完成，使用后端 `POST /api/v1/users/login` 并在成功后持久化 token。
- R2. 注册功能必须可从前端页面完成，使用后端 `POST /api/v1/users/register`，成功后自动进入登录态或清晰引导登录。
- R3. 页面必须能在已有 token 时初始化当前用户状态，并在 token 失效时清理状态。
- R4. 后端用户公开响应必须包含个人中心需要展示和编辑的资料字段：用户名、角色、创建时间、显示昵称、常用标签、个人简介、偏好设置和安全摘要所需字段。
- R5. 后端必须提供当前用户资料读取/更新接口，保存显示昵称、常用标签、个人简介和偏好设置。
- R6. 后端必须提供当前用户密码修改接口，要求旧密码正确、新密码满足当前密码规则，并保持 JWT 认证保护。
- R7. 用户个人中心必须展示当前登录用户的核心信息，并提供明确的已登录/未登录状态。
- R8. 未登录用户访问账户中心时，必须能看到登录入口或登录表单，不应进入空白或半成品界面。
- R9. 登录、注册、资料保存、偏好保存、密码修改、退出登录等状态必须有可见反馈。
- R10. 实现必须优先复用现有 API client、auth composable、router、样式、GORM repository、service 和 handler 模式，不引入无关依赖。
- R11. UI 与交互应参考 `frontend/example` 中对应登录和个人中心样例，但要适配当前 `frontend/vue-gallery` 的现有设计系统与页面结构。
- R12. 登录/注册/资料/安全表单必须具备可访问的 label、helper text、校验错误、提交中状态和成功/失败状态反馈。
- R13. 顶部导航账户入口必须根据登录状态展示合适文案，例如未登录为“登录”，已登录为用户名或“我的”。
- R14. Vue 前端实现前必须补齐或引用项目设计系统文档，保证账户页改动追溯到现有 Colorful token 和组件规范。

## Acceptance Criteria

- [ ] 从账户中心或导航入口可以完成登录，并调用 `/api/v1/users/login`。
- [ ] 从账户中心可以完成注册，并调用 `/api/v1/users/register`。
- [ ] 登录成功后 token 被保存，`/api/v1/users/me` 可加载并展示当前用户信息和个人中心资料。
- [ ] 刷新页面后仍能根据已保存 token 恢复登录状态。
- [ ] token 无效或用户主动退出后，前端清理认证状态并回到未登录视图。
- [ ] 当前用户可以保存显示昵称、常用标签、个人简介和偏好设置；刷新页面后仍展示后端持久化值。
- [ ] 当前用户可以修改密码；旧密码错误返回明确错误，新密码太短返回参数错误，成功后可用新密码登录。
- [ ] 账户中心未登录、加载中、登录失败、注册失败、资料保存失败、密码修改失败、已登录、保存成功、退出后状态均有明确 UI。
- [ ] 实现对照 `frontend/example` 后覆盖样例中适用于当前项目的登录/个人中心能力：标签页、表单 helper、inline status、资料、偏好、安全、最近活动空态。
- [ ] 登录/注册 tab 的 `role=tablist/tab/tabpanel`、`aria-selected`、`aria-controls`、`aria-labelledby`、键盘左右/上下切换均正确。
- [ ] 表单校验失败时，相关输入设置 `aria-invalid` 并显示 inline 错误；后端失败时显示 inline status 和 toast。
- [ ] Header 登录入口能反映当前认证状态。
- [ ] `frontend/vue-gallery/DESIGN.md` 或等价设计系统文档存在，并记录本任务使用的 Colorful token、表单、账户面板和状态反馈规则。
- [ ] Go 单元/路由测试覆盖注册、登录、`/users/me`、资料更新、偏好更新、密码修改成功与失败路径。
- [ ] TypeScript 类型检查、Go 测试、前端构建通过。
- [ ] 使用真实浏览器检查登录、注册、个人资料保存、偏好保存、密码修改和退出登录主要交互路径，至少覆盖桌面和移动宽度。

## Out of Scope

- 不新增第三方登录、找回密码、邮箱验证码、双重验证、活跃会话管理等安全系统。
- 不做无关页面 redesign。
- 不引入独立迁移框架；沿用现有 GORM AutoMigrate 机制扩展用户持久化字段。

## Open Questions

- None.
