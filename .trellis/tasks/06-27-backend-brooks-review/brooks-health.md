# Brooks-Lint — Health Dashboard

**Mode:** Health Dashboard
**Scope:** Go backend codebase (internal/ package)
**Composite Score:** 70/100

| Dimension | Score | Top Finding |
|-----------|-------|------------|
| Code Quality | 68/100 | Service layer imports repository package directly, violating dependency inversion |
| Architecture | 72/100 | Repository layer creates internal instances bypassing dependency injection |
| Tech Debt | 71/100 | Dependency disorder accumulation across repository features |
| Test Quality | 75/100 | Test coverage adequate but missing repository seam abstractions |

---

## Module Dependency Graph

```mermaid
graph TD
  subgraph Handlers
    HandlerRanking
    HandlerCollection
    HandlerRating
    HandlerTag
    HandlerImage
    HandlerUser
    Router
  end

  subgraph Services
    ServiceRanking
    ServiceCollection
    ServiceRating
    ServiceTag
    ServiceImage
    ServiceUser
    ViewBuffer
  end

  subgraph Jobs
    RankingJob
  end

  subgraph Repositories
    RepoRanking
    RepoCollection
    RepoImage
    RepoTag
    RepoRating
    RepoUser
  end

  subgraph Models
    DO[Domain Objects do]
    DTO[Data Transfer Objects dto]
    PO[Persistence Objects po]
  end

  subgraph Infrastructure
    DB[SQLite db]
    COS[COS Client]
    Search[Bleve Search]
  end

  HandlerRanking --> ServiceRanking
  HandlerCollection --> ServiceCollection
  HandlerRating --> ServiceRating
  HandlerTag --> ServiceTag
  HandlerImage --> ServiceImage
  HandlerUser --> ServiceUser
  Router --> ServiceRanking
  Router --> ServiceCollection
  Router --> ServiceRating
  Router --> ServiceTag
  Router --> ServiceImage
  Router --> ServiceUser

  ServiceRanking --> RepoRanking
  ServiceCollection --> RepoCollection
  ServiceRating --> RepoRating
  ServiceTag --> RepoTag
  ServiceImage --> RepoImage
  ServiceUser --> RepoUser
  ViewBuffer --> RepoImage

  RepoRanking --> DB
  RepoCollection --> DB
  RepoImage --> DB
  RepoTag --> DB
  RepoRating --> DB
  RepoUser --> DB
  RepoTag -.->|internal| RepoImage

  RankingJob --> RepoRanking

  ServiceImage --> DTO
  ServiceRanking --> DTO
  ServiceCollection --> DTO
  HandlerRanking --> DTO
  HandlerCollection --> DTO
  HandlerImage --> DTO

  ServiceRanking --> DO
  ServiceCollection --> DO
  ServiceImage --> DO
  RepoRanking --> DO
  RepoCollection --> DO
  RepoImage --> DO
  RepoTag --> DO
  RepoRating --> DO
  RepoUser --> DO
  RankingJob --> DO

  RepoRanking --> PO
  RepoCollection --> PO
  RepoImage --> PO
  RepoTag --> PO
  RepoRating --> PO
  RepoUser --> PO

  Search --> DO
  COS --> DO
  DB --> PO

  classDef critical fill:#ff6b6b,stroke:#c92a2a,color:#fff
  classDef warning fill:#ffd43b,stroke:#e67700
  classDef clean fill:#51cf66,stroke:#2b8a3e,color:#fff

  class RepoTag warning
  class ServiceRanking,ServiceCollection,ServiceImage,ServiceUser clean
  class HandlerRanking,HandlerCollection,HandlerRating,HandlerTag,HandlerImage,HandlerUser,Router clean
  class DO,DTO,PO clean
  class DB,COS,Search clean
  class RankingJob clean
```

---

## Top Findings (max 5 across all dimensions)

### 🔴 Critical

**Dependency Disorder — Repository layer creates other repository instances**
Symptom: `internal/repository/tag.go:213` creates `NewImageRepository()` inside method; violates dependency injection and creates hard-to-test dependencies
Source: Martin — Clean Architecture, Stable Dependencies Principle (SDP)

**Dependency Disorder — Service layer imports repository package directly**
Symptom: `internal/service/*.go` import repository package which pulls `gorm.DB` and `po` imports into domain layer
Source: Martin — Clean Architecture, Dependency Inversion Principle (DIP)

### 🟡 Warning

**Change Propagation — Cross-repository event handling duplication**
Symptom: Collection and rating repositories both directly modify `Image.favorite_count` and events
Source: Fowler — Refactoring, Shotgun Surgery

**Testability Seam Assessment — Repository interfaces inaccessible from tests**
Symptom: Repository tests use real SQLite database; no seam for test doubles
Source: Feathers — Working Effectively with Legacy Code, Ch. 4: The Seam Model

---

## Recommendation

Fix repository instantiation bypass (`TagRepository` creating `ImageRepository`) and introduce repository interface abstractions in a `ports/` package to restore dependency inversion. Both Architecture and Code Quality dimensions are degraded (72 and 68) and share same root cause. Consider running `/brooks-lint:brooks-review` for detailed PR-level findings or `/brooks-lint:brooks-audit` to refine the dependency graph remediation plan.