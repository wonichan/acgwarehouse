# 登录注册与用户个人中心 Design

## Architecture and Boundaries

This task spans the Go backend and the Vue frontend. The backend becomes the source of truth for account profile data; the frontend renders and edits that data through real `/api/v1/users/*` endpoints. The implementation must keep the existing layering: handler parses DTOs and writes responses, service owns business rules, ports define repository contracts, repository maps DO/PO through GORM, and frontend uses `api/client.ts` plus `useAuth.ts` rather than direct `fetch` calls.

Backend changes remain inside the existing user vertical:

- `internal/model/do/user.go`: add public profile fields and preference fields to `do.User`; keep password/password_hash scrubbed by `Public()`.
- `internal/model/po/user.go`: add GORM columns for profile and preferences; rely on existing `AutoMigrate()`.
- `internal/model/dto/user.go`: add response and request DTOs for current user profile update and password change.
- `internal/ports/repositories.go`: extend `UserRepository` with update capabilities.
- `internal/repository/user.go`: map new fields and implement current-user update/password-hash update methods.
- `internal/service/user.go`: validate profile/preference/password inputs and orchestrate repository updates.
- `internal/handler/user.go`: add authenticated endpoints for profile update and password change.
- `internal/handler/router/router.go`: register the new authenticated user routes under `/api/v1/users`.

Frontend changes remain inside the existing Vue app:

- `frontend/vue-gallery/DESIGN.md`: document the Vue app Colorful design system before UI work.
- `frontend/vue-gallery/src/api/types.ts` and `api/client.ts`: add typed DTOs and API methods for user profile and password update.
- `frontend/vue-gallery/src/composables/useAuth.ts`: keep global login state and expose refresh/update helpers only if they represent backend state.
- `frontend/vue-gallery/src/pages/AccountPage.vue`: implement semantic auth/profile/preference/security forms.
- `frontend/vue-gallery/src/components/AppHeader.vue`: show auth-aware account action.
- `frontend/vue-gallery/src/assets/app.css`: add small missing status/form styles only when existing tokens/classes are insufficient.

## API Contracts

Existing endpoints stay compatible:

- `POST /api/v1/users/register` with `{username,password}` returns `UserResponse`.
- `POST /api/v1/users/login` with `{username,password}` returns `{token}`.
- `GET /api/v1/users/me` returns the expanded `UserResponse`.

New endpoints:

- `PUT /api/v1/users/me` requires auth. Request: `{nickname:string, favorite_tags:string, bio:string, public_profile:boolean, email_notifications:boolean, sync_collections:boolean}`. Response: expanded `UserResponse`.
- `PUT /api/v1/users/password` requires auth. Request: `{old_password:string, new_password:string}`. Response: `null` or a small success DTO inside the standard envelope.

Expanded `UserResponse` fields:

- `id`, `username`, `role`, `created_at`
- `nickname`
- `favorite_tags`
- `bio`
- `public_profile`
- `email_notifications`
- `sync_collections`

Default values for existing users:

- `nickname`: default to `username` when empty.
- `favorite_tags`: default empty string.
- `bio`: default empty string.
- `public_profile`: default true.
- `email_notifications`: default true.
- `sync_collections`: default true.

Validation:

- `nickname`: trimmed, 1-20 characters.
- `favorite_tags`: trimmed, max 120 characters. It remains a display/edit string in this pass, not a normalized tag table.
- `bio`: trimmed, max 200 characters.
- `old_password`: must match current password hash.
- `new_password`: min 6 characters, same rule as registration.

Error mapping uses existing user error funnel: invalid input -> HTTP 400 / Code 40001; old password mismatch -> HTTP 401 / Code 40101 with a clear message; unauthenticated -> HTTP 401 / Code 40101; missing user -> HTTP 404 / Code 40401.

## Data Flow

Registration/login/session restore remains the same except `/users/me` now returns complete profile data. On login or restore, `useAuth.user` stores the expanded response and all account panels read from it.

Profile update flow:

1. Account page initializes form refs from `user` after login or restore.
2. User submits the profile/preference form.
3. Frontend validates required/max length fields and calls `updateCurrentUserProfile()`.
4. Backend validates DTO -> service normalizes -> repository updates current user's row.
5. Backend returns expanded `UserResponse`; frontend updates `useAuth.user` and shows inline success + toast.

Password update flow:

1. User submits old/new password form.
2. Frontend validates required/min length and calls `changeCurrentUserPassword()`.
3. Backend verifies old password against `password_hash`, hashes the new password, and updates only the hash.
4. Frontend clears password fields and shows success. Existing JWT may remain valid for this pass.

## UI Design

The Vue UI should follow `frontend/example` while using real backend-backed data. The sidebar displays avatar initial, username, nickname, role, creation date, collection/tag summary placeholders where backend aggregate data is unavailable, and sync/security badges derived from preferences and auth status.

Forms use semantic `<form>` submission, stable ids, helper text, inline status, `aria-invalid` on field errors, and keyboard-accessible tabs. The account page should include login/register, profile/preferences, password security, and recent activity empty state. Recent activity can remain an empty state because no activity feed API exists in scope.

The header keeps navigation unchanged but changes the account action label based on `useAuth()`: logged out shows “登录”; logged in shows nickname, username, or “我的” in that order of preference.

## Compatibility and Migration

Existing tokens remain under `acgwarehouse_token`; existing login/register/me routes stay compatible. GORM `AutoMigrate()` adds nullable/defaulted columns to the existing `user` table. The frontend must tolerate older API responses during development only through defaulting at the API/type mapping boundary if needed; once backend is updated, expanded fields are the contract.

## Trade-offs

This design intentionally stores `favorite_tags` as a string instead of a separate normalized user-tags table. The example represents it as a comma-separated profile preference, and a string keeps the implementation focused on personal-center completeness without creating a new tagging subsystem.

The design includes password change but excludes 2FA and active-session management. Password change is a direct extension of existing credential logic; 2FA/session management would require separate security models and is not necessary to make this account center complete for current sample parity.

## Validation and Rollback

Backend validation must include service/repository or route tests for profile update and password change, plus existing register/login/me paths. Frontend validation must include TypeScript/build checks and browser QA for login, register, profile save, preference save, password change, refresh restore, and logout.

Rollback is vertical: reverting the user model/DTO/repository/service/handler/router additions and frontend account changes restores the previous limited account behavior. Since schema changes are additive via AutoMigrate, rollback does not require dropping columns for local development.
