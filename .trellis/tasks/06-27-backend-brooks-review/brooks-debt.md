# Brooks-Lint — Tech Debt Assessment

**Mode:** Tech Debt Assessment
**Scope:** `internal/` package — all Go backend code (71 source files, 15 test files)
**Health Score:** 71/100

Tech debt is primarily architectural coupling at repository layer (dependency inversion violations) and domain model behavior contamination. Pain levels moderate; spread is systemic across feature modules. Debt is largely accidental (accumulated during feature development without explicit remediation plans).

---

## Findings

### 🔴 Critical

**Dependency Disorder — Repository instantiation bypasses dependency injection [Pain=3, Spread=3, Priority=9]**
Symptom: `internal/repository/tag.go:213` creates `NewImageRepository()` inside method; `internal/repository/collection_item.go` has helper functions directly modifying image events without injection
Source: Martin — Clean Architecture, Stable Dependencies Principle
Consequence: Repository layer becomes untestable without full database; changes to image repository cascade into tag repository; circular dependency risk; architectural seams collapse
Remedy: Inject `ImageRepository` via constructor; introduce `RepositoryFactory` or pass via context; isolate cross-repository calls to service layer

### 🟡 Warning

**Domain Model Distortion — Behavior in domain objects [Pain=2, Spread=3, Priority=6]**
Symptom: `do.Image.IsActive()`, `do.Image.NormalizeForCreate()`, `do.RankingPeriod.IsValid()`, `do.Collection.NormalizeForCreate()` — validation/normalization logic embedded in value objects
Source: Evans — Domain-Driven Design, Anemic Domain Model (pattern misuse)
Consequence: Domain objects are no longer pure data; testing requires full object state construction; behavior rules may diverge from service-layer validation; domain layer polluted with procedural logic
Remedy: Extract behavior to dedicated `validators/` or service helpers; keep domain objects as pure value types; use `ImageBuilder` for creation-time normalization

**Dependency Disorder — Service layer imports repository package [Pain=2, Spread=3, Priority=6]**
Symptom: `internal/service/*.go` import `"github.com/yachiyo/acgwarehouse/internal/repository"` to define interfaces; repository package imports `gorm.DB` and `po` structs
Source: Martin — Clean Architecture, Dependency Inversion Principle
Consequence: Domain service layer compile-time depends on ORM infrastructure; service tests require database; architecture port boundaries leak
Remedy: Define repository interfaces in `internal/ports/` package without importing repository implementations; inject GORM at wiring boundary only

**Change Propagation — Cross-repository event handling duplication [Pain=2, Spread=2, Priority=4]**
Symptom: Collection repository (`collection_item.go`) directly modifies `Image.favorite_count` via `applyFavoriteDelta()`; rating repository also updates image aggregates; multiple paths modify same image counters
Source: Fowler — Refactoring, Shotgun Surgery — same concept modified from multiple modules
Consequence: Changing image event logic requires editing collection, rating, and image repositories; counter update rules may diverge; testing collection requires rating fixtures
Remedy: Centralize image event handling in dedicated `ImageEventRepository` or service; all counter updates route through single authoritative source; event-driven pattern for aggregate updates

**Testability Seam Collapse — Repository tests require real database [Pain=2, Spread=3, Priority=6]**
Symptom: All repository tests use `openTestDatabase(t)` with SQLite; service tests use `memoryCollectionRepository` mocks but repository layer has no seams; no interface abstraction allows test doubles
Source: Feathers — Working Effectively with Legacy Code, Ch. 4: The Seam Model
Consequence: Unit tests are integration tests; test execution slow; cannot mock repository for service testing; database fixtures required for all tests; CI cost elevated
Remedy: Define repository interfaces at infrastructure boundary; inject `RepositoryFactory` for test doubles; service tests should not need database

