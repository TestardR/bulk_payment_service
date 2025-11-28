# Interview Preparation - Bulk Transfer Service
**Staff Engineer Position - Technical Assessment**

---

## 1. Architecture Presentation & Screen Share

### High-Level Overview
*"I built a bulk transfer payment service using **hexagonal architecture** (ports & adapters pattern) in Go. The service processes bulk money transfers atomically - ensuring balance updates and transaction records are committed together or not at all."*

### Architecture Diagram Walkthrough

**Show: `docs/c4-component-architecture.png`**

#### Three Main Layers:

1. **HTTP Layer (Primary Adapter)** - `internal/http/`
   - Entry point for external requests
   - Validates incoming JSON requests (required fields, EUR currency, positive amounts)
   - Converts HTTP DTOs to domain models
   - Maps domain errors to HTTP status codes (404, 422, 500)
   - Uses `go-playground/validator` for declarative validation

2. **Core Layer (Domain + Application)** - `internal/core/`
   - **Domain Models** (`models.go`): `Account`, `Transfer`, `BulkTransfer`
   - **Business Logic**: Account debit operations, funds validation
   - **Service** (`service.go`): Orchestrates the bulk transfer operation
   - **Repository Port** (`repository.go`): Interface defining persistence contract
   - **Pure business logic** - no HTTP/DB concerns

3. **Infrastructure Layer (Secondary Adapter)** - `internal/sqlite/`
   - Concrete implementation of `AccountRepository`
   - SQLite with WAL mode for concurrent reads during writes
   - Transaction management with `BEGIN IMMEDIATE`
   - Bulk insert optimization

### Data Flow - Successful Transfer

**Show: `docs/sequence.png`**

Walk through the sequence:
1. Client sends POST to `/transfers/bulk` with organization IBAN/BIC + transfers array
2. HTTP handler validates request (struct tags + validator)
3. Converts DTO to domain model (string amounts → int64 cents)
4. Service calls `Atomic()` transaction wrapper
5. **`BEGIN IMMEDIATE`** - acquires write lock immediately
6. Fetch account by IBAN/BIC
7. Validate: `account.HasSufficientFunds(total)`
8. Debit account: `account.Debit(total)`
9. Update balance in DB
10. Bulk insert transfers (single multi-row insert)
11. **`COMMIT`** - atomic operation complete
12. Return 201 Created

**Key Insight**: *"The hexagonal architecture ensures domain logic is completely isolated. I can unit test the service with mocks, integration test the repository with a real database, and E2E test the full stack - all independently."*

---

## 2. Technical Choices - Why These and Not Others?

### Architecture Pattern: Hexagonal (Ports & Adapters)

**Why Hexagonal?**
- **Testability**: Domain logic testable with mocks (unit tests), repository testable with real DB (integration tests)
- **Flexibility**: Can swap SQLite → PostgreSQL without touching business logic
- **Domain Isolation**: Business rules remain pure, independent of infrastructure
- **Critical for Finance**: Financial systems require rigorous testing at each layer

**Why Not Simpler?**
- Could have put everything in HTTP handlers (common in quick prototypes)
- Trade-off: More boilerplate (interfaces, DTOs) for significant gains in testability and maintainability
- For a staff engineer position, demonstrating architectural discipline is essential

### Database: SQLite with BEGIN IMMEDIATE + WAL Mode

**Why SQLite?**
- **Simplicity**: No external DB server needed, embedded in application
- **ACID Guarantees**: Full transaction support with SERIALIZABLE isolation
- **Sufficient for Assessment**: Demonstrates transaction handling concepts

**Why BEGIN IMMEDIATE?**
```go
dsn += "&_txlock=immediate"  // in config
```
- Acquires **RESERVED lock at transaction start** (not at first write)
- Prevents "check-then-act" race conditions
- Concurrent writes are serialized (safe)
- Concurrent reads allowed during writes (WAL mode)

**Alternative: BEGIN DEFERRED (default)**
- Problem: Lock acquired only on first write
- Race condition window: Multiple transactions can read stale balance, then all try to write
- Result: One succeeds, others retry or fail

**Alternative: BEGIN EXCLUSIVE**
- Too aggressive: Blocks ALL reads during transaction
- Unnecessary: WAL mode allows concurrent reads

**Why WAL (Write-Ahead Logging)?**
```go
dsn += "&_journal_mode=WAL"
```
- Allows concurrent readers while writer has transaction open
- Better throughput for read-heavy workloads
- Industry best practice for SQLite

**What I'd Do in Production: PostgreSQL**
- **Row-level locking** (not database-level like SQLite)
- Horizontal scaling: Multiple app instances can write to different accounts concurrently
- Better tooling: PgBouncer for connection pooling

### Money Representation: Integer Arithmetic (Cents)

**Why Integer Cents?**
```go
amountCents := int64(floatAmount * 100)  // €10.50 → 1050 cents
```
- **Avoids floating-point precision errors** (0.1 + 0.2 ≠ 0.3 in binary)
- All arithmetic operations are exact
- Standard practice in financial systems

Problem: Binary Floating-Point Representation
- Computers store decimal numbers in binary floating-point format.Decimal fractions like 0.1 have infinite binary representations, similar to how 1/3 = 0.333... in decimal. Computers must round these, creating tiny errors that accumulate.

Solution: I use integer arithmetic for all money calculations. The API accepts strings like "100.50", which I parse to 10050 integer cents. This avoids the classic floating-point problem where 0.1 + 0.2 doesn't equal 0.3 due to binary representation limitations. All arithmetic is exact - no rounding errors, no precision loss. This is the standard approach in financial systems.

**API vs. Storage Conversion:**
- API accepts: `"amount": "100.50"` (string)
- Parsed to: `10050` (int64 cents)
- Stored as: `-10050` (negative for debits, per accounting convention)

**Alternative Considered: Decimal Library**
- `shopspring/decimal` for exact decimal arithmetic
- Trade-off: Added dependency vs. simplicity
- Judgment: String→float→int is sufficient for 2 decimal places (cents)


### Validation Strategy: Layered Validation

**HTTP Layer Validation:**
```go
type CreditTransfer struct {
    Amount   string `json:"amount" validate:"required,gt=0"`
    Currency string `json:"currency" validate:"required,eq=EUR"`
    // ...
}
```
- Format validation (required fields, positive amounts, EUR only)
- Catches malformed requests early

**Domain Layer Validation:**
```go
func (a *Account) HasSufficientFunds(total int64) bool {
    return a.BalanceCents >= total
}
```
- Business rule validation (sufficient funds)
- Domain-specific logic

**Why Separate?**
- **Clear boundaries**: HTTP concerns vs. business concerns
- HTTP layer protects against bad input, domain layer enforces business rules

### Dependency Choices

**Key Libraries:**
- `go-playground/validator`: Declarative struct validation (industry standard)
- `mattn/go-sqlite3`: Official CGO SQLite driver (battle-tested)
- `kelseyhightower/envconfig`: Environment-based config (12-factor app)
- `stretchr/testify`: Assertion library (ergonomic tests)
- `go.uber.org/mock`: Mock generation (type-safe mocks)

**Why These?**
- Battle-tested, widely adopted in Go community
- Zero "magic" - explicit and predictable
- Minimal dependencies (lean binary)

---

## 3. Development Strategy

### Approach: Outside-In with TDD Principles

**Development Sequence (as seen in git commits):**

```
1. feat: initialize boilerplate
   ↓
2. feat(core): define domain models with repository
   ↓
3. feat(core): add service layer to orchestrate business logic
   ↓
4. feat(infrastructure): add account store with sqlite + integration tests
   ↓
5. feat(infrastructure): add http server and handler with unit tests
   ↓
6. feat(e2e): add e2e test to cover bulk transfers
   ↓
7. feat(infrastructure): review http input validation
   ↓
8. feat(documentation): add README.md
```

**Strategy Explained:**

1. **Domain First** (commits 2-3)
   - Started with domain models (`Account`, `Transfer`, `BulkTransfer`)
   - Defined repository interface (port)
   - Implemented service orchestration
   - **Why?** Domain is the heart - get business logic right first

2. **Infrastructure Second** (commit 4)
   - Implemented SQLite repository (adapter)
   - Added integration tests with real database
   - **Why?** Verify persistence layer works before connecting HTTP

3. **HTTP Layer Third** (commits 5, 7)
   - HTTP handlers + DTOs
   - Unit tests with mocked service
   - **Why?** API layer is the "glue" - depends on stable foundation

4. **E2E Tests** (commit 6)
   - Full stack test: HTTP → Service → SQLite
   - **Why?** Validates all components work together

5. **Documentation** (commit 8)
   - Comprehensive README with architecture diagrams
   - **Why?** Makes the work reviewable and presentable

### Test Strategy: Three-Tier Pyramid

