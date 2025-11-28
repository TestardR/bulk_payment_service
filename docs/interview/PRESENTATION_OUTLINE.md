# Presentation Outline - Screen Share Guide
**Duration: 15-20 minutes | Staff Engineer Interview**

---

## 🎯 Opening (1 min)

**Greeting & Context:**
> "Thanks for the opportunity to present my work. I built a bulk transfer payment service that processes multiple money transfers atomically. I'll walk you through the architecture, key technical decisions, and how I ensured correctness through comprehensive testing."

**What You'll Cover:**
1. Architecture overview (5 min)
2. Technical decisions (5 min)
3. Development & testing strategy (5 min)
4. Code walkthrough & demo (5 min)

---

## 📐 Part 1: Architecture Overview (5 min)

### Slide 1: High-Level Architecture
**Show:** `docs/c4-component-architecture.png`

**Script:**
> "I used hexagonal architecture - also called ports and adapters. This pattern isolates the business logic from infrastructure concerns like HTTP and databases."

**Point out three layers:**

1. **HTTP Layer** (top)
   - "Handles incoming requests, validates JSON, maps errors to HTTP status codes"
   - "If request fails validation, returns 400. If insufficient funds, returns 422."

2. **Core Layer** (middle)
   - "This is the heart - domain models and business logic"
   - "Account, Transfer, BulkTransfer with pure business logic"
   - "Service orchestrates the bulk transfer operation"
   - "Repository interface - this is the 'port' in ports and adapters"

3. **Infrastructure Layer** (bottom)
   - "SQLite implementation of the repository interface"
   - "Handles transactions, database operations"
   - "This is an 'adapter' - pluggable implementation"

**Key Point:**
> "The core depends on an interface, not concrete database. This means I can unit test the service with mocks, integration test the repository with real SQLite, and the layers are independently testable."

### Slide 2: Data Flow
**Show:** `docs/sequence.png`

**Script:**
> "Let's walk through a successful bulk transfer."

**Narrate the flow:**
1. "Client sends POST with organization IBAN/BIC and array of transfers"
2. "HTTP handler validates - required fields, EUR currency, positive amounts"
3. "Converts HTTP DTO to domain model - parsing string amounts to integer cents"
4. "Service calls repository's Atomic method with a callback"
5. **"BEGIN IMMEDIATE - this is crucial, I'll explain why in technical decisions"**
6. "Fetch account by IBAN and BIC"
7. "Validate business rule: does account have sufficient funds?"
8. "Debit the account - subtract total from balance"
9. "Update balance in database"
10. "Bulk insert transfers - single multi-row insert"
11. **"COMMIT - balance and transfers are updated atomically"**
12. "Return 201 Created"

**Key Point:**
> "If anything fails between BEGIN and COMMIT, everything rolls back. Balance and transfers are atomic - both succeed or both fail."

---

## 🔑 Part 2: Technical Decisions (5 min)

### Decision 1: Why Hexagonal Architecture?

**Script:**
> "You might ask - why hexagonal architecture for a relatively small service? Three reasons:"

1. **Testability:**
   - "Financial systems need rigorous testing. I can unit test business logic with mocks, integration test database operations with real SQLite, and E2E test the full stack. Each layer is independently testable."

2. **Flexibility:**
   - "If we need to swap SQLite for PostgreSQL, I change one line in main.go. The core business logic doesn't change."

3. **Domain Clarity:**
   - "Business logic stays pure. No HTTP or database concerns leak into domain models."

**Trade-off:**
> "Yes, it's more boilerplate - interfaces, DTOs, multiple files. But for a staff engineer position, showing architectural discipline is important. This is how I'd structure a production system."

### Decision 2: BEGIN IMMEDIATE (Transaction Isolation)

Situation: We want to support atomic operations, while making our system concurrent safe.

=> We want to prevent race conditions, specifically check-and-act race conditions. 

=> Isolation: how you determine that the changes made by one transaction are made visible to other transactions.

=> Isolation level + locking mechanisms

=> SQLITE: offers ACID guarantees. It serializable (read committed) + single writer due to DB level lock.

**Show code:** `bulk_transfer_transaction.sql`. Show default locking strategy with BEGIN DEFERRED (shared lock + exclusive lock)

