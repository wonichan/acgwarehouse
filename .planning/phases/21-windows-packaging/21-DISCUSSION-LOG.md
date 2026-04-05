# Phase 21: Windows Packaging - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in `21-CONTEXT.md` — this log preserves the alternatives considered.

**Date:** 2026-04-05
**Phase:** 21-windows-packaging
**Areas discussed:** 交付与安装体验, 分发形态, Python 运行时策略, 目录与数据布局, 失败诊断体验, 目标架构范围, 启动入口形式, 日志与运行时文件位置, 升级覆盖策略

---

## 交付与安装体验

| Option | Description | Selected |
|--------|-------------|----------|
| 标准安装器 | 把快捷方式、卸载入口和依赖检查统一收进安装流程 | |
| 绿色免安装包 | 解压即用、零安装 | ✓ |
| 两种都提供 | 同时提供安装器与绿色包 | |

**User's choice:** 绿色免安装包
**Notes:** 用户明确希望以绿色包作为主交付体验，而不是安装器优先。

---

## 分发形态

| Option | Description | Selected |
|--------|-------------|----------|
| 单个 ZIP 包 | 下载后解压即可运行 | ✓ |
| 自解压 EXE | 用 EXE 包装绿色版 | |
| ZIP + 自解压 EXE | 同时提供两种绿色分发物 | |

**User's choice:** 单个 ZIP 包
**Notes:** 选择最直接的便携分发方式，避免额外包装层。

---

## Python 运行时策略

| Option | Description | Selected |
|--------|-------------|----------|
| 内嵌 Python 运行时 | 随包内置，离线可用 | ✓ |
| 首次运行准备 Python | 首次启动再下载/准备运行时 | |
| Python 打成独立 EXE | 隐藏 Python 运行时感知 | |

**User's choice:** 内嵌 Python 运行时
**Notes:** 明确追求“无需单独安装 Python”的离线体验。

---

## 目录与数据布局

| Option | Description | Selected |
|--------|-------------|----------|
| 程序与用户数据分离 | 程序在解压目录，数据在用户目录 | |
| 全部放解压目录 | 程序、数据、日志、配置都放在包内 | ✓ |
| 默认分离+便携模式 | 同时支持分离与便携 | |

**User's choice:** 全部放解压目录
**Notes:** 用户更看重便携性，接受完全自包含目录布局。

---

## 失败诊断体验

| Option | Description | Selected |
|--------|-------------|----------|
| 明确错误页+日志位置 | 失败时清楚说明组件与日志位置 | ✓ |
| 简短报错+后台日志 | 只给简短错误提示 | |
| 独立诊断器 | 单独提供诊断窗口/工具 | |

**User's choice:** 明确错误页+日志位置
**Notes:** 需要产品级错误提示，不接受仅写日志的黑盒失败体验。

---

## 目标架构范围

| Option | Description | Selected |
|--------|-------------|----------|
| 仅 Windows x64 | 单一桌面主路径 | ✓ |
| x64 + ARM64 | 同时覆盖两种架构 | |
| 先 x64，后续扩展 | 首期 x64，后面再扩 | |

**User's choice:** 仅 Windows x64
**Notes:** 用户接受首期只覆盖主流桌面架构。

---

## 启动入口形式

| Option | Description | Selected |
|--------|-------------|----------|
| 统一启动入口 | 单一入口负责检查与拉起 Go/Python | ✓ |
| 直接点主程序 | 直接运行 Flutter 主程序 | |
| 主程序+诊断入口 | 同时提供常规入口与诊断入口 | |

**User's choice:** 统一启动入口
**Notes:** 用户希望产品只暴露一个清晰入口，减少多进程复杂度外露。

---

## 日志与运行时文件位置

| Option | Description | Selected |
|--------|-------------|----------|
| 放包内固定子目录 | 日志/manifest/诊断文件都留在解压目录内 | ✓ |
| 放用户目录 | 使用 AppData 等用户目录 | |
| 混合布局 | 关键日志和临时文件分开放置 | |

**User's choice:** 放包内固定子目录
**Notes:** 与便携包心智保持一致，方便用户直接定位与反馈问题。

---

## 升级覆盖策略

| Option | Description | Selected |
|--------|-------------|----------|
| 建议新目录解压 | 通过新目录避免覆盖风险 | |
| 允许原地覆盖升级 | 允许用户直接替换旧目录 | ✓ |
| 两者都支持，推荐新目录 | 同时支持但文档偏向新目录 | |

**User's choice:** 允许原地覆盖升级
**Notes:** 这会提高实现与验证复杂度，后续 planning 需要把残留文件、文件占用和数据保护列为显式任务。

---

## the agent's Discretion

- 具体选择哪一套 Windows 打包工具链
- 统一启动入口的工程实现方式
- 包内固定子目录命名与清理规则
- 原地覆盖升级时的文档、检测与回滚提示细节

## Deferred Ideas

- Windows ARM64 打包支持
- 独立诊断器
- 自动更新机制
- Python 故障时自动降级到 Go 后备计算路径
