# Interview Cheat Sheet - Quick Reference

## 🎯 One-Liner Pitch
*"I built a bulk transfer payment service using hexagonal architecture in Go, ensuring atomic balance updates and transaction records through SQLite's ACID guarantees with comprehensive three-tier testing."*

---

## 🏗️ Architecture (Show Diagram)

**Three Layers:**
1. **HTTP Layer** (`internal/http/`) → Validates input, maps errors
2. **Core Layer** (`internal/core/`) → Domain models + business logic
3. **Infrastructure** (`internal/sqlite/`) → Repository implementation

**Key Principle:** Domain is isolated from infrastructure via interfaces (ports & adapters)

---

## 🔑 Key Technical Decisions

| Decision | Why | Trade-off |
|----------|-----|-----------|
| **Hexagonal Architecture** | Testability + flexibility | More boilerplate vs. testability gains |
| **BEGIN IMMEDIATE** | Prevent race conditions | Serializes writes (fine for single instance) |
| **Integer Cents** | Avoid floating-point errors | Standard in finance |
| **Bulk Inserts** | Efficient multi-row INSERT | Memory vs. round trips |
| **SQLite (given in test)** | Required by assessment | Prod needs Postgres for scaling |

---

## 📊 Data Flow (8 Steps)

1. HTTP validates request
2. Convert DTO → Domain
3. **BEGIN IMMEDIATE** (acquire write lock)
4. Fetch account
5. Validate: `HasSufficientFunds()`
6. Debit account
7. Bulk insert transfers
8. **COMMIT** (atomic)

---

## 🧪 Testing Strategy

**Three Tiers:**
- **Unit Tests**: Service with mocked repository (business logic)
- **Integration Tests**: Repository with real SQLite (SQL correctness)
- **E2E Tests**: Full stack HTTP → Service → SQLite (integration)

**Key Tests:**
- Concurrent writes (5 goroutines, correct final balance)
- Multiple transfers (1, 5 transfers)
- Error paths (insufficient funds, account not found)

---

## 💡 Development Strategy

**Order:** Domain → Infrastructure → HTTP → E2E → Docs

**Why?** Stabilize core business logic first, then add adapters

---

## 📝 Commit Strategy

**Pattern:** `feat(scope): description`

**Examples:**
- `feat(core): define domain models with repository`
- `feat(infrastructure): add account store with integration tests`

**Why?** Atomic commits, reviewable units, clear history

---

## 🚀 Future-Proofing

**Tested:**
- 8 test files / 15 prod files
- Parallel execution
- All layers covered

**Scalable:**
- Evolution path: SQLite → PostgreSQL (swap adapter, core unchanged)
- Documented production needs (idempotency, auth, rate limiting)

**Robust:**
- Error handling (domain errors → HTTP status codes)
- Graceful shutdown (SIGINT/SIGTERM)
- Concurrent write safety (tested)

**Changeable:**
- Low coupling via interfaces
- Clear boundaries (DTO ≠ Domain)

---

## 🎤 Common Questions - Quick Answers

**Q: Why hexagonal for small service?**
A: Financial systems need rigorous testing. Hexagonal enables independent testing of each layer.

**Q: Why SQLite not PostgreSQL?**
A: Assessment simplicity + demonstrates concepts. Documented that prod needs Postgres.

**Q: How handle race conditions?**
A: BEGIN IMMEDIATE serializes writes. Integration test verifies 5 concurrent transfers succeed.

**Q: How add idempotency?**
A: Add `idempotency_key` to API, store key+outcome in DB, check before processing. Domain unchanged.

**Q: How test?**
A: Unit (mocks) → Integration (real DB) → E2E (full stack). Catches issues at appropriate level.

**Q: What would improve?**
A: Prod needs: PostgreSQL, idempotency, observability, auth, rate limiting. Hexagonal makes these easy to add.

**Q: Why callback pattern for transactions?**
A: Keeps domain independent of SQL. Repository handles BEGIN/COMMIT/ROLLBACK, service provides business logic.

---

## 🎬 Demo Flow (5 min)

1. **Show architecture diagram** (30s)
2. **Show sequence diagram** (30s)
3. **Show code structure** - navigate layers (1 min)
4. **Show key code:**
   - Domain model (`models.go`)
   - Repository interface (`repository.go`)
   - SQLite implementation (`account_store.go`)
   - Service orchestration (`service.go`)
5. **Show tests** - one from each tier (1 min)
6. **Run tests** - `make unit_test` (30s)
7. **Run service** - live curl (1 min)

---

## 💪 Confidence Boosters

✅ Clean, idiomatic Go  
✅ Comprehensive test coverage  
✅ Clear architectural separation  
✅ Documented trade-offs  
✅ Semantic commit history  
✅ Handles concurrency correctly  
✅ Production-quality error handling  

---

## 🔥 Key Talking Points

**Architecture:** Hexagonal for testability, domain isolated from infrastructure

**Concurrency:** BEGIN IMMEDIATE + WAL mode = safe concurrent writes, tested with 5 goroutines

**Money:** Integer cents avoid floating-point errors (standard in finance)

**Testing:** Three-tier pyramid - unit/integration/E2E, all parallel, all isolated

**Scalability:** Evolution path documented, clear separation enables easy swaps

**Commits:** Semantic, atomic, incremental - shows thought process

**Trade-offs:** Acknowledged SQLite limitations, documented production needs

---

## 📱 Quick Facts

- **Lines of Code:** ~2000 (excluding tests)
- **Test Files:** 8
- **Prod Files:** 15
- **Test Ratio:** 53% test code
- **Commits:** 9 semantic commits
- **Dependencies:** 5 key libraries (battle-tested)
- **Transaction Isolation:** SERIALIZABLE (SQLite default)
- **Bulk Inserts:** Efficient multi-row INSERT statements
- **Money Representation:** int64 cents (not float)

---

## 🎯 Closing Statement

*"I prioritized correctness and testability over premature optimization. The hexagonal architecture demonstrates how I'd structure a real production system - with clear separation, comprehensive testing, and a documented path to scale. For a staff engineer role, showing architectural discipline and understanding of trade-offs is as important as the implementation itself."*

---

**Remember:** Be confident, show your thought process, acknowledge trade-offs, and demonstrate you understand production concerns even if not implemented in this assessment. Good luck! 🚀
