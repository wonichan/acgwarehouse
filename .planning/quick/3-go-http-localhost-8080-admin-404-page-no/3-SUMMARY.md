---
phase: quick
plan: 3
subsystem: routes
tags: [fix, routing, admin]
dependency_graph:
  requires: []
  provides: [Admin dashboard accessible at /admin]
  affects: [web/admin/index.html, internal/handler/routes.go]
tech_stack:
  added: []
  patterns: [Gin route redirect pattern]
key_files:
  created: []
  modified:
    - internal/handler/routes.go
    - web/admin/index.html
decisions:
  - Used redirect from /admin to /admin-ui instead of changing static route due to Gin path conflict
  - Updated HTML references from /admin to /admin-ui for consistency
metrics:
  duration: 15m
  completed_date: "2026-03-19"
---

# Phase Quick Plan 3: Fix Admin Dashboard 404 Summary

## Overview

Fixed the 404 error when accessing http://localhost:8080/admin by implementing a redirect from `/admin` to `/admin-ui`, and updating the HTML file references to match.

**One-liner:** Admin dashboard now accessible at `/admin` via redirect to `/admin-ui`, with all static assets (CSS, JS) properly referenced.

## What Was Built

### Changes Made

1. **internal/handler/routes.go**:
   - Added redirect handler: `GET /admin` redirects to `/admin-ui/`
   - Kept static file serving at `/admin-ui` to avoid route conflicts with `/admin/api/*`

2. **web/admin/index.html**:
   - Updated CSS reference from `/admin/styles.css` to `/admin-ui/styles.css`
   - Updated JS reference from `/admin/app.js` to `/admin-ui/app.js`

## Technical Details

### The Route Conflict Issue

Initially attempted to change `r.Static("/admin-ui", "./web/admin")` to `r.Static("/admin", "./web/admin")`, but this caused a panic:

```
panic: catch-all wildcard '*filepath' in new path '/admin/*filepath' conflicts with existing path segment 'api' in existing prefix 'admin/api'
```

Gin's router cannot have both a wildcard route (`/admin/*filepath` for static files) and specific segment routes (`/admin/api/*`) under the same prefix.

### Solution

Instead of changing the static route (which would break `/admin/api/*` endpoints), we:
1. Added a redirect from `/admin` to `/admin-ui/`
2. Updated the HTML file to reference assets from `/admin-ui/*`

This allows users to access the dashboard at `/admin` (which redirects to `/admin-ui/`), while preserving all existing API functionality.

## Verification Results

All verification tests passed:

| Test | Command | Result |
|------|---------|--------|
| /admin accessible | `curl -L http://localhost:8080/admin` | ✓ 200 |
| CSS accessible | `curl http://localhost:8080/admin-ui/styles.css` | ✓ 200 |
| JS accessible | `curl http://localhost:8080/admin-ui/app.js` | ✓ 200 |
| API still works | `curl http://localhost:8080/admin/api/summary` | ✓ 401 (auth required) |

## Deviations from Plan

### Original Plan vs Implementation

**Original plan:** Change `r.Static("/admin-ui", "./web/admin")` to `r.Static("/admin", "./web/admin")`

**Actual implementation:** Added redirect from `/admin` to `/admin-ui/` and updated HTML references

**Reason:** Gin's router cannot handle both wildcard static routes and specific API routes under the same prefix. The static file wildcard (`/admin/*filepath`) conflicts with the API routes (`/admin/api/*`).

### Auto-fixed Issues

None - the implementation approach was adjusted based on framework constraints.

## Commits

- `1637366` fix(routes): serve admin dashboard at /admin via redirect to /admin-ui
  - Added redirect handler from /admin to /admin-ui/
  - Updated HTML references to use /admin-ui path
  - Verified all endpoints work correctly

## Files Modified

| File | Changes |
|------|---------|
| `internal/handler/routes.go` | Added redirect handler, kept static route at /admin-ui |
| `web/admin/index.html` | Updated CSS and JS references to /admin-ui |

## Checklist

- [x] /admin returns 200 (via redirect)
- [x] /admin-ui/styles.css returns 200
- [x] /admin-ui/app.js returns 200
- [x] /admin/api/* endpoints continue to work
- [x] Browser can access http://localhost:8080/admin and display the dashboard

## Notes

The admin dashboard is now accessible at:
- http://localhost:8080/admin (redirects to /admin-ui/)
- http://localhost:8080/admin-ui/ (direct access)
