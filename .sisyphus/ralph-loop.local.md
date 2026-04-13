---
active: true
iteration: 7
max_iterations: 500
completion_promise: "DONE"
initial_completion_promise: "DONE"
started_at: "2026-04-13T08:25:15.930Z"
session_id: "ses_27b5c4ceeffeAKtgSdmCD7IA7r"
ultrawork: true
strategy: "continue"
message_count_at_start: 274
---
@docs\superpowers\specs\2026-04-12-hierarchical-tag-governance-design.md   @docs\superpowers\plans\2026-04-12-hierarchical-tag-governance-plan.md   按照这2篇文档 开始开发 Called the Read tool with the following input: {"filePath":"E:\\program\\obsidian\\acg\\acgwarehouse\\docs\\superpowers\\specs\\2026-04-12-hierarchical-tag-governance-design.md"} <path>E:\program\obsidian\acg\acgwarehouse\docs\superpowers\specs\2026-04-12-hierarchical-tag-governance-design.md</path>
<type>file</type>
<content>
1: # Hierarchical Tag Governance Design
2: 
3: **Date:** 2026-04-12
4: 
5: ## Goal
6: 
7: Add hierarchical tag governance to the existing flat tag system with at most three levels: `root`, `parent`, and `child`. Existing tags must migrate to `child` by default. AI and manual tagging must reuse existing tags when matched and only create new tags when no match exists.
8: 
9: ## Confirmed Product Rules
10: 
11: 1. Tag hierarchy is a strict single-parent tree with at most 3 levels.
12: 2. Levels are:
13:    - `root` = 祖级
14:    - `parent` = 父级
15:    - `child` = 子级
16: 3. Existing tags migrate to `child` with no parent.
17: 4. AI-generated tags:
18:    - first try to match an existing tag by label, then alias
19:    - if matched, reuse that exact tag regardless of level
20:    - if not matched, create a new `child` tag
21: 5. Manual tag creation must let the user explicitly choose `root`, `parent`, or `child`.
22: 6. Images may be associated directly with any level (`root`, `parent`, or `child`).
23: 7. Search/filter semantics are hierarchical:
24:    - selecting a tag matches images linked directly to that tag
25:    - and images linked to any descendant tags
26: 8. Filter UI must be a full tree control, not a flat list.
27: 9. Tag governance statistics must update correctly when hierarchy changes.
28: 
29: ## Current-State Constraints From Code
30: 
31: - `tags.preferred_label` is globally unique.
32: - `tags.slug` is globally unique.
33: - aliases resolve directly to a single `tag_id`.
34: - current AI governance flow is label-first, alias-second, create-last.
35: - current SQLite triggers only maintain direct per-tag usage counts from `image_tags`.
36: - current image filtering logic assumes flat tag IDs.
37: 
38: These constraints should remain unless explicitly changed. In particular, global unique labels are important because the approved behavior is “match existing tag, do not create another one.”
39: 
40: ## Data Model Changes
41: 
42: Extend `tags` with:
43: 
44: - `level TEXT NOT NULL` with values `root | parent | child`
45: - `parent_id INTEGER NULL`
46: 
47: V1 should **not** add `root_id`; correctness is more important than denormalized acceleration in the first release.
48: 
49: ### Invariants
50: 
51: - `root` => `parent_id IS NULL`
52: - `parent` => `parent_id` references a `root`
53: - `child` => `parent_id IS NULL` or references a `parent`
54: - no cycles
55: - maximum depth is 3
56: 
57: ### Migration
58: 
59: - backfill all existing tags to `level = 'child'`
60: - backfill all existing tags to `parent_id = NULL`
61: 
62: ## Persistence Semantics
63: 
64: `image_tags` remains the single source of truth for image-to-tag association.
65: 
66: No extra association rows are created for ancestors. If an image is linked to a `child`, the `parent` and `root` match only through hierarchical query expansion, not duplicate rows.
67: 
68: ## Tag Creation and Matching Rules
69: 
70: ### AI flows
71: 
72: AI generation continues to route through `TagGovernanceService.MergeTags`.
73: 
74: For each generated label:
75: 
76: 1. exact-match existing tag by preferred label
77: 2. if not found, match alias
78: 3. if found, reuse that tag regardless of level
79: 4. if not found, create a new tag with:
80:    - `level = child`
81:    - `parent_id = NULL`
82:    - existing pending-review behavior preserved
83: 
84: ### Manual tagging flows
85: 
86: When attaching an existing tag to an image, reuse the selected tag regardless of level.
87: 
88: When manually creating a new tag:
89: 
90: - user must choose `root`, `parent`, or `child`
91: - `root`: no parent allowed
92: - `parent`: must choose a `root`
93: - `child`: may be orphaned or may choose a `parent`
94: 
95: ### Manual create duplicate handling
96: 
97: Manual create must follow the same dedupe-first rule as AI create:
98: 
99: 1. exact-match existing tag by preferred label
100: 2. if not found, match alias
101: 3. if found, reuse that existing tag instead of creating a duplicate
102: 4. if not found, create the requested level subject to hierarchy validation
103: 
104: The UI may still present this as a “create” flow, but the backend behavior is deterministic: matched tag => reuse, unmatched tag => create.
105: 
106: ## Governance Operations
107: 
108: ### Create
109: 
110: Support creating `root`, `parent`, and `child` from governance UI and image-detail tag creation flows.
111: 
112: ### Change level
113: 
114: Allowed with validation:
115: 
116: - `child -> parent`
117: - `child -> root`
118: - `parent -> root`
119: - `parent -> child` only when it has no children
120: - `root -> parent/child` only when it has no descendants
121: 
122: ### Reparent
123: 
124: - `parent` may only be attached to `root`
125: - `child` may only be attached to `parent`
126: - `root` cannot be reparented
127: - reject cycles and depth overflow
128: - `child` may also be detached into an orphan `child` by setting `parent_id = NULL`
129: 
130: ### Delete
131: 
132: Delete preview and delete enforcement must consider:
133: 
134: 1. direct image associations
135: 2. existence of child tags
136: 
137: A tag with children cannot be deleted until the children are moved or removed.
138: A tag with direct image associations cannot be deleted until those direct image-tag links are removed or reassigned.
139: 
140: ### Merge
141: 
142: V1 should allow merge only between tags of the same level.
143: 
144: Reason: cross-level merge makes image semantics ambiguous and complicates tree integrity.
145: 
146: ## Search and Filter Semantics
147: 
148: When one or more tag IDs are selected for filtering, the backend must expand each selected tag into:
149: 
150: - the tag itself
151: - all descendants
152: 
153: Then query `image_tags` against the expanded set and deduplicate image IDs before pagination/response assembly.
154: 
155: ### Multi-select semantics
156: 
157: Existing multi-tag filtering keeps current **AND semantics**.
158: 
159: For each selected tag:
160: 
161: 1. expand that tag to `self + descendants`
162: 2. treat the expanded set as one logical clause
163: 3. an image satisfies that clause if it is associated with any tag in that expanded set
164: 
165: If the user selects both an ancestor and one of its descendants, the descendant selection is logically redundant, but still valid. The query result must remain correct and must not duplicate or exclude images incorrectly.
166: 
167: This same expansion rule should be used by:
168: 
169: - image list filtering
170: - search endpoints with tag filters
171: - AI backfill filters that depend on tag selection
172: 
173: ## Statistics Model
174: 
175: Current trigger-maintained `tags.usage_count` should be treated as **direct usage count** only.
176: 
177: Backward compatibility rule:
178: 
179: - keep returning `usage_count` in existing payloads
180: - define `usage_count` as the direct count in V1
181: - add explicit `direct_*` and `tree_*` fields in governance/statistics/tree-oriented endpoints
182: 
183: V1 statistics should expose both direct and tree aggregates:
184: 
185: - `direct_usage_count`
186: - `tree_usage_count`
187: - `direct_pending_count`
188: - `tree_pending_count`
189: - `direct_confirmed_count`
190: - `tree_confirmed_count`
191: - `direct_ai_count`
192: - `tree_ai_count`
193: - `direct_manual_count`
194: - `tree_manual_count`
195: 
196: ### Direct counts
197: 
198: Maintained by existing SQLite trigger strategy from `image_tags` changes.
199: 
200: ### Tree counts
201: 
202: Computed at runtime in V1 by expanding descendants and deduplicating matched images. Do not add a cache table in the first release.
203: 
204: ### Hierarchy-change behavior
205: 
206: When a tag is upgraded, downgraded, or reparented:
207: 
208: - direct counts do not change unless direct image associations changed
209: - tree counts must reflect the new structure immediately
210: 
211: ## Backend Changes
212: 
213: ### Domain / repository
214: 
215: - extend `internal/domain/tag.go`
216: - update `internal/repository/schema.go`
217: - add migration(s) for new columns and backfill
218: - extend `internal/repository/tag_repository.go`
219: - add repository helpers for:
220:   - list children by parent
221:   - list roots
222:   - list valid parent candidates by target level
223:   - resolve descendants for one or many tags
224: 
225: ### Services
226: 
227: - update `internal/service/tag_governance_service.go`
228: - extend `internal/service/tag_admin_service.go`
229: - update search/image filtering services to expand descendants
230: 
231: ### HTTP API
232: 
233: Existing tag payloads should return hierarchy fields:
234: 
235: - `level`
236: - `parent_id`
237: 
238: Add/extend endpoints for:
239: 
240: - create tag with explicit level
241: - update tag hierarchy metadata
242: - fetch tree data for filter/governance UI
243: - fetch parent candidates
244: - change level
245: - reparent
246: 
247: ## Frontend Changes
248: 
249: ### Shared model/service/provider
250: 
251: - extend `flutter_app/lib/models/tag.dart`
252: - extend `flutter_app/lib/services/tag_service.dart`
253: - extend `flutter_app/lib/providers/tag_provider.dart`
254: 
255: ### Governance UI
256: 
257: The management experience should become tree-aware:
258: 
259: - tree display
260: - create root/parent/child actions
261: - change level action
262: - reparent action
263: - delete preview with child/dependency blockers
264: - same-level merge only
265: 
266: ### Image detail / add-tag flows
267: 
268: - selecting existing tags remains allowed at any level
269: - manual create flow must require choosing level
270: - parent selection UI must appear when required by chosen level
271: 
272: ### Filter UI
273: 
274: Replace flat tag filter UI with a full tree control:
275: 
276: - expand/collapse nodes
277: - show level badges
278: - show `tree_usage_count`
279: - allow multiselect
280: - preserve existing filter semantics, but with hierarchical expansion handled by backend
281: 
282: ## Risks
283: 
284: 1. Duplicate image matches if ancestor/descendant links are not deduplicated.
285: 2. Broken tree invariants during level change or reparent.
286: 3. Existing merge/delete behavior missing child-awareness.
287: 4. Flat assumptions in search and AI backfill filters.
288: 5. UI complexity in tree selection and hierarchy editing.
289: 
290: ## V1 Scope Boundaries
291: 
292: Included:
293: 
294: - three-level hierarchy
295: - direct association to any level
296: - AI reuse existing tags across all levels
297: - manual create with explicit level selection
298: - full tree filter UI
299: - runtime tree stats
300: - same-level merge only
301: 
302: Excluded:
303: 
304: - cached aggregate stats table
305: - cross-level merge rules
306: - more than three levels
307: - multi-parent graph semantics

(End of file - total 307 lines)
</content> Called the Read tool with the following input: {"filePath":"E:\\program\\obsidian\\acg\\acgwarehouse\\docs\\superpowers\\plans\\2026-04-12-hierarchical-tag-governance-plan.md"} <path>E:\program\obsidian\acg\acgwarehouse\docs\superpowers\plans\2026-04-12-hierarchical-tag-governance-plan.md</path>
<type>file</type>
<content>
1: # Hierarchical Tag Governance Implementation Plan
2: 
3: > **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.
4: 
5: **Goal:** Add three-level hierarchical tag governance with tree-based filtering, explicit manual level creation, and hierarchy-aware stats while preserving existing flat-tag behavior where possible.
6: 
7: **Architecture:** Extend the existing `tags` entity with `level` and `parent_id`, keep `image_tags` as the only image-association table, and compute descendant-aware filtering/statistics in backend query/service logic. Preserve current direct-usage trigger behavior while adding runtime tree aggregates for governance and filtering.
8: 
9: **Tech Stack:** Go, Gin, SQLite, Flutter, Provider
10: 
11:
