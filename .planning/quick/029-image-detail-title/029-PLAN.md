# Plan: 图片详情页标题从文件名改为固定名称

## 概述

修改 Flutter 图片详情页的 AppBar 标题，从显示文件名改为固定名称"图片详情"。

## 背景

当前 `image_detail_screen.dart` 第 262 行：
```dart
title: Text(widget.image.filename),
```

文件名可能很长，作为标题显示不够简洁，且文件名信息已在元数据区域展示。

## 任务

### Task 1: 修改 AppBar 标题为固定名称

**文件**: `flutter_app/lib/screens/image_detail_screen.dart`

**修改**:
- 第 262 行：`title: Text(widget.image.filename),` → `title: const Text('图片详情'),`

**预期结果**: AppBar 显示固定标题"图片详情"

## 验证

1. 运行 Flutter 应用
2. 进入任意图片详情页
3. 确认标题显示"图片详情"而非文件名