# Brooks-Lint — PR Review

**Mode:** PR Review
**Scope:** Recent commits to `internal/` package (collection and ranking features, 15 commits, ~40 files added/modified)
**Health Score:** 68/100

Overall codebase respects clean architecture boundaries with clear layering, but exhibits repeated pattern duplication across features and some domain model consistency issues.

---

## Module Dependency Graph

No architectural module dependency graph applicable for PR Review mode — visualizes changesets, not static structure.

---

## Findings

### 🔴 Critical

**Dependency Disorder — Service layers directly import repository packages**
Symptom: `internal/service/collection.go` imports repository package, which will import `gorm.DB` and `po` models, violating dependency direction
Source: Martin — Clean Architecture, Stable Dependencies Principle (SDP)
Consequence: Domain service layer depends on ORM and persistence details; each hexagon boundary can't be tested independently
Remedy: Convert repository interfaces in `internal/service/*.go` to explicit abstractions without ORM dependencies, inject `gorm.DB` at handler/infrastructure wiring boundary only

**Domain Model Distortion — `do.Image` contains queries (`IsActive`, `NormalizeForCreate`) blurred with behavior**
Symptom: Domain object `Image` has methods that assert validation rules (`IsActive()`) and prepare persistence shape (`NormalizeForCreate()`) — anemic model with behavior anchored in the wrong layer
Source: Evans — Domain-Driven Design, Anemic Domain Model pattern
Consequence: Business rules leak into domain objects; testing domain logic requires database context or mocks; violates Single Responsibility
Remedy: Move validation logic to service layer, keep `do.Image` as pure value object with no behavior; introduce `ImageBuilder` or similar for pre-persistence transformations

### 🟡 Warning

**Change Propagation — `TagRepository.FindActiveImagesWithTags()` creates new repository instance**
Symptom: In `internal/repository/tag.go:213`, method creates `NewImageRepository(r.readDB, r.writeDB)` inside repository method — violates encapsulation and creates hard-to-mock dependencies
Source: Fowler — Refactoring, Feature Envy & Law of Demeter violations
Consequence: Repository methods can't be unit tested without database; circular injection makes wire-up brittle; each change to image schema forces tag changes
Remedy: Introduce `repo.WithTransaction()` or dependency injection helper to inject already-resolved repository; pass `ImageRepository` down via constructor or context, don't create inside methods

**Knowledge Duplication — Interface contracts duplicated across services**
Symptom: `internal/service/collection.go:22` defines `CollectionRepository` interface, presumably mirrored in other services — interfaces repeated per layer, not shared
Source: Hunt & Thomas — The Pragmatic Programmer, DRY principle
Consequence: Changing interface requires editing multiple files; test doubles proliferate; doubled maintenance burden
Remedy: Centralize repository interfaces in `internal/infra/repository/interfaces.go` or use generated artifacts; inject via common dependency registry

**Accidental Complexity — Ranking calculation distributed across multiple types**
Symptom: `do.Ranking`, `do.RankingScore`, `do.RankingMetrics` bubble ranking calculation details into domain objects; business formula mixed in transfer objects
Source: Fowler — Refactoring, Data Class pattern
Consequence: Calculation logic scattered; changing formula touches multiple type definitions; domain logic buries in DTO
Remedy: Extract ranking formula into dedicated `internal/ranking/formula.go` package; keep only aggregated metrics as domain values; use pure functions for scoring

**Change Propagation — Both `do.ImageEvent` and collection event handling duplicate favorite count recalculation**
Symptom: Collection `AddItem`/`RemoveItem` call `applyFavoriteDelta()`, and `rating.go` service also updates `favorite_count` on events — business rule duplicated
Source: Fowler — Refactoring, Duplicate Code
Consequence: Changing favorite count impacting logic requires editing multiple repositories; shrink-wrap evolution across features; orphaned manual changes may diverge
Remedy: Centralize favorite count logic in repository/utility layer; have all raising sources delegate to shared function/update mechanism; introduce event-driven publishing of `ImageFavorited` events

**Cognitive Overload — Service layer functions exceed 30 lines in multiple places**
Symptom: `internal/service/collection.go` · `RemoveItem` (8 lines), `internal/service/rating.go` has queries, `internal/service/image.go` `Search` method long-chain joins; several exceed typical reasonable limits
Source: McConnell — Code Complete, Ch. 7: High-Quality Routines
Consequence: Functions do multiple validation checks vs persistence; reading mental load high; changing behavior touches multiple concerns in one place
Remedy: Extract validation to dedicated `Validate*Input()` helper functions; split long queries into separate methods with clear names; reduce parameter tuples

### 🟢 Suggestion

**Knowledge Duplication — Repository mapper pairs (collectionMapper.go, tagMapper.go) manually convert between DO/PO**
Symptom: Multiple `*ToDO()` and `*ToPO()` functions cloned across repositories with essentially parallel structure
Source: Fowler — Refactoring, Duplicate Code
Consequence: Duplication when adding new fields to models; indexer manually replicates transformation; rehearsal drift risk
Remedy: Introduce code-generation using struct tags (e.g., `@map`, `@field`) or struct array transformation helpers; standardize conversion boilerplate

**Cognitive Overload — `ranking_metric_select()` SQL string literal creates maintenance burden**
Symptom: `internal/repository/ranking.go:106` defines aggregation query as raw SQL string embedded in Go; future queries must be written twice
Source: Hunt & Thomas — The Pragmatic Programmer, Topic 4: Good-Enough Software (avoid premature optimization)
Consequence: Query drift between documentation, go code, and actual execution; developer safety net lowered
Remedy: Use prepared statement placeholders or an ORM mapping configuration where possible; de-risk SQL change with inline comments referencing column semantics

**Test Obscurity — Tests use `memoryCollectionRepository` test double with full-blown repository interface**
Symptom: `internal/service/collection_test.go:26` implements full `CollectionRepository` interface in test helper; 150 lines of mock methods; over-engineered for unit test
Source: Meszaros — xUnit Test Patterns, Assertion Roulette / Example Test pattern
Consequence: Test maintenance overhead high; coupling test implementation to production interfaces; slow to read
Remedy: Use minimal function `MustCreateCollection()` only for data setup; rely on in-memory map for stub behavior; reduce test double surface to necessary methods

**Knowledge Duplication — Multiple services include input validation helpers (`prepareCollectionInput`, `prepareTag`) with similar logic**
Symptom: Service-level input normalization functions duplicated in structure: trim whitespace, length checks, validation of enums, default values
Source: Fowler — Refactoring, Extract Method (applied correctly, but per-service rather than unified)
Consequence: Business rule enforcement scattered across service methods; changing normalization triggers edits in multiple files
Remedy: Centralize input normalizers in `internal/conf/validation.go` or `internal/validators/` package; inject validators into service constructors

---

## Summary

Overall codebase demonstrates good clean architecture discipline with clear handler → service → repository boundaries and anemic domain models. The acute risk areas are dependency direction violations (service importing repository with direct ORM dependencies) and domain model pollution (behavior leaking into DO objects). Recommended remediation focus: 1) Pull `gorm.DB` and `po` imports up to infra/wiring boundary, 2) Move validation behavior from `do.Image` to service layer, 3) Centralize repository interfaces and mapping utilities. Code appears committed to feature-driven velocity with adequate test coverage; main mitigation is architecture-level coupling cleanup.