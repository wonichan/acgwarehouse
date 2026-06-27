# Technical Design - 生产环境部署

## Architecture Overview

```
Internet (HTTPS)
    ↓
Cloudflare (existing)
    ↓
Nginx Reverse Proxy (:443)
    ├─ /           → Frontend Static Files (built Vue app)
    └─ /api/*      → Backend Go Service (:2018)
    
Backend Service (Hertz)
    ├─ Port: 2018 (internal)
    ├─ API: /api/v1/*
    └─ Data: SQLite + Bleve

Frontend (Vue 3)
    ├─ Built to static files
    ├─ API calls via fetch to /api/v1/*
    └─ No SSR, pure SPA
```

## Component Boundaries

### 1. Nginx Configuration

**File**: `/etc/nginx/sites-available/acgwarehouse.cloud`

**Responsibilities**:
- HTTPS termination (using Cloudflare certificates)
- HTTP → HTTPS redirect
- Static file serving for frontend
- Reverse proxy for backend API
- Security headers (HSTS, CSP, X-Frame-Options)

**Routing Rules**:
```nginx
# HTTPS server
server {
    listen 443 ssl http2;
    server_name acgwarehouse.cloud www.acgwarehouse.cloud;
    
    ssl_certificate /etc/ssl/cloudflare/cert.pem;
    ssl_certificate_key /etc/ssl/cloudflare/privkey.pem;
    
    # Frontend static files
    location / {
        root /opt/acgwarehouse/frontend/vue-gallery/dist;
        try_files $uri $uri/ /index.html;
    }
    
    # Backend API proxy
    location /api/ {
        proxy_pass http://127.0.0.1:2018;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# HTTP redirect
server {
    listen 80;
    server_name acgwarehouse.cloud www.acgwarehouse.cloud;
    return 301 https://$server_name$request_uri;
}
```

### 2. Backend Service

**Entry Point**: `cmd/web/main.go`

**Environment Variables**:
```bash
# Server
PORT=2018

# Security
JWT_SECRET=<generate-strong-secret>
JWT_DURATION=168h

# CORS
CORS_ALLOW_ORIGIN=https://acgwarehouse.cloud

# Database
SQLITE_PATH=/opt/acgwarehouse/data/acgwarehouse.db
BLEVE_PATH=/opt/acgwarehouse/data/bleve

# Admin Bootstrap
ADMIN_USERNAME=yachiyo
ADMIN_PASSWORD=YACHIYO

# COS (Tencent Cloud)
COS_SECRET_ID=<from-existing-config>
COS_SECRET_KEY=<from-existing-config>
COS_BUCKET=acgwarehouse-1301393037
COS_REGION=ap-shanghai
COS_DOMAIN=https://acgwarehouse-1301393037.cos.ap-shanghai.myqcloud.com
COS_PREFIX=/thumbnails

# Logging
LOG_LEVEL=info
```

**Systemd Service**: `/etc/systemd/system/acgwarehouse.service`
```ini
[Unit]
Description=ACGWarehouse Backend API Service
After=network.target

[Service]
Type=simple
User=www
Group=www
WorkingDirectory=/opt/acgwarehouse
EnvironmentFile=/opt/acgwarehouse/.env
ExecStart=/opt/acgwarehouse/bin/web
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 3. Frontend API Layer

**New File**: `frontend/vue-gallery/src/api/client.ts`

**Design**:
```typescript
// API base URL - uses relative path in production (same domain)
const API_BASE = '/api/v1'

// Generic fetch wrapper
async function apiCall<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })
  
  if (!response.ok) {
    throw new Error(`API Error: ${response.status}`)
  }
  
  return response.json()
}

// API methods
export const api = {
  // Auth
  login: (username: string, password: string) => 
    apiCall('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
  
  // Images
  getImages: (params?: { tag?: string; page?: number }) =>
    apiCall(`/images?${new URLSearchParams(params)}`),
  
  getImage: (id: string) => apiCall(`/images/${id}`),
  
  // Tags
  getTags: () => apiCall('/tags'),
  
  // Collections
  getCollections: () => apiCall('/collections'),
  
  // Rankings
  getRankings: () => apiCall('/rankings'),
}
```

**Integration Points**:
- Replace Mock data in `GalleryPage.vue` with `api.getImages()`
- Replace Mock data in `SearchPage.vue` with `api.getImages(query)`
- Add login form to `AccountPage.vue` using `api.login()`

## Data Flow

### Request Flow (HTTPS)
```
Browser (https://acgwarehouse.cloud)
    ↓
Cloudflare (existing SSL)
    ↓
Nginx :443
    ├─ GET /         → Static File (dist/index.html)
    └─ GET /api/v1/images → Proxy to localhost:2018
                                    ↓
                            Hertz Server (Go)
                                    ↓
                            SQLite / Bleve
```

### Authentication Flow
```
1. User POST /api/v1/auth/login
2. Backend validates credentials
3. Backend generates JWT token
4. Frontend stores token (localStorage)
5. Subsequent requests include Authorization header
```

## Compatibility & Migration

### No Breaking Changes
- Backend API already implements `/api/v1/*` routes
- CORS middleware already exists in code
- Environment variable loading already implemented
- Admin bootstrap logic already present

### Frontend Changes Required
- Add `src/api/client.ts` for API calls
- Update pages to use real API instead of Mock data
- Add authentication state management
- Handle API error states

## Operational Considerations

### Deployment Order
1. Build frontend → `dist/`
2. Configure nginx
3. Set up environment variables
4. Build backend binary
5. Start backend service
6. Reload nginx
7. Verify HTTPS and API

### Rollback Plan
1. Keep previous nginx config backup
2. Systemd service supports `systemctl restart`
3. Frontend rollback: redeploy previous `dist/`
4. Backend rollback: redeploy previous binary

### Health Checks
- Backend: `GET /api/v1/health` (if exists) or any API endpoint
- Frontend: `GET /` returns HTML
- Nginx: `nginx -t` validates config

### Monitoring Points
- Backend logs: journalctl -u acgwarehouse
- Nginx access logs: `/var/log/nginx/access.log`
- Nginx error logs: `/var/log/nginx/error.log`

## Security Considerations

### HTTPS Configuration
- TLS 1.2+ only
- Strong cipher suites
- HSTS header enabled

### CORS Policy
- Allow only `https://acgwarehouse.cloud`
- No wildcard in production

### JWT Security
- Strong secret key (32+ random bytes)
- Token expiration: 168h (7 days)
- Token stored in httpOnly cookie or localStorage

### File Permissions
- Certificate files: root:root 600
- SQLite database: www:www 640
- Bleve index: www:www 750
- .env file: www:www 600
