---
phase: quick
plan: 40
type: tdd
wave: 1
depends_on: []
files_modified: [
  "flutter_app/lib/providers/navigation_provider.dart",
  "flutter_app/lib/app/fluent_app_shell.dart",
  "flutter_app/lib/app/fluent_screens.dart",
  "flutter_app/lib/widgets/fluent_collections_content.dart",
  "flutter_app/test/app/fluent_app_shell_test.dart",
  "flutter_app/test/widgets/fluent_collections_content_test.dart"
]
autonomous: false
requirements: [QUICK-40]
must_haves:
  truths:
    - "Windows/Fluent 左侧导航出现 收藏 入口"
    - "收藏页可以加载合集列表"
    - "点击合集后可以看到合集内图片或明确的空状态"
    - "继续复用现有 collection API，不引入新的 favorites 持久化语义"
  artifacts:
    - path: "flutter_app/lib/widgets/fluent_collections_content.dart"
      provides: "桌面端收藏/合集浏览内容区"
    - path: "flutter_app/lib/app/fluent_app_shell.dart"
      provides: "收藏导航入口"
    - path: "flutter_app/test/widgets/fluent_collections_content_test.dart"
      provides: "收藏页关键空态/数据态测试"
---

<objective>
修复 Flutter Windows 桌面端“可以收藏但找不到在哪里看收藏”的问题。

输出：左侧导航新增“收藏”，并提供一个最小可用的合集/收藏浏览页。
</objective>

<context>
@.planning/quick/40-flutter-desktop-favorites-entry/40-CONTEXT.md

## 实现策略

采用“合集浏览页作为收藏页”的最小方案：

1. `NavigationProvider` 增加新的收藏索引与标题。
2. `FluentAppShell` 增加 `PaneItem(title: 收藏)`。
3. 新建 `FluentCollectionsContent`，直接用 `CollectionService` 拉取合集与合集图片，避免为这次修复扩散全局 provider 注册面。
4. `fluent_screens.dart` 增加 `FluentCollectionsPage` 作为页面外壳。
5. 先补 widget/shell 测试，再写实现，最后跑 Flutter 定向测试与 analyze。
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: 先补收藏页与导航的失败测试</name>
  <files>
    flutter_app/test/widgets/fluent_collections_content_test.dart,
    flutter_app/test/app/fluent_app_shell_test.dart
  </files>
  <behavior>
    - 收藏页在无合集时显示明确空状态
    - 收藏页在有合集时显示合集列表并可切换到合集图片视图
    - FluentAppShell 暴露新的“收藏”导航项并能切换到收藏页
  </behavior>
</task>

<task type="auto" tdd="true">
  <name>Task 2: 实现桌面收藏页内容与页面外壳</name>
  <files>
    flutter_app/lib/widgets/fluent_collections_content.dart,
    flutter_app/lib/app/fluent_screens.dart
  </files>
  <behavior>
    - 使用现有 CollectionService 加载合集和合集图片
    - 无合集、加载中、合集为空三种状态都有明确反馈
    - 尽量复用现有 FluentImageCard 和图片详情打开模式
  </behavior>
</task>

<task type="auto" tdd="true">
  <name>Task 3: 接入导航并校正共享导航索引</name>
  <files>
    flutter_app/lib/providers/navigation_provider.dart,
    flutter_app/lib/app/fluent_app_shell.dart
  </files>
  <behavior>
    - 添加收藏索引与标题
    - Fluent 左侧导航出现收藏入口
    - 切换收藏页不会影响现有搜索/设置等入口
  </behavior>
</task>

<task type="auto" tdd="true">
  <name>Task 4: 验证与回归</name>
  <files>verify only</files>
  <behavior>
    - 相关 Flutter 测试通过
    - 变更文件 analyze 通过
    - 若 Material 共享导航语义被影响，则补齐最小兼容改动或记录原因
  </behavior>
</task>

</tasks>
