# Login Register Profile Center Implementation Plan

> **For agentic workers:** REQUIRED: Use `trellis-before-dev` before editing and `trellis-check` after editing. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete login, registration, and the user profile center end-to-end with backend persistence and Vue UI parity against `frontend/example`.

**Architecture:** Extend the existing Go user vertical with profile/preferences/password APIs, then connect the Vue account page to those APIs through the existing API client and auth composable. Keep handler/service/repository boundaries intact and document the Colorful frontend design system before UI edits.

**Tech Stack:** Go + Hertz + GORM + SQLite AutoMigrate; Vue 3 `<script setup>` + TypeScript + Vue Router + Vite; shared CSS in `frontend/vue-gallery/src/assets/app.css`.

---

## Files

- Modify: `internal/model/do/user.go` for profile/preference fields and public scrubbing.
- Modify: `internal/model/po/user.go` for persisted profile/preference columns.
- Modify: `internal/model/dto/user.go` for expanded response and new request DTOs.
- Modify: `internal/ports/repositories.go` for user update repository methods.
- Modify: `internal/repository/user.go` for mapping and persistence updates.
- Modify: `internal/service/user.go` for profile/password business rules.
- Modify: `internal/handler/user.go` for authenticated profile/password handlers.
- Modify: `internal/handler/router/router.go` for new user routes.
- Create/modify tests near existing user service/repository/handler/router tests.
- Create: `frontend/vue-gallery/DESIGN.md` for the Vue app design-system contract.
- Modify: `frontend/vue-gallery/src/api/types.ts` and `frontend/vue-gallery/src/api/client.ts` for expanded user/profile/password APIs.
- Modify: `frontend/vue-gallery/src/composables/useAuth.ts` for expanded user state and refresh/update helpers.
- Modify: `frontend/vue-gallery/src/pages/AccountPage.vue` for semantic auth/profile/preference/password UI.
- Modify: `frontend/vue-gallery/src/components/AppHeader.vue` for auth-aware account action text.
- Modify if needed: `frontend/vue-gallery/src/assets/app.css` for small missing state styles.

## Task 1: Backend Failing Tests

- [ ] Add service tests for profile validation: valid update, empty nickname, too-long tags, too-long bio.
- [ ] Add service tests for password change: old password mismatch, new password too short, successful hash update.
- [ ] Add repository tests or route smoke tests proving expanded `UserResponse` fields persist through GORM.
- [ ] Add route smoke tests for `GET /users/me`, `PUT /users/me`, and `PUT /users/password` auth/error/success paths where existing test utilities allow.
- [ ] Run the targeted Go tests and confirm new tests fail for missing methods/routes/fields.

## Task 2: Backend Domain and Persistence

- [ ] Extend `do.User` with `Nickname`, `FavoriteTags`, `Bio`, `PublicProfile`, `EmailNotifications`, and `SyncCollections`.
- [ ] Extend `po.User` with matching GORM columns and defaults where appropriate.
- [ ] Extend `dto.UserResponse` and add `UserProfileUpdateRequest` plus `UserPasswordUpdateRequest`.
- [ ] Update `repository.toDO()` and `repository.toPO()` mappings.
- [ ] Extend `ports.UserRepository` and `service.UserRepository` with focused methods for profile update and password hash update.
- [ ] Implement repository update methods without returning PO objects across layer boundaries.

## Task 3: Backend Service and Handler

- [ ] Add service method for updating current user profile/preferences with trimming and max-length validation.
- [ ] Add service method for changing password by checking old password with bcrypt and hashing the new password.
- [ ] Add handler methods to bind DTOs, read current user id from auth context, call service, and return standard responses.
- [ ] Extend `writeUserError()` only with necessary typed/sentinel cases, preserving existing mappings.
- [ ] Register `PUT /api/v1/users/me` and `PUT /api/v1/users/password` under `middleware.Auth(jwtManager)`.
- [ ] Run targeted Go tests; then run the relevant package tests.

## Task 4: Frontend Design System Gate

- [ ] Read `frontend/example/DESIGN.md`, `frontend/vue-gallery/src/assets/app.css`, `.trellis/spec/frontend/index.md`, and `.trellis/spec/frontend/api-client.md`.
- [ ] Create `frontend/vue-gallery/DESIGN.md` with Colorful tokens and component rules for account forms, profile panels, statuses, and header account actions.

## Task 5: Frontend API and Auth State

- [ ] Extend `UserResponse` in `frontend/vue-gallery/src/api/types.ts` with backend profile/preference fields.
- [ ] Add typed request shapes for profile update and password change.
- [ ] Add `updateCurrentUserProfile()` and `changeCurrentUserPassword()` to `api/client.ts` using `/users/me` and `/users/password`.
- [ ] Update `useAuth.ts` so login/register/init use expanded user data and expose a refresh or update helper if needed.
- [ ] Keep type imports as `import type`; do not add type suppression.

## Task 6: Frontend Account Page

- [ ] Convert login and register panes from click-only containers to semantic `<form>` elements with `@submit.prevent`.
- [ ] Add stable tab ids and correct `aria-labelledby` / `aria-controls` links.
- [ ] Add helper text, inline status, field-level errors, `aria-invalid`, disabled/loading states, and toast feedback.
- [ ] Add backend-backed profile and preference forms initialized from `user`.
- [ ] Add password-change form for old/new password with validation and backend submission.
- [ ] Add recent activity empty state from the example using SVG, not emoji.
- [ ] Ensure successful profile save updates displayed user state without refresh; refresh still restores persisted values from backend.

## Task 7: Frontend Header

- [ ] In `AppHeader.vue`, import `useAuth()` and derive `user` / `isLoggedIn`.
- [ ] Keep the account action as a `RouterLink` to `/account`.
- [ ] Render “登录” when logged out and nickname/username/“我的” when logged in.
- [ ] Preserve current nav labels, active-route behavior, and search-mini input.

## Task 8: Validation

- [ ] Run Vue diagnostics on changed `.vue` files.
- [ ] Run Go diagnostics or `gopls` diagnostics on changed Go files.
- [ ] Run targeted Go tests for user service/repository/routes.
- [ ] Run `go test ./...` from repo root if feasible.
- [ ] Run `npm run build` in `frontend/vue-gallery`; expected result: `vue-tsc -b && vite build` exits 0.
- [ ] Launch backend and frontend when feasible, then browser-test `/account` at mobile and desktop widths. Cover logged-out initial state, registration, login, profile save, preference save, password change, refresh restore, logout, and validation/API errors.

## Review Gate Before Start

- [ ] Confirm `prd.md`, `design.md`, `implement.md`, `implement.jsonl`, and `check.jsonl` are present and consistent with full frontend+backend scope.
- [ ] Get user approval to start implementation or an explicit “开始实现/继续” instruction before running `task.py start`.
