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
                  Text('Logs', style: Theme.of(context).textTheme.titleMedium),
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
      StartupFailureType.go => 'Failure class: go',
      StartupFailureType.python => 'Failure class: python',
      StartupFailureType.startupChain => 'Failure class: startup_chain',
    };
  }
}
