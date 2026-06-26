## 阶段 0 · 地基

- [x] **codegraph 起点**：确认 `.codegraph` 索引就绪（`codegraph status`）；如缺失则 `codegraph init`。

- [x] `go mod init`（module path 待定，建议 `github.com/yachiyo/acgwarehouse`）。
- [ ] 引入依赖：hertz、gorm + modernc.org/sqlite gorm driver、bleve/v2 + cjk、go-pinyin、cos-go-sdk-v5、pkg/errors、zap、golang-jwt、bcrypt(golang.org/x/crypto)。
  - 进度：阶段 00/01 已引入实际用到的 hertz、gorm、纯 Go SQLite(`github.com/glebarez/sqlite`/modernc transitively)、pkg/errors、zap、bcrypt；JWT 为本地 HS256 实现。bleve/pinyin/COS 依赖留到阶段 02 引入，避免地基阶段空依赖。
- [x] `internal/conf`：env + 默认值（COS 凭证、JWT secret、DB/bleve 路径、端口、热榜权重 w1/w2/w3 + 贝叶斯 C + 周期、view flush 间隔、JWT 有效期、ADMIN_USERNAME/PASSWORD）。
- [x] `pkg/logger`：zap 封装，签名对齐 spec `Info/Warn/Error(ctx,msg,...zap.Field)`，禁 fmt.Println。
- [x] `pkg/errors`：复用 github.com/pkg/errors，统一错误码常量（对齐 spec code 矩阵：40001/40101/40301/40401/50001）。
- [x] `infra/db/sqlite.go`：WAL + busy_timeout=5000 + 双连接池（读池 N / 写池 1）+ AutoMigrate 全部 po。
- [x] `handler/common.go`：Response 封装（Success/Fail）、分页解析（page 默认1 / size 默认20 上限100）、列表响应 `{total,page,size,list}`、默认排序 created_at desc。
- [x] CORS 中间件（origin 走 env，开发默认 *）；时间统一 UTC 存储、RFC3339 输出。
- [x] `cmd/web` 优雅关闭骨架：signal.NotifyContext + 按序 flush/stop/close（后续阶段往钩子里挂资源）。
- 验证：`go build ./...`；起一个最小 `cmd/web` 健康检查 `GET /api/v1/ping`；Ctrl-C 能优雅退出。


## 验证命令

```bash
go build ./...
go vet ./...
gofmt -s -l .            # 应无输出
go run ./cmd/sync        # 占位凭证应报错；真实凭证应幂等同步
go run ./cmd/web         # 起服务
```

## 风险点 / 回滚

- SQLite `database is locked`：确认 WAL + busy_timeout + 写池=1（读池并发）+ 事务包裹批量写；view 高频写必须走缓冲批量，不可逐条写。
- bleve cjk 未注册：确认 blank import。
- COS 凭证为占位符：sync 必须显式报错而非静默。
- 每阶段独立可 build，失败回滚到上阶段提交点。

- [x] **codegraph sync**：本阶段完成后同步索引（`codegraph sync`）。
