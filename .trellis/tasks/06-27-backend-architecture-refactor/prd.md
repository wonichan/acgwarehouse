# 后端代码架构重构 - 修复 Brooks 审核问题

## 目标

根据 Brooks lint 审核报告发现的问题，对 Go 后端代码进行架构重构，修复依赖 disorder、领域模型扭曲和测试性缺失问题，提升代码质量和可测试性。

## 需求

### Sprint 1 - 修复依赖 Disorder（优先级 7-9，严重）

- 在 `internal/ports/` 包中定义 Repository 接口，与具体实现分离
- 修复 `TagRepository` 中创建 `ImageRepository` 实例的问题，改为依赖注入
- 移除 Service 层对 Repository 包的直接导入，改为使用 ports 包中的接口
- 将 GORM 相关导入移至 wiring boundary（handler/main 层）

### Sprint 2 - 修复领域模型扭曲（优先级 4-6，计划）

- 从 Domain Objects 中提取验证逻辑到 `internal/validators/` 包
- 保持 DO 对象为纯值类型，无行为方法
- 为创建时的规范化引入 Builder 模式

### Sprint 3 - 添加测试性 Seam（优先级 4-6，计划）

- 在 Repository 边界定义接口抽象，支持测试替身
- 注入 RepositoryFactory 用于测试场景
- Service 测试不应依赖真实数据库

## 约束

- 不改变现有业务逻辑行为
- 保持 API 兼容性
- 所有现有测试必须继续通过
- 重构过程中保持代码可编译状态

## 接收标准

### Sprint 1 验收

- [ ] 创建 `internal/ports/repositories.go` 文件，定义所有 Repository 接口
- [ ] `TagRepository.FindActiveImagesWithTags()` 使用注入的 `ImageRepository`
- [ ] Service 层不再导入 `internal/repository` 包
- [ ] GORM 导入仅出现在 `internal/repository` 和 `cmd/` wiring 层
- [ ] 所有现有测试通过
- [ ] Brooks health score 从 70 提升至 80+

### Sprint 2 验收

- [ ] 创建 `internal/validators/` 包，包含 `ImageValidator`、`CollectionValidator` 等
- [ ] DO 对象仅保留纯值类型定义，移除所有行为方法
- [ ] 创建时规范化逻辑移至 Builder 或 Validator
- [ ] 所有现有测试通过

### Sprint 3 验收

- [ ] Repository 接口可被 mock 替换
- [ ] Service 测试无需数据库 fixture
- [ ] 测试执行时间减少 30%+

## 备注

- 参考审核报告：`.trellis/tasks/06-27-backend-brooks-review/` 目录下的详细分析
- 优先修复优先级 7-9 的严重问题
- 每个 Sprint 完成后运行 Brooks lint 验证改进效果