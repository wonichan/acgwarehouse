import 'package:flutter/material.dart' as material;
import '../models/image.dart';

class JustifiedImageGrid extends material.StatelessWidget {
  final List<ImageModel> images;
  final material.ScrollController? controller;
  final material.Widget Function(material.BuildContext context, int index) itemBuilder;
  final double targetRowHeight;
  final double spacing;

  const JustifiedImageGrid({
    super.key,
    required this.images,
    required this.itemBuilder,
    this.controller,
    this.targetRowHeight = 220.0,
    this.spacing = 8.0,
  });

  @override
  material.Widget build(material.BuildContext context) {
    return material.LayoutBuilder(
      builder: (context, constraints) {
        final maxWidth = constraints.maxWidth;
        if (maxWidth <= 0) return const material.SizedBox.shrink();

        final rows = _computeRows(maxWidth);

        return material.ListView.builder(
          controller: controller,
          padding: const material.EdgeInsets.all(8),
          itemCount: rows.length,
          itemBuilder: (context, rowIndex) {
            final rowData = rows[rowIndex];
            final isLastRow = rowIndex == rows.length - 1;
            return material.Padding(
              padding: material.EdgeInsets.only(bottom: isLastRow ? 0 : spacing),
              child: _buildRow(context, rowData, maxWidth - 16, isLastRow), // -16 for padding
            );
          },
        );
      },
    );
  }

  List<_RowData> _computeRows(double maxWidth) {
    final availableWidth = maxWidth - 16; // accounting for ListView padding
    final List<_RowData> rows = [];
    List<int> currentRowIndices = [];
    double currentWidth = 0.0;

    for (int i = 0; i < images.length; i++) {
      final image = images[i];
      double aspect = image.width / image.height;
      if (aspect <= 0 || aspect.isNaN) aspect = 1.0;

      currentRowIndices.add(i);
      currentWidth += aspect * targetRowHeight;

      final currentWidthWithSpacing = currentWidth + (currentRowIndices.length - 1) * spacing;

      if (currentWidthWithSpacing >= availableWidth) {
        rows.add(_RowData(indices: currentRowIndices));
        currentRowIndices = [];
        currentWidth = 0.0;
      }
    }

    if (currentRowIndices.isNotEmpty) {
      rows.add(_RowData(indices: currentRowIndices));
    }

    return rows;
  }

  material.Widget _buildRow(material.BuildContext context, _RowData rowData, double maxWidth, bool isLastRow) {
    if (rowData.indices.isEmpty) return const material.SizedBox.shrink();

    double totalAspect = 0.0;
    for (final index in rowData.indices) {
      final img = images[index];
      double aspect = img.width / img.height;
      if (aspect <= 0 || aspect.isNaN) aspect = 1.0;
      totalAspect += aspect;
    }

    final availableWidthForImages = maxWidth - (rowData.indices.length - 1) * spacing;
    
    double rowHeight = availableWidthForImages / totalAspect;
    
    if (isLastRow && (totalAspect * targetRowHeight < availableWidthForImages)) {
      rowHeight = targetRowHeight;
    }

    final children = <material.Widget>[];
    for (int i = 0; i < rowData.indices.length; i++) {
      final index = rowData.indices[i];
      final img = images[index];
      double aspect = img.width / img.height;
      if (aspect <= 0 || aspect.isNaN) aspect = 1.0;
      
      final imageWidth = aspect * rowHeight;

      children.add(
        material.SizedBox(
          width: imageWidth,
          height: rowHeight,
          child: itemBuilder(context, index),
        ),
      );

      if (i < rowData.indices.length - 1) {
        children.add(material.SizedBox(width: spacing));
      }
    }

    return material.Row(
      crossAxisAlignment: material.CrossAxisAlignment.start,
      children: children,
    );
  }
}

class _RowData {
  final List<int> indices;
  _RowData({required this.indices});
}
