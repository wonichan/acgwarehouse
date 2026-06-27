# Implementation Plan - 生产环境部署

## Execution Checklist

### Phase 1: Frontend API Layer (Priority: High)

**1.1 Create API Client Module**
- File: `frontend/vue-gallery/src/api/client.ts`
- Action: Create TypeScript fetch wrapper with auth support
- Validation: TypeScript compilation passes, no errors
- Command: `cd frontend/vue-gallery && npm run build`

**1.2 Add TypeScript Types for API Responses**
- File: `frontend/vue-gallery/src/api/types.ts`
- Action: Define API response types matching backend models
- Validation: No type errors in IDE, build succeeds

**1.3 Update GalleryPage.vue**
- File: `frontend/vue-gallery/src/pages/GalleryPage.vue`
- Action: Replace Mock data with `api.getImages()`, add loading state
- Validation: Build succeeds, TypeScript no errors

**1.4 Update SearchPage.vue**
- File: `frontend/vue-gallery/src/pages/SearchPage.vue`
- Action: Replace Mock with real API search
- Validation: Build succeeds

**1.5 Update AccountPage.vue**
- File: `frontend/vue-gallery/src/pages/AccountPage.vue`
- Action: Add login form with `api.login()`, auth state management
- Validation: Build succeeds

**1.6 Build Frontend Production Bundle**
- Command: `cd frontend/vue-gallery && npm run build`
- Validation: `dist/` directory created with `index.html`
- Expected output: Build succeeds with no errors

---

### Phase 2: Environment Configuration (Priority: High)

**2.1 Generate JWT Secret**
- Command: `openssl rand -base64 32`
- Action: Generate strong JWT secret key
- Validation: Output is 32+ character random string

**2.2 Create Environment File**
- File: `/opt/acgwarehouse/.env`
- Action: Create with all required variables
- Template:
```bash
PORT=2018
JWT_SECRET=<generated-secret>
JWT_DURATION=168h
CORS_ALLOW_ORIGIN=https://acgwarehouse.cloud
SQLITE_PATH=/opt/acgwarehouse/data/acgwarehouse.db
BLEVE_PATH=/opt/acgwarehouse/data/bleve
ADMIN_USERNAME=yachiyo
ADMIN_PASSWORD=YACHIYO
COS_SECRET_ID=<existing>
COS_SECRET_KEY=<existing>
COS_BUCKET=acgwarehouse-1301393037
COS_REGION=ap-shanghai
COS_DOMAIN=https://acgwarehouse-1301393037.cos.ap-shanghai.myqcloud.com
COS_PREFIX=/thumbnails
LOG_LEVEL=info
```
- Validation: File exists, all variables set

**2.3 Set File Permissions**
- Command: `chmod 600 /opt/acgwarehouse/.env && chown www:www /opt/acgwarehouse/.env`
- Validation: `ls -la` shows correct permissions

---

### Phase 3: Backend Build & Service (Priority: High)

**3.1 Build Backend Binary**
- Command: `go build -o bin/web ./cmd/web`
- Working Dir: `/opt/acgwarehouse`
- Validation: `bin/web` binary exists and is executable

**3.2 Create Systemd Service File**
- File: `/etc/systemd/system/acgwarehouse.service`
- Action: Create systemd service configuration
- Validation: File exists with correct content

**3.3 Enable and Start Service**
- Commands:
```bash
systemctl daemon-reload
systemctl enable acgwarehouse
systemctl start acgwarehouse
systemctl status acgwarehouse
```
- Validation: Service is `active (running)`, no errors in logs

**3.4 Verify Backend Health**
- Command: `curl http://localhost:2018/api/v1/images`
- Validation: Returns JSON response, no errors

---

### Phase 4: Nginx Configuration (Priority: High)

**4.1 Create Nginx Site Config**
- File: `/etc/nginx/sites-available/acgwarehouse.cloud`
- Action: Create nginx configuration from design template
- Validation: File exists

**4.2 Enable Site**
- Command: `ln -sf /etc/nginx/sites-available/acgwarehouse.cloud /etc/nginx/sites-enabled/`
- Validation: Symlink exists

**4.3 Test Nginx Config**
- Command: `nginx -t`
- Validation: Output shows `syntax is ok` and `test is successful`

**4.4 Reload Nginx**
- Command: `systemctl reload nginx`
- Validation: Service reloads without errors

---

### Phase 5: Verification (Priority: Critical)

**5.1 Verify HTTPS Access**
- Command: `curl -I https://acgwarehouse.cloud`
- Validation: Returns 200 OK, SSL certificate valid

**5.2 Verify API Proxy**
- Command: `curl https://acgwarehouse.cloud/api/v1/images`
- Validation: Returns JSON from backend

**5.3 Verify Frontend**
- Command: Browser access to `https://acgwarehouse.cloud`
- Validation: Vue app loads, no console errors

**5.4 Verify CORS**
- Test: Frontend can call API without CORS errors
- Validation: Network tab shows successful API calls

**5.5 Verify Admin Login**
- Test: Login with `yachiyo` / `YACHIYO`
- Validation: JWT token returned, auth works

---

## Validation Commands Summary

After all phases complete, run:

```bash
# Backend health
curl http://localhost:2018/api/v1/images

# HTTPS access
curl -I https://acgwarehouse.cloud

# API proxy
curl https://acgwarehouse.cloud/api/v1/images

# Nginx status
systemctl status nginx

# Backend service status
systemctl status acgwarehouse

# Logs check
journalctl -u acgwarehouse --no-pager -n 50
```

## Rollback Points

| Phase | Rollback Action |
|-------|----------------|
| Frontend Build | Keep previous `dist/` backup |
| Backend Build | Keep previous `bin/web` backup |
| Nginx Config | `rm /etc/nginx/sites-enabled/acgwarehouse.cloud && systemctl reload nginx` |
| Systemd Service | `systemctl stop acgwarehouse && systemctl disable acgwarehouse` |

## Risky Files

| File | Risk | Mitigation |
|------|------|------------|
| `.env` | Contains secrets | Ensure chmod 600, never commit |
| `nginx config` | Syntax errors | Always run `nginx -t` before reload |
| `systemd service` | Service fails to start | Check logs with journalctl |

## Dependencies

- Phase 1 (Frontend) → Phase 5 (Verification): Frontend must be built before final testing
- Phase 2 (Env) → Phase 3 (Backend): Backend needs env vars to start
- Phase 3 (Backend) → Phase 5: Backend must be running for API tests
- Phase 4 (Nginx) → Phase 5: Nginx must proxy correctly for HTTPS access

## Estimated Time

- Phase 1: 30-60 min (frontend API layer development)
- Phase 2: 5 min (env setup)
- Phase 3: 10 min (backend build + service)
- Phase 4: 5 min (nginx config)
- Phase 5: 10 min (verification)

**Total**: ~60-90 minutes
