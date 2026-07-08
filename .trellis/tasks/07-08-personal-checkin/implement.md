# 签到功能执行计划

## 验证命令

- 后端编译：`go build ./...`
- 后端测试：`go test ./...`
- 前端构建：`cd frontend/vue-gallery && npm run build`

## 执行步骤

### 阶段 1：后端数据模型与迁移

- [ ] **1.1** 新建 `internal/model/do/checkin.go`：定义 `CheckIn`、`CheckInResult`、`MonthlyCheckIns` DO 结构。
- [ ] **1.2** 新建 `internal/model/po/checkin.go`：定义 `CheckIn` PO（含 `(user_id, check_in_date)` 复合唯一索引），`TableName() = "check_in"`。
- [ ] **1.3** 修改 `internal/model/do/user.go`：`User` 新增 `Points int64` 字段。
- [ ] **1.4** 修改 `internal/model/po/user.go`：`User` 新增 `Points int64 `gorm:"not null;default:0"``。
- [ ] **1.5** 修改 `internal/infra/db/sqlite.go` `AutoMigrate`：注册 `&po.CheckIn{}`。
- [ ] **验证**：`go build ./...` 通过。

### 阶段 2：后端 Repository 层

- [ ] **2.1** 修改 `internal/ports/repositories.go`：新增 `CheckInRepository` 接口（`CheckInToday`、`ListByMonth`）。
- [ ] **2.2** 新建 `internal/repository/checkin.go`：实现 `CheckInRepository`。
  - `CheckInToday`：GORM 事务，先查 `(user_id, date)`，存在返回 false；否则插入 checkin + `UPDATE user SET points = points + ? WHERE id = ?`，返回 true。唯一索引冲突时返回 false。
  - `ListByMonth`：按日期前缀 `BETWEEN '{year}-{month:02d}-01' AND '{year}-{month:02d}-31'` 查询，升序。
- [ ] **验证**：`go build ./...` 通过。

### 阶段 3：后端 Service 层

- [ ] **3.1** 新建 `internal/service/checkin.go`：`CheckInService` 注入 `CheckInRepository` + `UserRepository`。
  - 常量 `pointsPerDay = 10`，`cstLocation = time.FixedZone("CST", 8*3600)`。
  - `CheckInToday(ctx, userID)`：计算今日日期，调 repo，返回 `CheckInResult`。
  - `ListMonthly(ctx, userID, year, month)`：调 repo 查月度记录 + `UserRepository.FindByID` 取积分，返回 `MonthlyCheckIns`。
- [ ] **3.2** 新建 `internal/service/checkin_test.go`：mock repo 测试首签加分、重复签到幂等、月度查询。
- [ ] **验证**：`go test ./internal/service/...` 通过。

### 阶段 4：后端 Handler 与路由

- [ ] **4.1** 修改 `internal/model/dto/user.go`：`UserResponse` 新增 `Points int64 `json:"points"``。
- [ ] **4.2** 修改 `internal/handler/user.go`：
  - `UserHandler` 新增 `checkInService` 字段，更新 `NewUserHandler` 签名。
  - `Me` 方法：调用 `checkInService.CheckInToday`（best-effort，失败仅 warn 日志），再调 `CurrentUser`。
  - `toUserResponse` 映射 `Points`。
  - 新增 `ListCheckIns` 方法：解析 `year`/`month` 查询参数，调 `checkInService.ListMonthly`，返回响应。
- [ ] **4.3** 修改 `internal/handler/router/router.go`：
  - `Services` 新增 `CheckIn *service.CheckInService`。
  - `registerUserRoutes` 接收 `checkInService`，`NewUserHandler(userService, checkInService)`。
  - 注册 `users.GET("/me/check-ins", middleware.Auth(jwtManager), userHandler.ListCheckIns)`。
- [ ] **4.4** 修改 `cmd/web/main.go`：构造 `checkInRepo`、`checkInService`，注入 `router.Services`。
- [ ] **验证**：`go build ./...` + `go test ./...` 通过。

### 阶段 5：前端类型与 API

- [ ] **5.1** 修改 `frontend/vue-gallery/src/api/types.ts`：`UserResponse` 新增 `readonly points: number`；新增 `MonthlyCheckInsResponse` 接口。
- [ ] **5.2** 修改 `frontend/vue-gallery/src/api/client.ts`：新增 `getMonthlyCheckIns(year, month)` 方法。
- [ ] **验证**：`cd frontend/vue-gallery && npm run build` 类型检查通过。

### 阶段 6：前端签到日历组件

- [ ] **6.1** 新建 `frontend/vue-gallery/src/components/CheckInCalendar.vue`：
  - 月度日历网格（周一至周日列），已签到日期高亮，今日标记。
  - 月份切换按钮，累计积分展示。
  - onMounted 与月份切换时调用 `getMonthlyCheckIns`。
- [ ] **6.2** 修改 `frontend/vue-gallery/src/pages/AccountPage.vue`：
  - 导入 `CheckInCalendar`。
  - 登录态 section 内"登录状态"面板后插入 `<CheckInCalendar />`。
  - aside 账户摘要新增积分展示块。
- [ ] **验证**：`npm run build` 通过；浏览器访问 `/account` 登录后查看日历渲染。

### 阶段 7：集成验证

- [ ] **7.1** 启动后端，登录后 `GET /users/me` 验证 `points` 字段 +10。
- [ ] **7.2** 再次 `GET /users/me` 验证积分不变（幂等）。
- [ ] **7.3** `GET /users/me/check-ins?year=2026&month=7` 验证返回签到日期列表。
- [ ] **7.4** 浏览器访问个人中心，验证日历框渲染、积分展示、月份切换。

## 回滚点

- 阶段 1-4 为后端独立改动，前端未变，可单独回滚后端。
- 阶段 5-6 为前端独立改动，依赖后端新字段/接口，回滚需同时回滚后端。
- 数据库迁移为加列/加表，无破坏性，无需回滚迁移。

## Review Gates

- 阶段 4 完成后：后端编译+测试通过，review 后端分层与接口设计。
- 阶段 6 完成后：前端构建通过，review UI 设计与组件结构。
- 阶段 7 完成后：集成验证全部通过，准备 trellis-check。