**Show code:** `internal/sqlite/client.go:46`
```go
dsn += "&_txlock=immediate"
```

**Script:**
> "This is one of the most important technical decisions. SQLite has three transaction modes:"

1. **BEGIN DEFERRED (default)**
   - "Lock acquired only at first write"
   - "Problem: Race condition window. Two transactions can read the same balance, both think there's enough money, both try to debit."

2. **BEGIN IMMEDIATE (my choice)**
   - "Acquires write lock immediately at BEGIN"
   - "Serializes write transactions from the start"
   - "No race condition - second transaction waits for first to finish"

3. **BEGIN EXCLUSIVE**
   - "Too aggressive - blocks all reads during transaction"
   - "Unnecessary since WAL mode allows concurrent reads"

**Key Point:**
> "BEGIN IMMEDIATE + WAL mode gives us: serialized writes (safe) + concurrent reads (performant). I have an integration test with 5 concurrent goroutines that verifies correct final balance."

### Decision 3: Integer Arithmetic for Money

**Show code:** `internal/http/dto.go:41`
```go
cents := int64(floatAmount * 100)
```

**Script:**
> "Never use floating-point for money. Classic example: 0.1 + 0.2 ≠ 0.3 in binary floating-point."

**Approach:**
- "API accepts string: `"amount": "100.50"`"
- "Parse to float, multiply by 100, store as int64: `10050` cents"
- "All arithmetic is exact - no rounding errors"

**Alternative:**
> "I considered a decimal library like shopspring/decimal, but for this assessment, integer cents is sufficient and simpler. Production might want decimal for audit precision."

---

## 🧪 Part 3: Development & Testing Strategy (5 min)

### Development Approach: Domain-First

**Show git log:**
```bash
git log --oneline --all
```

**Script:**
> "Let me show you my development approach through the commit history."

**Walk through commits:**
1. "Initialize boilerplate - project setup"
2. "Define domain models - Account, Transfer, business logic"
3. "Add service layer - orchestrates business operations"
4. "Add SQLite repository - concrete implementation"
5. "Add HTTP layer - API endpoint"
6. "Add E2E tests - full stack validation"
7. "Review input validation - iterative refinement"
8. "Add documentation - README with diagrams"

**Key Point:**
> "I started with domain - the business logic. Then infrastructure, then HTTP. This is domain-driven design influence: stabilize the core first, then add adapters."

### Testing Strategy: Three-Tier Pyramid

**Draw pyramid on whiteboard or show diagram:**
```
     E2E Tests (1)     ← Full stack
   Integration (3)     ← Service + Real DB
  Unit Tests (4)       ← Domain logic with mocks
```

**Script:**
> "I follow a three-tier testing pyramid:"

1. **Unit Tests** (bottom)
   - "Domain models: `models_test.go` - test HasSufficientFunds, Debit logic"
   - "Service: `service_test.go` - test orchestration with mocked repository"
   - "HTTP: `dto_test.go`, `post_transfers_test.go` - test validation, error mapping"
   - "Fast, isolated, high coverage"

2. **Integration Tests** (middle)
   - "`test/components/sqlite/account_store_test.go`"
   - "Real SQLite database, test SQL queries work"
   - "Tests: GetAccountByID, UpdateBalance, AddTransfers, Atomic"
   - "Notable: Concurrent write test - 5 goroutines, verify correct final balance"

3. **E2E Tests** (top)
   - "`test/http/http_test.go`"
   - "Full flow: HTTP request → Service → SQLite → Response"
   - "Seed database, send POST, verify balance and transfers in DB"

**Key Point:**
> "Each layer tests different concerns. Unit tests catch business logic bugs, integration tests catch SQL errors, E2E tests catch integration issues like DTO mapping."

### Test Coverage Highlights

**Script:**
> "Some notable test cases:"

- ✅ **Successful transfer** - happy path
- ✅ **Insufficient funds** - returns 422
- ✅ **Account not found** - returns 404
- ✅ **Concurrent writes** - 5 goroutines, correct final balance
- ✅ **Bulk insert boundaries** - single multi-row insert for all transfers
- ✅ **Empty transfer list** - no-op, returns 201
- ✅ **Atomicity** - rollback on error

