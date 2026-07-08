# 签到功能技术设计

## 1. 边界与范围

后端新增签到领域（CheckIn DO/PO/Repository/Service/Handler），User 模型新增 `Points` 累计积分字段。前端 AccountPage 新增签到日历面板与积分展示，新增 API 客户端方法。

## 2. 数据模型

### 2.1 新增 CheckIn 持久化对象

`internal/model/po/checkin.go`:

```go
type CheckIn struct {
    ID            int64     `gorm:"primaryKey"`
    UserID        int64     `gorm:"not null;uniqueIndex:idx_user_date,priority:1"`
    CheckInDate   string    `gorm:"size:10;not null;uniqueIndex:idx_user_date,priority:2"` // "2006-01-02"
    PointsAwarded int       `gorm:"not null"`
    CreatedAt     time.Time `gorm:"not null"`
}
```

- `(user_id, check_in_date)` 复合唯一索引，数据库层保证同日不重复签到。
- `check_in_date` 用字符串存储，便于日历查询与唯一约束。

### 2.2 新增 CheckIn 领域对象

`internal/model/do/checkin.go`:

```go
type CheckIn struct {
    ID            int64
    UserID        int64
    CheckInDate   string
    PointsAwarded int
    CreatedAt     time.Time
}

// CheckInResult 表示一次签到尝试的结果。
type CheckInResult struct {
    CheckedIn      bool // true=本次完成首签, false=今日已签过
    PointsAwarded  int  // 本次实际发放积分（首签=10, 已签=0）
}

// MonthlyCheckIns 表示月度签到查询结果。
type MonthlyCheckIns struct {
    Dates      []string // 已签到日期列表 ["2026-07-01", ...]
    TotalPoints int64   // 用户当前累计积分
}
```

### 2.3 User 模型新增积分字段

- `do.User` 新增 `Points int64`
- `po.User` 新增 `Points int64 `gorm:"not null;default:0"``
- `dto.UserResponse` 新增 `Points int64 `json:"points"``
- `handler.toUserResponse` 映射 `Points` 字段
- 前端 `types.ts` `UserResponse` 新增 `readonly points: number`

## 3. Repository 层

### 3.1 ports 接口

`internal/ports/repositories.go` 新增：

```go
type CheckInRepository interface {
    // CheckInToday 原子地完成当日签到：若 (userID, date) 已存在返回 (false, nil)；
    // 否则插入签到记录并在同一事务内给 user.points 加 pointsAwarded，返回 (true, nil)。
    CheckInToday(ctx context.Context, userID int64, date string, pointsAwarded int) (bool, error)
    // ListByMonth 查询指定用户某年某月的全部签到记录，按日期升序。
    ListByMonth(ctx context.Context, userID int64, year int, month int) ([]do.CheckIn, error)
}
```

### 3.2 repository 实现

`internal/repository/checkin.go`:

- `CheckInToday`：GORM 事务内 `SELECT ... WHERE user_id=? AND check_in_date=?`，存在则返回 false；不存在则 `INSERT` checkin 记录 + `UPDATE user SET points = points + ? WHERE id = ?`，返回 true。
- `ListByMonth`：按 `check_in_date BETWEEN 'YYYY-MM-01' AND 'YYYY-MM-31'` 查询，升序返回。
- `var _ ports.CheckInRepository = (*CheckInRepository)(nil)` 接口实现断言。

## 4. Service 层

`internal/service/checkin.go`:

```go
type CheckInService struct {
    repo        ports.CheckInRepository
    pointsPerDay int
}

func NewCheckInService(repo ports.CheckInRepository) *CheckInService

// CheckInToday 计算亚洲/上海时区当日日期，调用 repo 完成签到。
func (s *CheckInService) CheckInToday(ctx context.Context, userID int64) (do.CheckInResult, error)

// ListMonthly 查询月度签到记录，返回日期列表与用户累计积分。
func (s *CheckInService) ListMonthly(ctx context.Context, userID int64, year int, month int) (do.MonthlyCheckIns, error)
```

- 时区常量：`cstLocation = time.FixedZone("CST", 8*3600)`，`today := time.Now().In(cstLocation).Format("2006-01-02")`。
- `pointsPerDay = 10`，作为 service 常量。
- `ListMonthly` 的 `TotalPoints` 需读取 user.points--service 依赖 `UserRepository.FindByID`。故 `CheckInService` 注入 `CheckInRepository` + `UserRepository`。

