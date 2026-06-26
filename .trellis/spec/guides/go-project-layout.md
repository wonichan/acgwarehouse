# Go 代码组织与标准目录分层规范

AI 在创建新模块、新文件或重构代码时，必须严格遵守以下分层架构。禁止跨层直接调用。

## 1. 标准目录结构树
```
dex-core/
│
├── cmd/
│   └── web/
│       └── main.go                # 服务启动入口
│
├── internal/                      # 内部私有代码
│   ├── conf/                      # 配置管理
│   │   └── conf.go
│   │
│   ├── handler/                   # 传输处理层 (HTTP/gRPC/路由/中间件)
│   │   ├── auth.go
│   │   ├── common.go
│   │   ├── lineage.go
│   │   ├── panic.go
│   │   ├── user.go
│   │   ├── middleware/                # 中间件
│   │           └── authmiddleware.go
│   │   └── router/                    # 路由配置
│   │           └── router.go
│   ├── job/                       # 任务
│   ├── mq/                        # 消息队列
│   │   
│   ├── infra/                     # 基础设施层
│   │   ├── cache/                 # 缓存实现
│   │   ├── client/                # 外部服务客户端
│   │   │   └── cas/               
│   │   │       ├── client.go
│   │   │       └── dto.go
│   │   ├── db/                    # 数据库连接
│   │       ├── mongo.go
│   │       └── mongo_client.go
│   │
│   ├── model/                     # 数据模型
│   │   ├── do/                    # 领域对象
│   │   │   └── user.go
│   │   ├── dto/                   # 数据传输对象
│   │   │   ├── common.go
│   │   │   ├── lineage.go
│   │   │   └── user.go
│   │   └── po/                    # 持久化对象
│   │       ├── lineage.go
│   │       └── user.go
│   │
│   ├── repository/                # 数据访问层
│   │   ├── lineage.go
│   │   └── user.go
│   │
│   │
│   └── service/                   # 业务逻辑层
│       ├── lineage.go             # 接口定义
│       ├── user.go
│       └── lineage/               # 具体实现
│           └── lineageservice.go
│
├── pkg/                           # 可复用公共包
│   ├── errors/                    # 错误处理
│   │   └── error.go
│   ├── logger/                    # 日志工具
│   ├── queue/                     # 队列工具
│   └── util/                      # 通用工具
│
├── third_party/                   # 第三方依赖
│
├── go.mod                         
├── go.sum                         
└── README.md                      # 项目说明
```

## 2. 对象流转硬性约束 (Do's & Don'ts)
- **Do's**: Handler 层必须接收 `dto`，将其转换为 `do` 后调用 `service` 层；`repository` 和 `infra` 层使用 `po` 进行持久化，并向上传递 `do`。
- **Don'ts**: 禁止将 `po`（持久化对象）直接穿透返回给 `handler` 层或作为 HTTP Response 输出。
- **Don'ts**: 禁止在 `service` 层直接编写原生 SQL 或数据库驱动级代码，必须通过 `repository` 接口解耦。