**Demo:**
```bash
make unit_test        # Show fast execution
make integration_test # Show DB tests


go tool cover -func=unit_coverage.out
go tool cover -func=unit_coverage.out

```

### Commit Strategy: Semantic & Atomic

**Show commits again:**
```
feat(infrastructure): add account store with sqlite + integration tests
feat(core): add service layer to orchestrate business logic
```

**Script:**
> "I use semantic commit messages - `feat(scope): description`. Each commit is atomic - complete unit of work."

**Benefits:**
- "Reviewable - each commit tells a story"
- "Revertable - if something breaks, rollback specific commit"
- "Clear history - shows my thought process"

---

## 💻 Part 4: Code Walkthrough & Demo (5 min)

### Code Tour (3 min)

**Navigate in IDE:**

1. **Project Structure**
   ```
   internal/
   ├── core/          ← Domain + application logic
   ├── http/          ← HTTP adapter
   └── sqlite/        ← SQLite adapter
   ```

2. **Domain Model** (`internal/core/models.go`)
   ```go
   type Account struct {
       ID               int64
       BalanceCents     int64
       // ...
   }
   
   func (a *Account) HasSufficientFunds(total int64) bool {
       return a.BalanceCents >= total
   }
   ```
   > "Pure domain logic - no HTTP, no database concerns"

3. **Repository Interface** (`internal/core/repository.go`)
   ```go
   type AccountRepository interface {
       GetAccountByID(ctx, iban, bic) (Account, error)
       UpdateBalance(ctx, account) error
       AddTransfers(ctx, transfers) error
       Atomic(ctx, callback) error
   }
   ```
   > "This is the 'port' - contract between domain and infrastructure"

4. **Service** (`internal/core/service.go`)
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
   > "Orchestrates business operation. Callback pattern keeps domain independent of SQL transactions."

5. **SQLite Implementation** (`internal/sqlite/account_store.go`)
   > "Show `Atomic()` method - BEGIN/COMMIT/ROLLBACK"
   > "Show bulk insert logic in `AddTransfers()`"

6. **HTTP Handler** (`internal/http/post_transfers.go`)
   > "Validation with `validator.Struct(&req)`"
   > "Error mapping: `ErrAccountNotFound → 404`, `ErrInsufficientFunds → 422`"

### Live Demo (2 min)

**Terminal:**

1. **Run the service:**
   ```bash
   make local-run
   ```
   > "Service starts on localhost:8080"

2. **Send a bulk transfer:**
   ```bash
   curl -v -X POST http://localhost:8080/transfers/bulk \
     -H "Content-Type: application/json" \
     -d '{
       "organization_iban": "FR10474608000002006107XXXXX",
       "organization_bic": "OIVUSCLQXXX",
       "credit_transfers": [{
         "amount": "100.20",
         "currency": "EUR",
         "counterparty_name": "Test",
         "counterparty_bic": "TESTBIC",
         "counterparty_iban": "FR1234567890",
         "description": "Large transfer"
       }]
     }'
   ```
   > "201 Created - success"

3. **Query database to verify:**
   ```bash
   sqlite3 qonto_accounts.sqlite "SELECT balance_cents FROM bank_accounts WHERE iban='FR10474608000002006107XXXXX'"
   ```
   > "Balance is debited"

4. **Show insufficient funds error:**
   ```bash
   curl -v -X POST http://localhost:8080/transfers/bulk \
     -H "Content-Type: application/json" \
     -d '{
       "organization_iban": "FR10474608000002006107XXXXX",
       "organization_bic": "OIVUSCLQXXX",
       "credit_transfers": [{
         "amount": "99999999",
         "currency": "EUR",
         "counterparty_name": "Test",
         "counterparty_bic": "TESTBIC",
         "counterparty_iban": "FR1234567890",
         "description": "Large transfer"
       }]
     }'
   ```
   > "422 Unprocessable Entity - insufficient funds"

---

## 🚀 Part 5: Q&A - Anticipated Questions (Bonus)

### "Why hexagonal architecture for a small service?"

**Answer:**
> "Financial systems require rigorous testing. Hexagonal architecture enables me to unit test business logic with mocks, integration test the repository with real database, and E2E test the full stack - independently. For a staff engineer role, demonstrating architectural discipline is important."

