---
phase: quick-16
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/config/config.go
  - internal/worker/ai_tag_handler.go
  - internal/service/admin_service.go
  - cmd/server/main.go
  - deploy/config/config.example.yaml
  - web/admin/index.html
  - web/admin/app.js
autonomous: true
requirements:
  - QUICK-16
must_haves:
  truths:
    - "User can see the current AI tag prompt in Admin Dashboard"
    - "User can customize the prompt via config.yaml"
    - "System uses custom prompt when configured, falls back to default"
  artifacts:
    - path: "internal/config/config.go"
      provides: "TagPrompt config field"
      contains: "TagPrompt string"
    - path: "internal/worker/ai_tag_handler.go"
      provides: "Prompt-aware tag generation"
      pattern: "GenerateTags.*prompt"
    - path: "web/admin/index.html"
      provides: "Prompt display UI"
      contains: "AI 标签提示词"
  key_links:
    - from: "cmd/server/main.go"
      to: "internal/worker/ai_tag_handler.go"
      via: "RegisterAITagHandler with prompt parameter"
    - from: "internal/service/admin_service.go"
      to: "internal/config/config.go"
      via: "AI.TagPrompt field access"
---

# Plan: Customizable AI Tag Prompt

## Context

**User Request:** Support user-customizable AI tag generation prompts, with the default prompt displayed in the Admin Dashboard.

**Key Findings:**
- `DefaultTagPrompt` constant defined in `ai_tag_handler.go` (lines 25-46)
- `GenerateTags` already accepts a `prompt` parameter (line 64)
- `AIConfig` struct needs `TagPrompt` field added
- Admin Dashboard has "系统配置" section where prompt can be displayed
- `ConfigSummary` in `admin_service.go` exposes config to frontend

**Approach:** 
1. Add `TagPrompt` optional field to config with env override
2. Pass prompt to `RegisterAITagHandler`, fallback to `DefaultTagPrompt`
3. Expose both default and configured prompt via Admin API
4. Display in Admin Dashboard with clear default vs customized indication

## Task Dependency Graph

| Task | Depends On | Reason |
|------|------------|--------|
| Task 1: Config + Handler | None | Foundation - defines the prompt field and handler changes |
| Task 2: Admin API | Task 1 | Exposes the prompt through ConfigSummary |
| Task 3: Admin UI | Task 2 | Displays the API response in frontend |

## Parallel Execution Graph

```
Wave 1 (Start immediately):
└── Task 1: Config + Handler (no dependencies)

Wave 2 (After Wave 1):
└── Task 2: Admin API (depends: Task 1)

Wave 3 (After Wave 2):
└── Task 3: Admin UI (depends: Task 2)

Critical Path: Task 1 → Task 2 → Task 3
Sequential execution required due to interface changes.
```

## Tasks

### Task 1: Add TagPrompt Config Field and Update Handler

**Description:** Add `TagPrompt` field to config, update the AI tag handler to use it with fallback to default.

**Delegation Recommendation:**
- Category: `quick` - Simple config addition and handler modification
- Skills: [`test-driven-development`] - Update existing tests for prompt parameter

**Skills Evaluation:**
- ✅ INCLUDED `test-driven-development`: Handler tests need to be updated for the new prompt parameter
- ❌ OMITTED `systematic-debugging`: No bugs to debug, straightforward implementation

**Depends On:** None

**Files:**
- `internal/config/config.go`
- `internal/worker/ai_tag_handler.go`
- `internal/worker/ai_tag_handler_test.go`
- `cmd/server/main.go`
- `deploy/config/config.example.yaml`

**Action:**

1. **config.go** - Add to `AIConfig` struct:
   ```go
   type AIConfig struct {
       Provider          string `yaml:"provider"`
       APIKey            string `yaml:"api_key"`
       Model             string `yaml:"model"`
       RequestsPerMinute int    `yaml:"requests_per_minute"`
       TagPrompt         string `yaml:"tag_prompt"` // Optional custom prompt
   }
   ```

2. **config.go** - Add env override in `applyEnvOverrides`:
   ```go
   if v := os.Getenv("AI_TAG_PROMPT"); v != "" {
       cfg.AI.TagPrompt = v
   }
   ```