**Unit Tests** (`internal/*_test.go`)
- Domain logic with mocks
- Service layer tests: Mock repository, verify business logic
- HTTP layer tests: Mock service, verify request/response handling
- **Fast, isolated, high coverage**

**Integration Tests** (`test/components/sqlite/*_test.go`)
- Repository implementation with real SQLite database
- Tests: GetAccountByID, UpdateBalance, AddTransfers, Atomic transactions
- Concurrent write test to verify serialization
- **Verify infrastructure layer works correctly**

**E2E Tests** (`test/http/*_test.go`)
- Full stack: HTTP request → Service → SQLite → Response
- Real database seeded with test data
- **Validates entire system end-to-end**

**Coverage:**
- 8 test files for 15 production files (53% test code)
- Unit tests: Fast feedback on business logic
- Integration tests: Catch DB issues (query syntax, transaction handling)
- E2E tests: Catch integration issues (DTO mapping, error propagation)

### Why This Order?

**Domain-Driven Design (DDD) Influence:**
- Start with business logic (domain models)
- Add application orchestration (service)
- Plug in infrastructure (repository, HTTP)
- **Benefit:** Core logic stabilizes early, infrastructure adapts to it

**Iterative Refinement:**
- Commit 7 shows validation improvements after initial implementation
- Real-world: Build → Test → Refine

---

## 4. Functionality & Layer Preferences

### Repository Pattern: Single Repository

**Design:**
```go
type AccountRepository interface {
    GetAccountByID(ctx, iban, bic) (Account, error)
    UpdateBalance(ctx, account) error
    AddTransfers(ctx, transfers) error
    Atomic(ctx, callback) error
}
```

**Why Single Repository?**
- Bulk transfer operation needs **atomic multi-operation transaction**
- Single repository simplifies transaction management
- All operations happen on one account + its transfers

**Alternative Considered: Separate Repositories**
- `AccountRepository` + `TransferRepository`
- Problem: How to share transaction across repositories?
- Solutions:
  1. Pass `*sql.Tx` around (leaks DB concerns to domain)
  2. Unit of Work pattern (more boilerplate)
- **Judgment:** YAGNI - single repository is simpler for this use case

### Transaction Management: Callback Pattern

**Design:**
```go
func (s Service) ProcessBulkTransfer(ctx, bulkTransfer) error {
    callback := func(r AccountRepository) error {
        account, _ := r.GetAccountByID(...)
        account.Debit(total)
        r.UpdateBalance(account)
        r.AddTransfers(transfers)
        return nil
    }
    return s.repository.Atomic(ctx, callback)
}
```

**Why Callback Pattern?**
- **Encapsulates transaction boundaries** in infrastructure layer
- Domain layer doesn't know about `sql.Tx`
- Clean separation: Business logic in callback, transaction management in repository

**Alternative: Pass Transaction Explicitly**
```go
tx, _ := db.Begin()
account, _ := repo.GetAccountByID(tx, ...)
// Problem: Domain depends on sql.Tx
```

**Why Callback is Better:**
- Domain stays pure (no `sql.Tx` imports)
- Repository handles BEGIN/COMMIT/ROLLBACK
- Easier to test: Mock `Atomic()` and verify callback behavior

### Error Handling: Domain Errors

**Domain Errors:**
```go
var (
    ErrInsufficientFunds = errors.New("insufficient funds")
    ErrAccountNotFound   = errors.New("account not found")
)
```

**HTTP Mapping:**
```go
if errors.Is(err, core.ErrAccountNotFound) {
    http.Error(w, "Account not found", http.StatusNotFound)  // 404
}
if errors.Is(err, core.ErrInsufficientFunds) {
    http.Error(w, "Insufficient funds", http.StatusUnprocessableEntity)  // 422
}
```

**Why This Pattern?**
- **Domain defines business errors**
- HTTP layer maps to status codes
- Clear separation: Business meaning vs. HTTP protocol

### Configuration: Environment Variables

**Design:**
```go
type Config struct {
    LogLevel int `envconfig:"LOG_LEVEL" default:"-4"`
    Database sqlite.Config
    HTTP     http.Config
}
```

**Why Environment Variables?**
- **12-factor app principle**: Config should be environment-based
- Easy to override in different environments (dev, staging, prod)
- No secrets in code or version control

**Defaults Provided:**
- Sensible defaults for local development
- Override with `DATABASE_PATH=/data/qonto.db ./svc`

---

## 5. Commit Strategy

### Commit Philosophy: Atomic, Incremental, Semantic

**Observed Pattern:**
```
feat(infrastructure): renew DB before tech assessment submission
feat(documentation): add README.md to explain product and technical decisions
feat(infrastructure): review http input validation
feat(e2e): add e2e test to cover bulk transfers
feat(infrastructure): add http server and handler with unit tests
feat(infrastructure): add account store with sqlite + integration tests
feat(core): add service layer to orchestrate business logic
feat(core): define domain models with repository
feat: initialize boilerplate
```

**Commit Strategy Analysis:**

1. **Semantic Commit Messages** (Conventional Commits style)
   - Format: `feat(scope): description`
   - `feat` = feature, `(scope)` = component (core, infrastructure, e2e, documentation)
   - Clear categorization: Core vs. Infrastructure vs. Tests vs. Docs

2. **Atomic Commits**
   - Each commit is a self-contained unit of work
   - Example: "add account store with sqlite + integration tests"
   - Complete feature: Implementation + Tests together

3. **Incremental Development**
   - Bottom-up: Domain → Infrastructure → HTTP → E2E → Docs
   - Each layer builds on previous (stable foundation)

4. **Reviewable Units**
   - Each commit is small enough to review independently
   - Clear progression: Easy to understand development flow

**Why This Strategy?**

**Benefits:**
- **Easy to review**: Each commit tells a story
- **Easy to revert**: If something breaks, rollback specific commit
- **Clear history**: Future developers understand "why" and "when"

**Trade-offs:**
- More commits (9 commits for small project)
- Alternative: "Squash everything into one commit" (loses development history)
- **Judgment:** For technical assessment, showing thought process is valuable

**What I'd Do in Production:**

- **Feature branches** with Pull Requests
- Commits during development can be messy
- **Squash merge** to main with semantic message
- Keep PR history for context, clean main branch history

Example:
```
PR #123: Implement bulk transfer endpoint
  - 12 commits (detailed development)
  - Squash to: feat: add bulk transfer endpoint with validation
```

**Team Collaboration:**
- Commit messages help during code review ("What changed and why?")
- CI/CD: Trigger tests/deploys based on commit scopes
- Changelog generation: Extract features/fixes from commit messages

---

## 6. Future-Proofing: Tested, Scalable, Robust

### Testing: Comprehensive Three-Tier Strategy

**Coverage Metrics:**
- **8 test files** for 15 production files
- Unit tests, integration tests, E2E tests
- Tests exercise:
  - Happy paths (successful transfers)
  - Error paths (insufficient funds, account not found)
  - Edge cases (empty transfer list, concurrent writes, bulk insert boundaries)

**Test Independence:**
```go
t.Parallel()  // in all tests
```
- All tests run in parallel (faster feedback)
- Each test has isolated database (`:memory:` or temp file)
- No shared state between tests

**Mock Generation:**
```go
//go:generate go tool go.uber.org/mock/mockgen -source=repository.go ...
```
- Type-safe mocks (compile-time verification)
- Easy to regenerate when interfaces change

**What Makes This Future-Proof?**
- **Regression safety**: Changes that break existing behavior are caught immediately
- **Refactoring confidence**: Tests verify behavior, not implementation
- **Documentation**: Tests show how components should be used

### Scalability: Designed for Evolution

**Current Limitations (Acknowledged in README):**

| Aspect | Current | Production Needed |
|--------|---------|-------------------|
| Database | SQLite (single writer) | PostgreSQL (row-level locks) |
| Concurrency | 1 app instance | Horizontal scaling (multiple instances) |
| Idempotency | ❌ | ✅ Required (idempotency keys) |

**Why These Limitations Are Okay:**
- **Scope-appropriate**: Technical assessment, not production system
- **Documented**: README section "What I'd Do Differently in Production"
- **Demonstrates awareness**: Shows I understand trade-offs

**Evolution Path to Production:**

1. **Swap SQLite → PostgreSQL**
   - Change: `internal/sqlite/` → `internal/postgres/`
   - Core logic unchanged (hexagonal architecture benefit)
   ```go
   // main.go
   // accountRepository := sqlite.NewAccountStore(dbClient.DB())
   accountRepository := postgres.NewAccountStore(dbClient.DB())
   ```

2. **Add Idempotency**
   - API change: Add `idempotency_key` field to request
   - Repository: Check if key exists before processing
   - Domain logic unchanged

3. **Horizontal Scaling**
   - PostgreSQL enables multiple app instances
   - Each instance can write to different accounts (row-level locks)
   - No code changes needed (already designed for concurrency)

4. **Add Observability**
   - Inject metrics/tracing into service
   - Domain logic unchanged (dependency injection)

### Robustness: Handling Failures

**Database Failures:**
```go
if err := s.repository.Atomic(ctx, callback); err != nil {
    return err  // Propagates to HTTP layer
}
```
- Transaction rollback automatic (defer rollback in Atomic)
- Database errors return 500 (internal server error)

**Concurrent Writes:**
- Tested explicitly: `TestAccountStore_Atomic_ConcurrentWrites`
- 5 concurrent transactions, all succeed, balance is correct
- Demonstrates SQLite serialization works

**Input Validation:**
- HTTP layer: Catches malformed requests (400 Bad Request)
- Domain layer: Catches business rule violations (422 Unprocessable Entity)
- Defense in depth: Multiple validation layers

**Graceful Shutdown:**
```go
// main.go
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
<-stop
httpServer.Stop(ctx)  // Graceful shutdown
dbClient.Close()      // Close DB connections
```
- SIGINT/SIGTERM handlers
- 30-second timeout for in-flight requests

### Changeability: Low Coupling, High Cohesion

**Dependency Inversion:**
```go
type Service struct {
    accountRepository AccountRepository  // Interface, not concrete type
}
```
- Core depends on **interface** (port)
- Infrastructure implements interface (adapter)
- Can swap implementations without changing core

**DTO Conversion at Boundaries:**
```go
func (req BulkTransferRequest) ToDomain() (core.BulkTransfer, error)
```
- HTTP DTOs ≠ Domain models
- Protects domain from API changes
- If API format changes, only HTTP layer changes

**Configuration Flexibility:**
```go
envconfig.Process("", &config)
```
- All config via environment variables
- Easy to change behavior without code changes

**What Makes This Robust to Change?**

1. **Clear Boundaries**
   - HTTP layer can be replaced with gRPC without touching core
   - SQLite can be replaced with PostgreSQL without touching core
   - Core can evolve independently

2. **Interface-Based Design**
   - Mock repositories for testing
   - Swap implementations at runtime

3. **No Premature Optimization**
   - Code is simple and readable
   - Easy to optimize later when bottlenecks are identified

---

## Key Talking Points Summary

### Architecture
- **Hexagonal architecture** for testability and flexibility
- **Three layers**: HTTP (adapter), Core (domain+application), Infrastructure (adapter)
- **Dependency inversion**: Core depends on interfaces, not concrete implementations

### Technical Choices
- **BEGIN IMMEDIATE**: Prevents race conditions, enables concurrent reads (WAL)
- **Integer arithmetic**: Avoids floating-point errors in financial calculations
- **Bulk inserts**: Single multi-row INSERT for all transfers
- **Layered validation**: HTTP (format) + Domain (business rules)

### Development Strategy
- **Domain-first**: Models → Service → Infrastructure → HTTP → E2E
- **Three-tier testing**: Unit (mocks) → Integration (real DB) → E2E (full stack)
- **Iterative refinement**: Build → Test → Improve

### Commit Strategy
- **Semantic commits**: `feat(scope): description`
- **Atomic units**: Each commit is complete and reviewable
- **Incremental development**: Clear progression from domain to docs

### Future-Proofing
- **Tested**: Unit, integration, E2E tests with parallel execution
- **Scalable**: Clear evolution path to PostgreSQL and horizontal scaling
- **Robust**: Error handling, graceful shutdown, concurrent write safety
- **Changeable**: Low coupling via interfaces, clear boundaries

---

## Anticipated Interview Questions

### Q: Why hexagonal architecture for a small service?
**A:** *"Financial systems require rigorous testing. Hexagonal architecture enables independent testing of each layer (unit tests with mocks, integration tests with real DB, E2E tests). The upfront cost of interfaces pays off in testability and flexibility. For a staff engineer, demonstrating architectural discipline is important."*

### Q: Why SQLite instead of PostgreSQL?
**A:** *"For a technical assessment, SQLite demonstrates transaction handling concepts without requiring external infrastructure. It provides full ACID guarantees. However, I documented in the README that production would need PostgreSQL for row-level locking and horizontal scaling."*

### Q: How do you handle race conditions?
**A:** *"SQLite's BEGIN IMMEDIATE acquires a write lock at transaction start, serializing concurrent writes. I have an integration test (`TestAccountStore_Atomic_ConcurrentWrites`) that verifies 5 concurrent transfers all succeed with correct final balance. WAL mode allows concurrent reads during writes."*

### Q: How would you add idempotency?
**A:** *"1. Add `idempotency_key` field to API request. 2. Store key + outcome in DB. 3. In repository, check if key exists: if yes, return cached outcome; if no, process and store. 4. Domain logic unchanged - this is purely infrastructure concern. Hexagonal architecture makes this straightforward to add."*

### Q: How do you test this?
**A:** *"Three-tier strategy: 1. Unit tests mock repository, verify service business logic. 2. Integration tests use real SQLite, verify repository implementation. 3. E2E tests exercise full stack. All tests run in parallel with isolated state. This catches issues at the appropriate level: business logic bugs (unit), SQL errors (integration), DTO mapping issues (E2E)."*

### Q: What would you improve?
**A:** *"Production needs: 1. PostgreSQL for scalability. 2. Idempotency keys for reliability. 3. Observability (metrics, traces, structured logs). 4. Auth/RBAC. 5. Rate limiting. But for a technical assessment, I prioritized correctness and testability. The hexagonal architecture makes these additions straightforward."*

### Q: How do you handle errors?
**A:** *"Domain errors (insufficient funds, account not found) are defined in core package. HTTP layer maps them to status codes (404, 422). Infrastructure errors (DB failures) propagate as 500. This separation keeps domain pure while allowing HTTP layer to provide appropriate user feedback."*

### Q: Why callback pattern for transactions?
**A:** *"Keeps domain layer independent of SQL transactions. Repository's `Atomic()` method handles BEGIN/COMMIT/ROLLBACK, service passes a callback with business logic. This maintains hexagonal architecture principle: domain doesn't depend on infrastructure details like `sql.Tx`."*

---

## Demo Script (Screen Share)

1. **Show architecture diagram** (`docs/c4-component-architecture.png`)
   - Walk through three layers and their responsibilities
   - Point out dependency inversion (Core → Interface ← Infrastructure)

2. **Show sequence diagram** (`docs/sequence.png`)
   - Walk through successful bulk transfer flow
   - Highlight BEGIN IMMEDIATE and transaction boundaries

3. **Show code structure** (`tree` or IDE)
   - Navigate through `internal/core/`, `internal/http/`, `internal/sqlite/`
   - Point out clear separation

4. **Show domain models** (`internal/core/models.go`)
   - Simple, pure business logic
   - No HTTP/DB concerns

5. **Show repository interface** (`internal/core/repository.go`)
   - Port definition (contract)

6. **Show SQLite implementation** (`internal/sqlite/account_store.go`)
   - Adapter implementation
   - Point out BEGIN IMMEDIATE, bulk inserts

7. **Show tests** (pick one from each tier)
   - Unit test: `internal/core/service_test.go` (with mocks)
   - Integration test: `test/components/sqlite/account_store_test.go` (real DB)
   - E2E test: `test/http/http_test.go` (full stack)

8. **Run tests live**
   ```bash
   make unit_test        # Show fast, isolated unit tests
   make integration_test # Show integration tests with real DB
   ```

9. **Run the service**
   ```bash
   make local-run        # Start service
   curl -X POST http://localhost:8080/transfers/bulk -d @docs/sample1.json
   ```

10. **Show README** (`README.md`)
    - Point out "What I'd Do Differently in Production" section
    - Demonstrates awareness of trade-offs

---

## Final Thoughts

**Key Message for Interview:**

*"I built this service to demonstrate architectural discipline and correctness for financial systems. The hexagonal architecture provides testability and flexibility. While it's overengineered for a 3-hour assessment, it shows how I'd structure a real production system - with clear separation of concerns, comprehensive testing, and a path to scale. I documented the trade-offs (SQLite vs. PostgreSQL) because for a staff engineer role, it's important to show I understand when to optimize and when not to prematurely optimize."*

**Confidence Boosters:**
- ✅ Clean, idiomatic Go code
- ✅ Comprehensive tests (unit, integration, E2E)
- ✅ Clear architectural separation
- ✅ Documented trade-offs and production evolution path
- ✅ Semantic commit history shows thought process
- ✅ Handles concurrency correctly (tested)
- ✅ Production-quality error handling and graceful shutdown

**Be Ready to Discuss:**
- Alternative architectures (why NOT just HTTP handlers?)
- Database transaction isolation levels in detail
- How you'd add specific production features (idempotency, auth, rate limiting)
- Trade-offs you made and why
- How you'd test this in production (integration tests in CI, staging environment)

---

Good luck with your interview! 🚀
