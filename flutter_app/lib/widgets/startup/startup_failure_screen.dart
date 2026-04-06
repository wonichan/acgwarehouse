import 'package:flutter/material.dart';

import '../../bootstrap/packaged_desktop_bootstrap.dart';

class StartupFailureScreen extends StatelessWidget {
  const StartupFailureScreen({super.key, required this.failure});

  final StartupFailure failure;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 640),
          child: Card(
            margin: const EdgeInsets.all(24),
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    failure.title,
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 12),
                  Text(_labelFor(failure.type)),
                  const SizedBox(height: 12),
                  Text(failure.message),
                  const SizedBox(height: 16),
                  Text('日志', style: Theme.of(context).textTheme.titleMedium),
                  const SizedBox(height: 8),
                  for (final logPath in failure.logPaths)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 6),
                      child: SelectableText(logPath),
                    ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  String _labelFor(StartupFailureType type) {
    return switch (type) {
      StartupFailureType.go => '失败类型: go',
      StartupFailureType.python => '失败类型: python',
      StartupFailureType.startupChain => '失败类型: startup_chain',
    };
  }
}
