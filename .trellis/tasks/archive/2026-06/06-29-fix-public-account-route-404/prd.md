# Fix public frontend route 404

## Goal

Public users can open frontend subroutes such as `https://www.acgwarehouse.cloud/account` directly without receiving a server-level 404.

The fix preserves the existing clean URL routes and makes static-hosted Vue history-mode routes fall back to the SPA entrypoint.

## Confirmed Facts

- Public reproduction: `GET https://www.acgwarehouse.cloud/account` returns HTTP 404, and the same direct-access problem applies to other frontend subroutes.
- Public root route works: `GET https://www.acgwarehouse.cloud/` returns HTTP 200 with the built Vue `index.html`.
- Frontend routes in `frontend/vue-gallery/src/router/index.ts` use `createWebHistory()` and include `/`, `/detail`, `/search`, `/trending`, `/collections`, and `/account`.
- The navigation links to `/account` from `frontend/vue-gallery/src/components/AppHeader.vue`.
- The backend Hertz router only registers API routes under `/api/v1`; it does not serve the Vue app or provide a frontend fallback.
- The repository has no tracked Nginx, Vercel, Netlify, Cloudflare Pages, or other deployment fallback config.
- The public response is behind Cloudflare, but the repository does not identify the origin host.

## Requirements

- Keep frontend subroutes as direct clean URLs rather than switching users to hash routes such as `/#/account`.
- Add repository-owned static hosting fallback configuration so non-API frontend routes serve the SPA entrypoint.
- Do not change API route paths or backend behavior.
- Do not refactor account page functionality; this task only addresses direct route loading.
- Prefer low-risk Vite/static-host metadata that is included in the production build output.

## Acceptance Criteria

- [ ] A production build includes fallback metadata/configuration for static SPA hosting.
- [ ] The fallback routes `/account`, `/search`, `/trending`, `/collections`, and `/detail` to the Vue app entrypoint instead of a static-host 404.
- [ ] Existing API calls remain under `/api/v1/*` and are not rewritten to `index.html` by the added config.
- [ ] Frontend build/type-check passes.
- [ ] Public verification is attempted against `https://www.acgwarehouse.cloud/account`; if deployment has not picked up the change yet, local production-preview verification must demonstrate the direct route works.

## Out of Scope

- Changing the account page UI or authentication flow.
- Replacing Vue Router history mode with hash mode unless static fallback config is impossible.
- Changing Cloudflare DNS, cache settings, or source-host configuration outside the repository.

## Technical Notes

- The implementation will use repository-owned static-host fallback metadata and keep `/api/v1/*` untouched.
- The tracked repository does not identify the exact source host behind Cloudflare, so public verification may require deployment before the live domain reflects the fix.