## 5. Handler 与路由

### 5.1 UserHandler 扩展

`UserHandler` 新增 `checkInService` 字段，构造函数 `NewUserHandler(userService, checkInService)`。

`Me` 方法改为：

```go
func (h *UserHandler) Me(c context.Context, ctx *app.RequestContext) {
    id, ok := requiredCurrentUserID(c, ctx)
    if !ok { return }
    // 自动签到（幂等，best-effort：签到失败不影响 /me 正常返回）
    if _, err := h.checkInService.CheckInToday(c, id); err != nil {
        logger.Warn(c, "auto check-in failed", zap.Error(err))
    }
    user, err := h.userService.CurrentUser(c, id)
    ...
}
```

新增 `ListCheckIns` 方法处理 `GET /users/me/check-ins?year=2026&month=7`，返回 `MonthlyCheckIns`。

### 5.2 路由注册

`registerUserRoutes` 新增：

```go
users.GET("/me/check-ins", middleware.Auth(jwtManager), userHandler.ListCheckIns)
```

### 5.3 Services 聚合与 wiring

- `router.Services` 新增 `CheckIn *service.CheckInService`。
- `registerUserRoutes` 签名新增 `checkInService *service.CheckInService` 参数。
- `cmd/web/main.go` 构造 `checkInRepo := repository.NewCheckInRepository(db)`，`checkInService := service.NewCheckInService(checkInRepo, userRepo)`，注入 router。

## 6. 前端设计

### 6.1 类型与 API

`types.ts` `UserResponse` 新增 `readonly points: number`。

新增接口：

```typescript
export interface MonthlyCheckInsResponse {
  readonly dates: readonly string[]
  readonly total_points: number
}
```

`client.ts` 新增：

```typescript
export async function getMonthlyCheckIns(year: number, month: number): Promise<MonthlyCheckInsResponse>
```

调用 `GET /users/me/check-ins?year=${year}&month=${month}`。

### 6.2 签到日历组件

新建 `frontend/vue-gallery/src/components/CheckInCalendar.vue`：

- Props: `userId`（变更时重新加载当月）
- 内部状态：当前查看年月 `viewYear`/`viewMonth`、签到日期集合 `Set<string>`、累计积分 `totalPoints`、加载态。
- 月度网格：7 列（周一至周日），渲染当月每日格子，已签到日期高亮。
- 月份切换：上一月/下一月按钮。
- 积分展示：顶部显示"累计积分 {totalPoints}"。
- 今日标记：当日格子额外标记"今日已签到"。
- onMounted 加载当月数据；切换月份重新加载。

### 6.3 AccountPage 集成

- `AccountPage.vue` 登录态 section 内，"登录状态"面板之后插入 `<CheckInCalendar />`。
- aside 账户摘要 `grid-2` 扩展为展示积分（或新增一个摘要块）。

## 7. 数据流

```
用户访问个人中心
  -> AccountPage onMounted -> initAuth -> GET /users/me
     -> Me handler: checkInService.CheckInToday (幂等签到)
     -> userService.CurrentUser (返回含 points 的 UserResponse)
  -> CheckInCalendar onMounted -> GET /users/me/check-ins?year=Y&month=M
     -> ListCheckIns handler -> checkInService.ListMonthly
     -> 返回 { dates, total_points }
```

## 8. 并发与幂等

- `(user_id, check_in_date)` 唯一索引为最终防线：即使并发两次 `/me`，至多一条 INSERT 成功。
- `CheckInToday` 事务内先查后插，唯一索引冲突时捕获并返回 `false`（已签到）。
- 积分更新在事务内 `points = points + 10`，保证一致性。

## 9. 兼容性与迁移

- `po.User` 新增 `Points` 字段，GORM AutoMigrate 自动 `ALTER TABLE` 添加列，默认 0，存量用户积分 0。
- `po.CheckIn` 新表，AutoMigrate 自动创建。
- `dto.UserResponse` 新增 `points` 字段，前端旧版本忽略该字段，向后兼容。

## 10. 验证策略

- 后端单测：`service/checkin_test.go`（mock repo 测试首签/幂等/时区）、`repository/checkin_test.go`（真实 SQLite 测试事务与唯一约束）。
- 前端：`npm run build` 类型检查。
- 集成：启动后端，登录后访问 `/users/me` 验证积分 +10，再访问验证不变；访问 `/users/me/check-ins` 验证日历数据。