3. **ai_tag_handler.go** - Modify `RegisterAITagHandler` signature:
   ```go
   func RegisterAITagHandler(manager *Manager, client ai.AIProvider, obsRepo repository.TagObservationRepository, governance TagGovernanceMerger, customPrompt string) {
       manager.RegisterHandler("ai_tag_generation", func(ctx context.Context, id int64, payload string) error {
           return handleAITagGeneration(ctx, id, payload, client, obsRepo, governance, customPrompt)
       })
   }
   ```

4. **ai_tag_handler.go** - Update `handleAITagGeneration` to use custom prompt:
   ```go
   func handleAITagGeneration(..., customPrompt string) error {
       prompt := customPrompt
       if prompt == "" {
           prompt = DefaultTagPrompt
       }
       result, err := client.GenerateTags(ctx, p.Path, prompt)
       // ...
   }
   ```

5. **main.go** - Update the call to pass the prompt:
   ```go
   func registerAIWorker(manager *worker.Manager, client ai.AIProvider, obsRepo repository.TagObservationRepository, governanceSvc worker.TagGovernanceMerger, cfg *config.Config) {
       worker.RegisterAITagHandler(manager, client, obsRepo, governanceSvc, cfg.AI.TagPrompt)
   }
   ```

6. **config.example.yaml** - Add under `ai:` section:
   ```yaml
   ai:
     # ... existing fields ...
     # Optional: Custom tag generation prompt (defaults to built-in prompt)
     # tag_prompt: ""  # Uncomment and set to customize
   ```

7. **ai_tag_handler_test.go** - Update test calls to pass empty string for prompt parameter.

**Verify:**
```bash
go build ./... && go test ./internal/worker/... -v -run TestRegisterAITagHandler
```

**Done:**
- `TagPrompt` field added to `AIConfig`
- Environment variable `AI_TAG_PROMPT` supported
- Handler uses custom prompt when set, falls back to `DefaultTagPrompt`
- All existing tests pass

---

### Task 2: Expose Tag Prompt via Admin API

**Description:** Add the tag prompt to `ConfigSummary` so the Admin Dashboard can display it.

**Delegation Recommendation:**
- Category: `quick` - Simple struct field addition
- Skills: [] - Straightforward addition, no special skills needed

**Skills Evaluation:**
- ❌ OMITTED `test-driven-development`: No business logic to test, just data exposure

**Depends On:** Task 1

**Files:**
- `internal/service/admin_service.go`

**Action:**

1. **admin_service.go** - Add fields to `ConfigSummary`:
   ```go
   type ConfigSummary struct {
       // ... existing fields ...
       TagPrompt       string `json:"tag_prompt"`
       DefaultTagPrompt string `json:"default_tag_prompt"` // The built-in default for reference
   }
   ```

2. **admin_service.go** - Import the worker package for the default constant:
   ```go
   import "github.com/wonichan/acgwarehouse-backend/internal/worker"
   ```

3. **admin_service.go** - Update `GetSummary` to populate the fields:
   ```go
   Config: ConfigSummary{
       // ... existing fields ...
       TagPrompt:        s.cfg.AI.TagPrompt,
       DefaultTagPrompt: worker.DefaultTagPrompt,
   },
   ```

**Verify:**
```bash
go build ./... && go test ./internal/service/... -v
```

**Done:**
- `ConfigSummary` includes `tag_prompt` and `default_tag_prompt` fields
- API endpoint `/admin/api/summary` returns both values
- Default prompt is always available for reference

---

### Task 3: Display Tag Prompt in Admin Dashboard

**Description:** Add a section in the Admin Dashboard to display the AI tag prompt, showing both default and configured values.

**Delegation Recommendation:**
- Category: `visual-engineering` - UI/frontend work
- Skills: [] - Simple HTML/JS addition

**Skills Evaluation:**
- ❌ OMITTED `frontend-ui-ux`: Simple text display, no complex UI design needed

**Depends On:** Task 2

**Files:**
- `web/admin/index.html`
- `web/admin/app.js`
- `web/admin/styles.css` (if needed for styling)

**Action:**

