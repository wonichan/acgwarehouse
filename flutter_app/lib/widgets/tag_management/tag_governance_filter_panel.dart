import 'package:fluent_ui/fluent_ui.dart';
import '../../models/tag_governance_filter.dart';

class TagGovernanceFilterPanel extends StatefulWidget {
  final TagGovernanceFilterState draftFilter;
  final TagGovernanceFilterState appliedFilter;
  final ValueChanged<TagGovernanceFilterState> onDraftChanged;
  final VoidCallback onApply;
  final VoidCallback onReset;

  const TagGovernanceFilterPanel({
    super.key,
    required this.draftFilter,
    required this.appliedFilter,
    required this.onDraftChanged,
    required this.onApply,
    required this.onReset,
  });

  @override
  State<TagGovernanceFilterPanel> createState() =>
      _TagGovernanceFilterPanelState();
}

class _TagGovernanceFilterPanelState extends State<TagGovernanceFilterPanel> {
  late TextEditingController _minUsageController;
  late TextEditingController _maxUsageController;

  @override
  void initState() {
    super.initState();
    _minUsageController = TextEditingController(
      text: widget.draftFilter.minUsageCount?.toString() ?? '',
    );
    _maxUsageController = TextEditingController(
      text: widget.draftFilter.maxUsageCount?.toString() ?? '',
    );
  }

  @override
  void didUpdateWidget(covariant TagGovernanceFilterPanel oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.draftFilter.minUsageCount != oldWidget.draftFilter.minUsageCount) {
      final text = widget.draftFilter.minUsageCount?.toString() ?? '';
      if (_minUsageController.text != text) {
        _minUsageController.text = text;
      }
    }
    if (widget.draftFilter.maxUsageCount != oldWidget.draftFilter.maxUsageCount) {
      final text = widget.draftFilter.maxUsageCount?.toString() ?? '';
      if (_maxUsageController.text != text) {
        _maxUsageController.text = text;
      }
    }
  }

  @override
  void dispose() {
    _minUsageController.dispose();
    _maxUsageController.dispose();
    super.dispose();
  }

  void _toggleLevel(String level) {
    final levels = Set<String>.from(widget.draftFilter.levels);
    if (levels.contains(level)) {
      levels.remove(level);
    } else {
      levels.add(level);
    }
    widget.onDraftChanged(widget.draftFilter.copyWith(levels: levels));
  }

  void _updateMinUsage(String value) {
    final parsed = int.tryParse(value);
    widget.onDraftChanged(
      widget.draftFilter.copyWith(
        minUsageCount: parsed,
        clearMinUsage: parsed == null,
      ),
    );
  }

  void _updateMaxUsage(String value) {
    final parsed = int.tryParse(value);
    widget.onDraftChanged(
      widget.draftFilter.copyWith(
        maxUsageCount: parsed,
        clearMaxUsage: parsed == null,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildLevelChips(),
            const SizedBox(height: 8),
            _buildToggles(),
            const SizedBox(height: 8),
            _buildUsageRange(),
            const SizedBox(height: 8),
            _buildActions(),
            if (widget.appliedFilter.isNotEmpty) ...[
              const SizedBox(height: 8),
              _buildSummaryChips(),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildLevelChips() {
    return Row(
      children: [
        Text('层级:', style: TextStyle(fontWeight: FontWeight.bold)),
        const SizedBox(width: 8),
        _buildLevelChip('祖级', 'root'),
        const SizedBox(width: 4),
        _buildLevelChip('父级', 'parent'),
        const SizedBox(width: 4),
        _buildLevelChip('子级', 'child'),
      ],
    );
  }

  Widget _buildLevelChip(String label, String level) {
    final isSelected = widget.draftFilter.levels.contains(level);
    return ToggleButton(
      checked: isSelected,
      onChanged: (_) => _toggleLevel(level),
      child: Text(label),
    );
  }

  Widget _buildToggles() {
    return Row(
      children: [
        ToggleSwitch(
          checked: widget.draftFilter.orphanOnly,
          onChanged: (v) => widget.onDraftChanged(
            widget.draftFilter.copyWith(orphanOnly: v),
          ),
        ),
        const SizedBox(width: 6),
        Text('无父级'),
        const SizedBox(width: 20),
        ToggleSwitch(
          checked: widget.draftFilter.sourceAI,
          onChanged: (v) => widget.onDraftChanged(
            widget.draftFilter.copyWith(sourceAI: v),
          ),
        ),
        const SizedBox(width: 6),
        Text('AI 生成'),
        const SizedBox(width: 20),
        ToggleSwitch(
          checked: widget.draftFilter.sourceManual,
          onChanged: (v) => widget.onDraftChanged(
            widget.draftFilter.copyWith(sourceManual: v),
          ),
        ),
        const SizedBox(width: 6),
        Text('手动生成'),
      ],
    );
  }

  Widget _buildUsageRange() {
    return Row(
      children: [
        Text('使用量:', style: TextStyle(fontWeight: FontWeight.bold)),
        const SizedBox(width: 8),
        SizedBox(
          width: 80,
          child: TextBox(
            controller: _minUsageController,
            placeholder: '最小',
            onChanged: _updateMinUsage,
            keyboardType: TextInputType.number,
          ),
        ),
        const SizedBox(width: 4),
        Text('~'),
        const SizedBox(width: 4),
        SizedBox(
          width: 80,
          child: TextBox(
            controller: _maxUsageController,
            placeholder: '最大',
            onChanged: _updateMaxUsage,
            keyboardType: TextInputType.number,
          ),
        ),
      ],
    );
  }

  Widget _buildActions() {
    return Row(
      children: [
        FilledButton(
          onPressed: widget.onApply,
          child: Text('应用筛选'),
        ),
        const SizedBox(width: 8),
        Button(
          onPressed: widget.onReset,
          child: Text('重置'),
        ),
      ],
    );
  }

  Widget _buildSummaryChips() {
    final chips = widget.appliedFilter.summaryChips;
    final theme = FluentTheme.of(context);
    return Wrap(
      spacing: 6,
      runSpacing: 4,
      children: chips
          .map(
            (chip) => Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: theme.accentColor.withValues(alpha: 0.16),
                borderRadius: BorderRadius.circular(4),
                border: Border.all(
                  color: theme.accentColor.withValues(alpha: 0.35),
                  width: 0.5,
                ),
              ),
              child: Text(
                chip,
                style: TextStyle(
                  fontSize: 11,
                  color: theme.resources.textFillColorPrimary,
                ),
              ),
            ),
          )
          .toList(),
    );
  }
}
