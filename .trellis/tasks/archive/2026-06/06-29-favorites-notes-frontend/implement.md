# Implementation Plan

## Preconditions

- Active task must remain `planning` until this plan is reviewed and `task.py start` is run.
- Before editing code in Phase 2, load `trellis-before-dev` and the relevant frontend specs.
- Use `visual-engineering` with frontend/taste skills for UI work.

## Ordered Checklist

1. API contracts
   - This frontend package currently has no `test` script or test runner. Do not add new test dependencies by default in this task; rely on build/type checks, API smoke, and browser QA unless the user explicitly approves adding a frontend test harness.
   - Add frontend types for `ImageTagResponse` and tag batch helpers.
   - Add API client helpers for tag create/assign/unassign and reuse the existing `addImageToCollection` helper for collection mutations.
   - Verify no direct `fetch('/api/v1')` calls are introduced outside the API client.

2. Shared workflow UI/state
   - Build a reusable lightweight picker pattern for collection and tag actions using existing `.panel`, `.form-grid`, `.field`, `.status`, `.tag`, and `.btn` classes.
   - Keep it local to affected components unless duplication becomes obvious; avoid adding global state beyond `useSelection` unless necessary.

3. Detail page favorite integration
   - Replace `handleFavorite()` placeholder with real logged-in collection picker flow.
   - Load collections, handle empty collection state, allow creating a collection, and call `addImageToCollection` for the current `imageId`.
   - Refresh detail after success.

4. Detail page tag integration
   - Replace `handleTagging()` placeholder with real tag picker flow.
   - Load/suggest tags, allow new tag creation, assign selected tags, unassign existing tags, and refresh detail after success.

5. Batch panel integration
   - Parse selected IDs from `useSelection()` into positive backend image IDs.
   - Replace fake batch favorite/tag success toasts with real workflows.
   - Clear selection only after successful backend mutation.

6. Collections page management polish
   - Add visibility choice to collection creation.
   - Do not add collection rename/delete or item-removal controls in this task.
   - Do not add fake image thumbnails from collection item IDs.

7. Visual and accessibility pass
   - Apply Colorful/taste polish within `frontend/vue-gallery/DESIGN.md` tokens.
   - Confirm labels, helper text, disabled/loading states, and aria-live/status behavior.
   - Check responsive behavior for detail page and bottom batch panel.

8. Verification
   - Run diagnostics/build for changed frontend files.
   - Run `npm run build` in `frontend/vue-gallery`.
   - Browser-test logged-out and, if credentials/test user are available, logged-in collection/tag mutation flows.

## Validation Commands

```bash
cd frontend/vue-gallery && npm run build
```

If backend/dev server is available:

```bash
curl -i http://localhost:2018/api/v1/tags
curl -i http://localhost:2018/api/v1/collections
```

Browser QA should use the Vite dev server and verify actual `/api/v1/*` network requests through the proxy.

Required validation:

- `npm run build`.
- Browser QA for logged-out states.
- Browser QA for logged-in collection add, tag create+assign, tag remove, and batch collection/tag flows using a real or seeded test account when credentials/data are available.
- Network inspection confirms all mutations use `/api/v1/*` through `client.ts`, with no fake success toasts.

## Atomic Commit Strategy

No commit is made unless the user explicitly asks. If committing later, keep changes atomic:

1. `feat(api): add typed collection and tag mutation helpers`
2. `feat(detail): wire detail collection picker`
3. `feat(detail): wire detail tag picker`
4. `feat(batch): wire batch collection and tag workflows`
5. `feat(collections): add collection visibility choice`
6. `test: record browser/manual verification and final build pass`

## Risky Files

- `frontend/vue-gallery/src/pages/DetailPage.vue`: already owns detail loading, rating, favorite placeholder, and tag placeholder.
- `frontend/vue-gallery/src/components/BatchPanel.vue`: global selected-image action surface.
- `frontend/vue-gallery/src/api/client.ts` and `src/api/types.ts`: shared API surface; type mistakes can break multiple pages.
- `frontend/vue-gallery/src/assets/app.css`: shared visual tokens/classes; keep changes narrow.

## Rollback Points

- Stop after API helper addition if build fails unexpectedly.
- Stop after detail page integration if backend collection/tag contracts differ from DTO evidence.
- Stop after batch panel integration if selection IDs do not reliably map to backend image IDs.

## Review Gates Before Start

- `prd.md`, `design.md`, and `implement.md` reviewed or explicitly approved.
- `implement.jsonl` and `check.jsonl` contain real context entries, not seed examples.
- Task status remains `planning` until approval.
