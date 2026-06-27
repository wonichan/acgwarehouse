# 执行计划 - 后端架构重构

## 执行步骤

### Sprint 1：修复依赖 Disorder

#### 1.1 创建 ports 包定义 Repository 接口

**验证命令：**
```bash
go build ./internal/ports/...
```

**检查点：**
- `internal/ports/repositories.go` 文件存在
- 所有 Repository 接口定义完整
- 编译通过

#### 1.2 Service 层改为导入 ports 包

**验证命令：**
```bash
go build ./internal/service/...
```

**检查点：**
- Service 文件不再导入 `internal/repository`
- Service 依赖 `internal/ports` 接口
- 编译通过

#### 1.3 修复 TagRepository 内部实例化

**验证命令：**
```bash
go test ./internal/repository/... -run TestTagRepository
```

**检查点：**
- `TagRepository` 构造函数接收 `ImageRepository` 参数
- `FindActiveImagesWithTags` 使用注入的 repository
- 测试通过

#### 1.4 调整 wiring 层注入

**验证命令：**
```bash
go build ./cmd/...
go test ./... -short
```

**检查点：**
- main/wiring 层正确构造依赖链
- 所有测试通过
- 应用可启动

#### 1.5 验证改进效果

**验证命令：**
```bash
# 运行 Brooks lint 验证 health score 提升
```

**检查点：**
- Health score >= 80
- Dependency Disorder 问题数量减少

### Sprint 2：修复领域模型扭曲

#### 2.1 创建 validators 包

**验证命令：**
```bash
go build ./internal/validators/...
```

**检查点：**
- `internal/validators/image_validator.go` 存在
- `internal/validators/collection_validator.go` 存在
- 编译通过

#### 2.2 从 DO 提取验证方法

**验证命令：**
```bash
go test ./internal/model/do/... -v
go test ./internal/service/... -v
```

**检查点：**
- Validator 实现验证逻辑
- DO 对象移除行为方法
- 测试通过

#### 2.3 Service 调用 Validator

**验证命令：**
```bash
go test ./... -short
```

**检查点：**
- Service 使用 Validator 替代 DO 方法
- 业务逻辑行为不变
- 所有测试通过

### Sprint 3：添加测试性 Seam（可选）

#### 3.1 创建 Repository fake 实现

**检查点：**
- `internal/test/fakes/` 目录存在
- Fake Repository 实现接口
- 编译通过

#### 3.2 Service 测试使用 fake

**验证命令：**
```bash
go test ./internal/service/... -v
```

**检查点：**
- Service 测试无需数据库 fixture
- 测试执行时间减少

## 验证命令汇总

每个步骤完成后运行：

```bash
# 编译检查
go build ./...

# 单元测试
go test ./internal/... -short

# 集成测试
go test ./... -short

# 覆盖率检查
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

## 回滚计划

### Sprint 1 回滚

```bash
git revert <commit-hash>
# 或
git reset --hard <pre-refactor-commit>
```

恢复点：重构前的最后一次提交。

### Sprint 2 回滚

```bash
git revert <commit-hash>
# 移除 validators 包
rm -rf internal/validators/
```

### Sprint 3 回滚

```bash
# 恢复测试配置
git checkout HEAD~1 -- internal/service/*_test.go
```

## 风险缓解

### 风险：重构引入 bug

**缓解：**
- 每个 Sprint 结束后运行完整测试套件
- 保持小步提交，易于定位问题
- 重构不改变业务逻辑，仅改变结构

### 风险：测试覆盖不足

**缓解：**
- 重构前确保现有测试通过
- 为新增接口编写测试
- 使用 coverage 工具检查

### 风险：时间超期

**缓解：**
- 优先完成 Sprint 1（最高优先级）
- Sprint 2 和 3 可并行或延后
- 每个 Sprint 独立可交付

## 里程碑

- [ ] Sprint 1 完成 - 依赖 Disorder 修复
- [ ] Health score >= 80
- [ ] Sprint 2 完成 - 领域模型扭曲修复
- [ ] Sprint 3 完成 - 测试性 Seam 添加
- [ ] 最终验证 - Health score >= 85