---
phase: quick
plan: 3
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/handler/routes.go
autonomous: true
requirements:
  - Fix /admin 404 error
must_haves:
  truths:
    - "GET /admin returns the admin dashboard HTML"
    - "GET /admin/styles.css returns the CSS file"
    - "GET /admin/app.js returns the JavaScript file"
    - "GET /admin/api/* endpoints still work correctly"
  artifacts:
    - path: "internal/handler/routes.go"
      provides: "Admin dashboard route at /admin"
  key_links:
    - from: "GET /admin"
      to: "web/admin/index.html"
      pattern: "r.Static\(\"/admin\""
---

<objective>
Fix the 404 error when accessing http://localhost:8080/admin by correcting the static file route from `/admin-ui` to `/admin`.

Purpose: The admin dashboard is documented to be at `/admin` but currently returns 404 because the route is incorrectly registered as `/admin-ui`.
Output: Working admin dashboard accessible at http://localhost:8080/admin
</objective>

<execution_context>
@./.opencode/get-shit-done/workflows/execute-plan.md
@./.opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@internal/handler/routes.go
@web/admin/index.html

The issue is in `internal/handler/routes.go` line 68:
```go
// Using /admin-ui instead of /admin to avoid route conflicts with /admin/api/* and /api/*
r.Static("/admin-ui", "./web/admin")
```

This should be `/admin` to match the documentation and the HTML file references (`/admin/styles.css`, `/admin/app.js`).
</context>

<tasks>

<task type="auto" tdd="true">
  <name>task 1: Fix admin dashboard route from /admin-ui to /admin</name>
  <files>internal/handler/routes.go</files>
  <behavior>
    - Test: GET /admin should return 200 with HTML content
    - Test: GET /admin/styles.css should return CSS content
    - Test: GET /admin/app.js should return JavaScript content
    - Test: GET /admin/api/summary should still return JSON (API routes must work)
  </behavior>
  <action>
    Change line 68 in `internal/handler/routes.go` from:
    ```go
    r.Static("/admin-ui", "./web/admin")
    ```
    to:
    ```go
    r.Static("/admin", "./web/admin")
    ```
    
    Also update or remove the comment on line 67 since the route conflict concern was unfounded - Gin correctly routes `/admin/api/*` to the API group and `/admin/*` to static files.
  </action>
  <verify>
    <automated>
      # Start the server in background
      go run cmd/server/main.go &
      SERVER_PID=$!
      sleep 2
      
      # Test /admin returns HTML
      curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/admin | grep -q "200" && echo "PASS: /admin returns 200"
      
      # Test /admin/styles.css returns CSS
      curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/admin/styles.css | grep -q "200" && echo "PASS: /admin/styles.css returns 200"
      
      # Test /admin/app.js returns JS
      curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/admin/app.js | grep -q "200" && echo "PASS: /admin/app.js returns 200"
      
      # Test /admin/api/summary still works (will return 401 without auth, which is expected)
      curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/admin/api/summary | grep -q "401" && echo "PASS: /admin/api/summary returns 401 (auth required)"
      
      # Cleanup
      kill $SERVER_PID 2>/dev/null
    </automated>
  </verify>
  <done>
    - Route changed from `/admin-ui` to `/admin`
    - All curl tests pass
    - Admin dashboard accessible at http://localhost:8080/admin
  </done>
</task>

</tasks>

<verification>
After the fix:
1. `curl http://localhost:8080/admin` returns 200 with HTML
2. `curl http://localhost:8080/admin/styles.css` returns 200 with CSS
3. `curl http://localhost:8080/admin/app.js` returns 200 with JavaScript
4. `curl http://localhost:8080/admin/api/summary` returns 401 (auth required, not 404)
</verification>

<success_criteria>
- [ ] Route in routes.go changed from `/admin-ui` to `/admin`
- [ ] Browser can access http://localhost:8080/admin and display the dashboard
- [ ] API endpoints at /admin/api/* continue to work
</success_criteria>

<output>
After completion, create `.planning/quick/3-go-http-localhost-8080-admin-404-page-no/3-SUMMARY.md`
</output>

## Task Dependency Graph

| Task | Depends On | Reason |
|------|------------|--------|
| Task 1 | None | Single atomic fix, no dependencies |

## Parallel Execution Graph

Wave 1 (Start immediately):
└── Task 1: Fix admin dashboard route (no dependencies)

## Commit Strategy

Single atomic commit:
```
fix(routes): serve admin dashboard at /admin instead of /admin-ui

- Changed static file route from /admin-ui to /admin
- Fixes 404 error when accessing http://localhost:8080/admin
- API routes at /admin/api/* continue to work correctly
```