### "How do you handle race conditions?"

**Answer:**
> "BEGIN IMMEDIATE acquires a write lock at transaction start, serializing concurrent writes. WAL mode allows concurrent reads. I have an integration test that runs 5 concurrent bulk transfers and verifies the final balance is correct. SQLite's SERIALIZABLE isolation guarantees safety."

### "How would you add idempotency?"

**Answer:**
> "Three steps: 1. Add `idempotency_key` field to API request. 2. Store key + outcome in database. 3. In repository, check if key exists: if yes, return cached outcome; if no, process and store. Domain logic doesn't change - this is purely infrastructure concern. Hexagonal architecture makes this straightforward."

### "What would you improve for production?"

**Answer:**
> "Documented in README: 1. PostgreSQL for row-level locking and horizontal scaling. 2. Idempotency keys for reliability. 3. Authentication and authorization. 4. Observability - metrics, traces, structured logs. 5. Rate limiting per organization. The hexagonal architecture makes these additions straightforward - they're additive, not rewrites."

### "How do you test this?"

**Answer:**
> "Three-tier pyramid: Unit tests mock repository and verify business logic. Integration tests use real SQLite and verify SQL correctness. E2E tests exercise full stack. I test happy paths, error paths, edge cases like bulk insert boundaries and concurrent writes. All tests run in parallel with isolated state."

### "Why callback pattern for transactions?"

**Answer:**
> "Keeps domain layer independent of SQL transactions. Repository's `Atomic()` method handles BEGIN/COMMIT/ROLLBACK, service passes a callback with business logic. This maintains hexagonal architecture principle: domain doesn't depend on infrastructure details like `sql.Tx`."

---

## 🎯 Closing (1 min)

**Summary:**
> "To summarize: I built a bulk transfer service with hexagonal architecture, ensuring atomic operations through SQLite's ACID guarantees. I prioritized correctness and testability - using BEGIN IMMEDIATE for transaction safety, integer arithmetic for money, and comprehensive three-tier testing. I documented the evolution path to production, showing I understand trade-offs."

**Key Message:**
> "For a staff engineer role, I wanted to demonstrate not just implementation skills, but architectural discipline, testing rigor, and production awareness. The hexagonal architecture may seem like overkill for a 3-hour assessment, but it shows how I'd structure a real production system."

**Ready for Questions:**
> "I'm happy to dive deeper into any of these topics or discuss how I'd evolve this for specific production requirements."

---

## 📋 Presentation Checklist

Before the interview:

- [ ] Open `docs/c4-component-architecture.png`
- [ ] Open `docs/sequence.png`
- [ ] Open IDE with project
- [ ] Terminal ready with service stopped
- [ ] `docs/sample1.json` ready for curl
- [ ] Git log command ready: `git log --oneline --all`
- [ ] Test commands ready: `make unit_test`, `make integration_test`
- [ ] This outline open for reference
- [ ] README.md open (for "What I'd Do Differently")

During the presentation:

- [ ] Share screen (full screen or specific window)
- [ ] Stay calm, speak clearly
- [ ] Use diagrams to explain concepts
- [ ] Show code, don't just talk about it
- [ ] Run live demo (tests + service)
- [ ] Acknowledge trade-offs
- [ ] Be ready to go deeper on any topic

---

## 🎤 Pro Tips

1. **Pace yourself** - Don't rush, 15-20 minutes is plenty
2. **Use diagrams** - Visual explanations are powerful
3. **Show, don't just tell** - Navigate code, run tests live
4. **Acknowledge trade-offs** - Shows maturity ("I chose X because Y, trade-off is Z")
5. **Be enthusiastic** - Show you enjoyed the problem
6. **Ask if questions** - "Does this make sense? Any questions before I continue?"
7. **Know your audience** - Staff engineers care about: architecture, testing, scalability, trade-offs
8. **Prepare for deeper dives** - Be ready to explain any technical decision in detail

---

**You've got this!** 🚀

Remember: You know this codebase intimately. You made thoughtful decisions. You tested thoroughly. You documented well. Be confident and show your expertise.

Good luck with your interview!
