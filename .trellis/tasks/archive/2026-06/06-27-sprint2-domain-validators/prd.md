# Sprint 2 - 从 Domain Objects 提取验证逻辑

## 目标

从 Domain Objects 中提取验证和规范化逻辑，保持 DO 对象为纯值类型，提升代码可测试性和可维护性。

## 需求

### 识别需要提取的行为方法

根据 Sprint 1 的 Brooks 审核报告，以下 DO 对象包含行为方法：

- `internal/model/do/image.go`
  - `IsActive()` - 判断图片是否可展示
  - `NormalizeForCreate()` - 创建时规范化字段

- `internal/model/do/ranking.go`
  - `IsValid()` - 判断热榜周期是否有效
  - `Window()` - 返回时间窗口长度
  - `RankingPeriods()` - 返回所有支持的周期

- `internal/model/do/collection.go`
  - `IsValid()` - 判断可见性是否有效
  - `NormalizeForCreate()` - 创建时规范化字段

- `internal/model/do/user.go`
  - `IsValid()` - 判断用户角色是否有效
  - `Public()` - 清除敏感字段

### 分类处理策略

**保留的行为（纯值计算，合理）：**
- `IsActive()` - 纯状态判断，无副作用
- `IsValid()` - 纯枚举验证，无副作用
- `Window()` - 纯计算，无副作用
- `RankingPeriods()` - 纯工厂函数，无副作用
- `Public()` - 返回新对象，无副作用，类似值对象复制

**需要提取的行为（有业务规则）：**
- `NormalizeForCreate()` - 包含默认值设置逻辑，应移至 Validator 或 Service 层

### 验证器设计

创建 `internal/validators/` 包，包含：

1. `ImageValidator` - 图片创建验证
2. `CollectionValidator` - 收藏夹输入验证
3. 现有 Service 层的 `prepare*Input()` 函数可作为参考

## 约束

- 不改变现有业务逻辑行为
- 所有现有测试必须继续通过
- 重构过程中保持代码可编译状态
- 提取后的验证逻辑应可复用

## 接收标准

- [ ] 评估 DO 对象中的行为方法，确定哪些需要提取
- [ ] 创建 `internal/validators/` 包结构
- [ ] 实现必要的验证器
- [ ] 更新 Service 层使用新的验证器
- [ ] 移除 DO 中需要提取的行为方法
- [ ] 所有现有测试通过
- [ ] 代码编译通过

## 备注

- 参考前一个任务的审核报告：`.trellis/tasks/06-27-backend-brooks-review/`
- Sprint 1 已完成依赖 Disorder 修复
- 本任务为 Sprint 2，优先级 4-6