1. **index.html** - Add new section after "系统配置":
   ```html
   <!-- AI Prompt Section -->
   <section class="section">
       <h2>AI 标签提示词</h2>
       <div class="prompt-info">
           <div class="config-item">
               <span class="config-label">当前状态:</span>
               <span class="config-value" id="promptStatus">-</span>
           </div>
       </div>
       <div class="prompt-display">
           <label class="prompt-label">当前使用的提示词:</label>
           <textarea id="currentPrompt" class="prompt-textarea" readonly rows="8"></textarea>
       </div>
       <p class="prompt-hint">提示: 要自定义提示词，请在 config.yaml 的 <code>ai.tag_prompt</code> 字段中设置，或使用环境变量 <code>AI_TAG_PROMPT</code>。</p>
   </section>
   ```

2. **app.js** - Add elements to the elements object:
   ```javascript
   // Prompt
   promptStatus: document.getElementById('promptStatus'),
   currentPrompt: document.getElementById('currentPrompt'),
   ```

3. **app.js** - Update `renderSummary` function:
   ```javascript
   // Prompt
   const hasCustomPrompt = config.tag_prompt && config.tag_prompt.trim() !== '';
   elements.promptStatus.textContent = hasCustomPrompt ? '✓ 已自定义' : '使用默认提示词';
   elements.promptStatus.className = 'config-value ' + (hasCustomPrompt ? 'status-healthy' : '');
   elements.currentPrompt.value = hasCustomPrompt ? config.tag_prompt : config.default_tag_prompt;
   ```

4. **styles.css** - Add styles (if file exists, append; otherwise inline in HTML):
   ```css
   .prompt-display {
       margin-top: 1rem;
   }
   .prompt-label {
       display: block;
       font-weight: 600;
       margin-bottom: 0.5rem;
       color: #666;
   }
   .prompt-textarea {
       width: 100%;
       padding: 1rem;
       font-family: monospace;
       font-size: 0.9rem;
       border: 1px solid #ddd;
       border-radius: 8px;
       background: #f9f9f9;
       resize: vertical;
   }
   .prompt-hint {
       margin-top: 0.75rem;
       font-size: 0.85rem;
       color: #888;
   }
   .prompt-hint code {
       background: #eee;
       padding: 2px 6px;
       border-radius: 4px;
       font-family: monospace;
   }
   ```

**Verify:**
```bash
# Manual verification - start server and check admin dashboard
go run cmd/server/main.go &
# Open http://localhost:8080/admin and verify prompt section displays correctly
```

**Done:**
- New "AI 标签提示词" section visible in Admin Dashboard
- Shows current status (custom vs default)
- Displays the actual prompt text in a textarea
- Clear instructions for customization via config/env

---

## Commit Strategy

Atomic commits after each task:

```bash
# After Task 1
git add internal/config/config.go internal/worker/ai_tag_handler.go internal/worker/ai_tag_handler_test.go cmd/server/main.go deploy/config/config.example.yaml
git commit -m "feat(ai): add customizable tag prompt support

- Add TagPrompt field to AIConfig with AI_TAG_PROMPT env override
- Update RegisterAITagHandler to accept custom prompt parameter
- Fall back to DefaultTagPrompt when custom prompt not set
- Add tag_prompt field to config.example.yaml"

# After Task 2
git add internal/service/admin_service.go
git commit -m "feat(admin): expose tag prompt in API summary

- Add tag_prompt and default_tag_prompt to ConfigSummary
- Admin dashboard can now display prompt configuration"

# After Task 3
git add web/admin/index.html web/admin/app.js web/admin/styles.css
git commit -m "feat(admin-ui): display AI tag prompt in dashboard

- Add AI 标签提示词 section showing current/default prompt
- Display status indicator (custom vs default)
- Provide instructions for customization"
```

## Success Criteria

1. **Config:** `tag_prompt` field available in `config.yaml` with environment variable override
2. **Backend:** AI tag handler uses configured prompt, falls back to default
3. **API:** `/admin/api/summary` returns both `tag_prompt` and `default_tag_prompt`
4. **UI:** Admin Dashboard displays current prompt with clear indication of default vs custom
5. **Tests:** All existing tests pass with the new handler signature

---

## Output

After completion, create `.planning/quick/16-ai/16-SUMMARY.md`