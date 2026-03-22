# Plan: 修复图片详情页面无法访问的问题

## Problem
用户点击图片无法进入图片详情页面，提示"图片详情功能将在08-03中实现"。

## Root Cause
`flutter_app/lib/app/fluent_screens.dart` 中有两处 `_showImageDetail` 方法（第98-112行和第273-287行）显示了占位对话框，而不是导航到已存在的 `ImageDetailScreen`。

## Solution
将占位对话框替换为实际导航到 `ImageDetailScreen`。

---

## Tasks

### Task 1: 修复 FluentGalleryPage 中的图片详情导航
- **File**: `flutter_app/lib/app/fluent_screens.dart`
- **Location**: Line 98-112, `_showImageDetail` method in `FluentGalleryPage`
- **Action**: Replace the dialog with `Navigator.push` to `ImageDetailScreen`
- **Verify**: Import `ImageDetailScreen` and use correct navigation

### Task 2: 修复 FluentSearchPage 中的图片详情导航  
- **File**: `flutter_app/lib/app/fluent_screens.dart`
- **Location**: Line 273-287, `_showImageDetail` method in `_FluentSearchPageState`
- **Action**: Replace the dialog with `Navigator.push` to `ImageDetailScreen`
- **Verify**: Same as task 1

---

## Expected Result
- 用户在图库页面点击图片 → 进入图片详情页
- 用户在搜索页面点击图片 → 进入图片详情页
- 图片详情页功能正常（显示元数据、AI标签、标签管理等）