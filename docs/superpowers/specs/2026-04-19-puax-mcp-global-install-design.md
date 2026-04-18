# PUAX MCP 全局安装设计

- 日期：2026-04-19
- 主题：在 Claude Code 中以全局方式接入 PUAX MCP，启动方式使用 `npx`

## 目标

让当前机器上的 Claude Code 在所有项目中都可使用 PUAX MCP，而不需要为每个仓库单独配置。

## 推荐方案

将 PUAX MCP server 以用户级配置写入 `C:\Users\Administrator\.claude\settings.json`，通过 `npx puax-mcp-server --stdio` 启动。

这是最符合当前要求的方案，因为用户已经确认：

1. 需要全局可用，而不是仅当前仓库可用。
2. 需要使用 `npx`，而不是本地 clone/build。

## 配置设计

在全局 Claude Code 配置中新增或合并如下配置：

```json
"mcpServers": {
  "puax": {
    "command": "npx",
    "args": ["puax-mcp-server", "--stdio"]
  }
}
```

如果 `settings.json` 中已经存在 `mcpServers`，则只向该对象追加 `puax` 条目；其余 MCP server 配置保持不变。

如果 `settings.json` 中尚不存在 `mcpServers`，则新增该顶层字段，同时保留已有的 `env`、`enabledPlugins`、`extraKnownMarketplaces`、`effortLevel` 等设置。

## 组件与边界

### Claude Code 全局配置

- 文件：`C:\Users\Administrator\.claude\settings.json`
- 作用：保存当前用户级别的 Claude Code 配置
- 本次职责：持久化 `mcpServers.puax` 定义

### PUAX MCP Server

- server 名称：`puax`
- 启动命令：`npx`
- 参数：`puax-mcp-server --stdio`
- 通信方式：stdio

### 当前仓库配置

- 文件：`D:\production\goworkspace\ACGWarehouse\.claude\settings.json`
- 现状：已有项目级权限配置
- 本次职责：不修改

这样可以避免把全局工具接入误放到单仓库设置中。

## 数据流

1. Claude Code 启动或重新加载配置。
2. Claude Code 从 `C:\Users\Administrator\.claude\settings.json` 读取 `mcpServers.puax`。
3. 当需要使用该 MCP server 时，Claude Code 通过 `npx puax-mcp-server --stdio` 启动服务。
4. `npx` 在本机环境中解析并执行 `puax-mcp-server`。
5. Claude Code 通过 stdio 与该 server 通信。

## 失败场景与处理

### Node.js 或 npx 不可用

如果本机缺少 `npx`，则 server 无法启动。此时需要先安装 Node.js/npm 环境。

### 首次启动较慢

首次运行 `npx` 可能需要拉取包，因此会比本地已安装命令更慢。这是接受的权衡，因为当前优先级是低配置成本。

### JSON 合并错误

如果误覆盖现有 `settings.json` 内容，可能会破坏当前全局配置。因此实施时必须在读取现有文件后进行增量合并，并在修改后验证 JSON 结构有效。

### 配置未生效

即使配置正确，当前 Claude Code 会话也可能需要重新加载配置。实施完成后应提示用户重新打开 Claude Code、重启会话，或通过相关界面让配置重新加载。

## 验证方案

实施完成后进行以下验证：

1. 读取 `C:\Users\Administrator\.claude\settings.json`，确认 `mcpServers.puax` 已存在。
2. 验证 JSON 语法有效。
3. 告知用户当前配置已经写入，并提示重新加载 Claude Code 配置。
4. 如有需要，再让用户触发一次 MCP 列表或连接检查，确认 `puax` 已被识别。

## 不做的事

本次不做以下内容：

- 不修改项目级 `.claude/settings.json`
- 不 clone `PUAX` 仓库
- 不执行本地 build
- 不替换现有全局设置
- 不提交 git commit，除非用户单独要求

## 实施边界

本设计只覆盖 Claude Code 的全局 MCP 配置接入，不涉及：

- PUAX server 的源码修改
- 额外鉴权参数注入
- 自动安装 Node.js
- 项目内权限策略调整
