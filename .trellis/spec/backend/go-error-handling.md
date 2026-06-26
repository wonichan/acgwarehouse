# 错误处理与堆栈追踪规范

本项目对 Error 的链路追踪有严格要求，AI 在编写错误流时必须遵守以下四条核心原则。

## 1. 核心原则
1. **error必须包含堆栈信息**：所有新抛出的、或包装的错误必须带有时序堆栈。
2. **错误要么返回，要么打印日志**：绝对禁止既抛出 error，又在当前函数打印 error 日志。
3. **分层日志捕获**：底层的错误只管向上 `return` 或 `wrap`，**最终在返回到 handler 层时统一进行 log 打印**。

## 2. 代码实现准则 (Do's & Don'ts)

### 推荐做法 (Do's)
- **新抛出错误**：必须引入 `"github.com/pkg/errors"`，使用 `errors.New()` 抛出以自带堆栈：
  ```go
  return errors.New("foo error")
  ```
- **向下游传播并附加上下文**：使用 errors.WithMessage(err, "context message") 包装错误。
- **错误断言判断**：必须使用标准库的 errors.Is 或 errors.As：
- ```go
  if errors.Is(err, gorm.ErrRecordNotFound) { ... }
  ```
- **纯日志记录（不返回时）**: 如果不需要向上抛出错误，必须使用全局包装过的 log 库记录并携带 ctx：
  ```go
  log.Error(ctx, "foo log err", zap.Error(err))
  ```

### 禁止做法 (Don'ts)
- **Don'ts**: 禁止使用标准库的 errors.New 或 fmt.Errorf 抛出不带堆栈的错误（包装时除外）。

- **Don'ts**: 禁止出现 res (结果) 和 err (错误) 同时为 nil 的返回情况。

- **Don'ts**: 禁止在 service 或 infra 层滥用 log.Error 拦截并打印那些还要继续向上 return 的错误。