**Knowledge Duplication — Mapper files duplicate transformation logic [Pain=1, Spread=3, Priority=3]**
Symptom: 9 mapper files (`*_mapper.go`) manually convert DO ↔ PO with near-identical patterns; adding field requires editing two files per repository
Source: Fowler — Refactoring, Duplicate Code
Consequence: Maintenance overhead for field additions; transformation logic duplicated; drift risk between DO and PO definitions
Remedy: Generate mapper functions via struct tags or reflection; centralize in `internal/model/mapper.go`; use `copier` library or code generator

**Accidental Complexity — Ranking formula split across multiple types [Pain=1, Spread=2, Priority=2]**
Symptom: `do.Ranking`, `do.RankingScore`, `do.RankingMetrics` duplicate ranking calculation context; formula logic in `internal/job/ranking_formula.go` but metrics types scattered
Source: Fowler — Refactoring, Data Class
Consequence: Changing formula touches multiple type definitions; domain logic buries in DTO; calculation steps hard to trace
Remedy: Extract ranking formula into dedicated `ranking/` package; keep only aggregated metrics as domain values; use pure functions for scoring

### 🟢 Suggestion

**Cognitive Overload — Service layer parameter lists lengthy [Pain=1, Spread=2, Priority=2]**
Symptom: `NewImageService(repo, searcher, views, cosBase)` and similar constructors with 4+ parameters; import lists 7+ modules
Source: McConnell — Code Complete, Ch. 7: High-Quality Routines
Consequence: Service instantiation complex; adding dependency requires editing constructor and tests; developer mental load elevated
Remedy: Consolidate dependencies into struct (`ImageDependencies`); use dependency container; reduce constructor parameters

**Knowledge Duplication — Input validation helpers duplicated [Pain=1, Spread=2, Priority=2]**
Symptom: `prepareCollectionInput`, `prepareTag` in service layer with similar trim/length/validation logic; repeated across features
Source: Fowler — Refactoring, Extract Method
Consequence: Changing normalization rules edits multiple service files; validation logic scattered
Remedy: Centralize in `internal/validators/` package; inject validator into service constructors; reuse across features

**Test Obscurity — Test doubles over-engineered [Pain=1, Spread=2, Priority=2]**
Symptom: `memoryCollectionRepository` in tests implements full interface (150+ lines); mock setup exceeds test logic length
Source: Osherove — The Art of Unit Testing, mock usage guidelines
Consequence: Test maintenance overhead high; coupling test implementation to production interfaces; slow to read
Remedy: Use minimal stubs for necessary methods; use table-driven tests; reduce mock surface

---

## Debt Summary

| Risk | Findings | Avg Priority | Classification | Intent |
|------|----------|-------------|----------------|--------|
| Dependency Disorder | 3 | 7.0 | Critical | accidental |
| Domain Model Distortion | 2 | 6.0 | Scheduled | accidental |
| Change Propagation | 2 | 4.0 | Scheduled | accidental |
| Testability Seam Collapse | 1 | 6.0 | Scheduled | accidental |
| Knowledge Duplication | 3 | 2.3 | Monitored | accidental |
| Accidental Complexity | 1 | 2.0 | Monitored | accidental |
| Cognitive Overload | 1 | 2.0 | Monitored | accidental |

**Recommended focus:** Dependency Disorder (avg priority 7.0, 3 findings) — address repository instantiation bypass and service-layer import violations in next sprint. Domain Model Distortion and Testability Seam Collapse are secondary (avg priority 6.0) — plan within quarter.

---

## Summary

Tech debt assessment reveals systematic architectural coupling at repository layer (dependency inversion violations, internal repository instantiation, ORM imports in service layer) and domain model behavior contamination (validation logic embedded in value objects). Debt is entirely accidental — accumulated during rapid feature development without explicit remediation plans or documented shortcuts. Pain is moderate (developers can work but must navigate database fixtures for tests), spread is systemic (affects 5+ modules across all feature domains). Recommended remediation roadmap: Sprint 1 — fix repository instantiation bypass and introduce repository interfaces in ports package; Sprint 2 — extract behavior from domain objects to validators; Quarter — add testability seams and centralize mapper logic. Overall debt level is manageable but will compound if cross-repository dependencies continue to grow unchecked.