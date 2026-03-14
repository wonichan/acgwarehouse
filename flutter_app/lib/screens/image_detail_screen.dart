import 'package:flutter/material.dart';
import 'package:extended_image/extended_image.dart';
import '../models/image.dart';

class ImageDetailScreen extends StatelessWidget {
  final ImageModel image;
  
  const ImageDetailScreen({super.key, required this.image});
  
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(image.filename),
      ),
      body: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Image viewer with zoom
            _buildImageViewer(),
            
            // Metadata section
            _buildMetadataSection(context),
            
            // AI Tag placeholder
            _buildAITagPlaceholder(context),
          ],
        ),
      ),
    );
  }
  
  Widget _buildImageViewer() {
    final largeUrl = image.thumbnailLargeUrl;
    
    if (largeUrl == null || largeUrl.isEmpty) {
      return Container(
        height: 300,
        color: Colors.grey[200],
        child: const Center(child: Icon(Icons.image, size: 64, color: Colors.grey)),
      );
    }
    
    return Container(
      constraints: const BoxConstraints(maxHeight: 400),
      child: ExtendedImage.network(
        largeUrl,
        fit: BoxFit.contain,
        mode: ExtendedImageMode.gesture,
        initGestureConfigHandler: (state) {
          return GestureConfig(
            minScale: 0.9,
            animationMinScale: 0.7,
            maxScale: 3.0,
            animationMaxScale: 3.5,
            speed: 1.0,
            inertialSpeed: 100.0,
            initialScale: 1.0,
            inPageView: false,
          );
        },
        loadStateChanged: (state) {
          if (state.extendedImageLoadState == LoadState.loading) {
            return const Center(child: CircularProgressIndicator());
          }
          if (state.extendedImageLoadState == LoadState.failed) {
            return const Center(child: Icon(Icons.error, color: Colors.red));
          }
          return null;
        },
      ),
    );
  }
  
  Widget _buildMetadataSection(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('元数据', style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
          _buildMetadataRow('文件名', image.filename),
          _buildMetadataRow('尺寸', image.displaySize),
          _buildMetadataRow('格式', image.format.toUpperCase()),
          _buildMetadataRow('大小', image.displayFileSize),
          _buildMetadataRow('路径', image.path),
          _buildMetadataRow('导入时间', image.createdAt.toString()),
        ],
      ),
    );
  }
  
  Widget _buildMetadataRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 80,
            child: Text(label, style: const TextStyle(color: Colors.grey)),
          ),
          Expanded(
            child: Text(value, style: const TextStyle(fontWeight: FontWeight.w500)),
          ),
        ],
      ),
    );
  }
  
  Widget _buildAITagPlaceholder(BuildContext context) {
    return Container(
      margin: const EdgeInsets.all(16),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.grey[100],
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.auto_awesome, color: Colors.blue),
              const SizedBox(width: 8),
              Text('AI 标签', style: Theme.of(context).textTheme.titleMedium),
            ],
          ),
          const SizedBox(height: 12),
          const Text(
            'AI 标签生成中...',
            style: TextStyle(color: Colors.grey),
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            children: List.generate(
              3,
              (i) => Container(
                width: 60 + i * 20,
                height: 24,
                decoration: BoxDecoration(
                  color: Colors.grey[300],
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
            ),
          ),
          const SizedBox(height: 12),
          const Text(
            '标签将在 AI 分析完成后显示，您可以在此确认或修改。',
            style: TextStyle(fontSize: 12, color: Colors.grey),
          ),
        ],
      ),
    );
  }
}
