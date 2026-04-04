# Roadmap: ACGWarehouse

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-03-19)
  - 详见: `.planning/milestones/v1.0-ROADMAP.md`
- ✅ **v2.0 UI/UX 重构** — Phases 7-10 (shipped 2026-03-22)
  - 详见: `.planning/milestones/v2.0-ROADMAP.md`
- ✅ **v3.0 导入后任务平台化** — Phases 11-14 (shipped 2026-03-29)
  - 详见: `.planning/milestones/v3.0-ROADMAP.md`
- 🚧 **v4.0 Windows Photos 风格重构与计算层拆分** — Phases 15-22 (in progress)

## Phases

**Phase Numbering:**
- Integer phases (15, 16, 17...): Planned v4.0 milestone work
- Decimal phases (15.1, 15.2): Urgent insertions after planning (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 15: Compute Sidecar Infrastructure** - Establish Go ↔ Python process lifecycle and communication foundation (completed 2026-04-03)
- [x] **Phase 16: Duplicate Detection Migration** - Migrate duplicate detection computation to Python sidecar (completed 2026-04-04)
- [x] **Phase 17: Desktop Shell Foundation** - Deliver Windows Photos-style toolbar and grid browsing (completed 2026-04-05)
- [x] **Phase 18: Independent Viewer & Filmstrip** - Implement non-blocking viewer with filmstrip navigation (completed 2026-04-05)
- [ ] **Phase 19: Tag Management** - Enable in-app tag management without legacy entry switching
- [ ] **Phase 20: Operations Monitoring** - Integrate task monitoring and sidecar diagnostics into desktop UI
- [ ] **Phase 21: Windows Packaging** - Package Flutter + Go + Python as single distributable bundle
- [ ] **Phase 22: Large Gallery Performance** - Optimize browsing responsiveness for 10k+ image galleries

## Phase Details

### Phase 15: Compute Sidecar Infrastructure
**Goal**: Establish reliable process orchestration so Go and Python services can start, communicate, and remain healthy
**Depends on**: Phase 14 (v3.0 task platform foundation)
**Requirements**: COMP-01, COMP-02, COMP-06
**Success Criteria** (what must be TRUE):
  1. User can start the desktop application and see all services ready within acceptable startup time
  2. Application enters usable state only after both Go and Python confirm readiness
  3. Flutter frontend can connect to Go backend without relying on hardcoded fixed ports
  4. Admin can observe Python sidecar running status through process monitoring
**Plans**: 3 plans
Plans:
- [x] `15-01-PLAN.md` — Build Go-owned sidecar lifecycle, startup/shutdown orchestration, and layered health boundaries
- [x] `15-02-PLAN.md` — Implement runtime manifest generation and Flutter startup discovery without hardcoded ports
- [x] `15-03-PLAN.md` — Add admin diagnostics, degraded-mode regressions, and phase-closing cross-stack verification
**UI hint**: no

### Phase 16: Duplicate Detection Migration
**Goal**: Python sidecar handles duplicate detection computation with clear recommendations for each group
**Depends on**: Phase 15 (sidecar infrastructure ready)
**Requirements**: COMP-03, COMP-04
**Success Criteria** (what must be TRUE):
  1. User can trigger duplicate detection and receive results computed by Python sidecar
  2. User can see recommended keep items for each duplicate group with clear rationale
  3. System provides diagnosable failure status when Python sidecar is unavailable
**Plans**: 3 plans
Plans:
- [x] `16-01-PLAN.md` — Build Python sidecar duplicate detection compute pipeline (hashing, grouping, scoring, async task endpoints)
- [x] `16-02-PLAN.md` — Extend Go domain models and SQLite schema for 256-bit pHash and recommendation rationale
- [x] `16-03-PLAN.md` — Wire Go to Python sidecar: HTTP client, service refactor, handler pre-check, old code deletion
**UI hint**: no

### Phase 17: Desktop Shell Foundation
**Goal**: Users can browse and filter the gallery in Windows Photos-inspired desktop interface
**Depends on**: Phase 16 (compute layer stable)
**Requirements**: DSK-01, DSK-02, DSK-03
**Success Criteria** (what must be TRUE):
  1. User can access search, import, and settings from top toolbar without navigating away from gallery
  2. User can browse images in square grid layout with consistent tile sizes
  3. User can filter gallery content by selecting tags from accessible filter panel
**Plans**: 3 plans
Plans:
- [x] `17-01-PLAN.md` — Build the custom desktop top bar, search handoff, and shell action contract
- [x] `17-02-PLAN.md` — Convert the gallery into a grid-first workspace with a persistent right filter panel
- [x] `17-03-PLAN.md` — Wire the top-bar import action to a real backend endpoint with lightweight feedback
**UI hint**: yes

### Phase 18: Independent Viewer & Filmstrip
**Goal**: Users can inspect images in non-blocking viewer windows with quick filmstrip navigation
**Depends on**: Phase 17 (desktop shell foundation)
**Requirements**: VIEW-01, VIEW-02, VIEW-03, VIEW-04
**Success Criteria** (what must be TRUE):
  1. User can double-click image to open in separate viewer window without blocking main gallery
  2. User can switch between images in the current result set using bottom filmstrip strip
  3. User can zoom, drag, and double-click-to-zoom within viewer window
  4. User can view image metadata (filename, resolution, size, tags) in viewer sidebar
**Plans**: 3 plans
Plans:
- [x] `18-01-PLAN.md` — Establish Windows secondary-window bootstrap, viewer session payloads, and launch coordinator boundaries
- [x] `18-02-PLAN.md` — Build the reusable viewer workspace with stage, metadata sidebar, filmstrip, and keyboard contract
- [x] `18-03-PLAN.md` — Wire gallery/search double-click launches into real viewer windows and close Phase 18 verification
**UI hint**: yes

### Phase 19: Tag Management
**Goal**: Admins can manage tags directly from desktop app without switching to legacy interfaces
**Depends on**: Phase 17 (desktop shell foundation)
**Requirements**: DSK-04
**Success Criteria** (what must be TRUE):
  1. Admin can view all tags with usage statistics in dedicated tag management view
  2. Admin can edit tag names and merge duplicate tags in-place
  3. Admin can delete unused tags with confirmation showing affected image count
**Plans**: 3 plans
Plans:
- [ ] `19-01-PLAN.md` — Build backend governance contracts for enriched list rows, explicit merge, and safe delete/cleanup semantics
- [ ] `19-02-PLAN.md` — Build Flutter governance models, service methods, provider selection state, and batch orchestration
- [ ] `19-03-PLAN.md` — Build the Fluent desktop governance workspace, merge panel, batch action bar, and gallery drilldown
**UI hint**: yes

### Phase 20: Operations Monitoring
**Goal**: Admins can monitor import tasks and diagnose Python sidecar health from desktop UI
**Depends on**: Phase 18 (viewer complete), Phase 16 (sidecar integration)
**Requirements**: OPS-01, OPS-02
**Success Criteria** (what must be TRUE):
  1. Admin can view batch and task status from import task monitoring entry in desktop navigation
  2. Admin can see Python sidecar running status, recent error summary, and manual restart option
  3. Admin can diagnose sidecar issues without checking external logs
**Plans**: TBD
**UI hint**: yes

### Phase 21: Windows Packaging
**Goal**: Users can install and run the application without installing Python runtime separately
**Depends on**: Phase 20 (all features verified)
**Requirements**: OPS-03
**Success Criteria** (what must be TRUE):
  1. User can download single package containing Flutter + Go + Python and run without installing Python
  2. Application starts correctly after extraction with proper Flutter → Go → Python startup sequence
  3. User on machine without Python environment can use all features including duplicate detection
**Plans**: TBD
**UI hint**: no

### Phase 22: Large Gallery Performance
**Goal**: Browsing remains responsive even with tens of thousands of images in gallery
**Depends on**: Phase 21 (packaging complete)
**Requirements**: PERF-01, PERF-02, PERF-03, COMP-05
**Success Criteria** (what must be TRUE):
  1. User can browse grid view smoothly without noticeable lag when gallery contains 10k+ images
  2. User sees filter results within acceptable response time when applying tag filters on large gallery
  3. User can continue browsing gallery while duplicate detection runs in background without UI freezing
  4. System shows clear failure message with recovery hint when sidecar becomes unavailable
**Plans**: TBD
**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 15 → 16 → 17 → 18 → 19 → 20 → 21 → 22

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 15. Compute Sidecar Infrastructure | v4.0 | 3/3 | Complete    | 2026-04-03 |
| 16. Duplicate Detection Migration | v4.0 | 3/3 | Complete    | 2026-04-04 |
| 17. Desktop Shell Foundation | v4.0 | 3/3 | Complete    | 2026-04-05 |
| 18. Independent Viewer & Filmstrip | v4.0 | 3/3 | Complete    | 2026-04-05 |
| 19. Tag Management | v4.0 | 0/3 | Planned | - |
| 20. Operations Monitoring | v4.0 | 0/TBD | Not started | - |
| 21. Windows Packaging | v4.0 | 0/TBD | Not started | - |
| 22. Large Gallery Performance | v4.0 | 0/TBD | Not started | - |

---
*Roadmap created: 2026-04-03*
*Current milestone: v4.0 Windows Photos 风格重构与计算层拆分*
