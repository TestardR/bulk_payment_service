# Transaction Concepts - Interview Reference

## 🎯 Core Concept

**Transaction** = A unit of work that must be executed **atomically** — either all operations succeed, or none do.

---

## 🔒 ACID Properties

| Property | What It Means | Example |
|----------|---------------|---------|
| **Atomicity** | All-or-nothing | Balance + transfers succeed together or both fail |
| **Consistency** | Database rules maintained | Balance can't go negative |
| **Isolation** | Transactions don't interfere | Concurrent transfers see correct balance |
| **Durability** | Changes persist after commit | Data survives crashes |

---

## 🔍 Isolation Levels (SQL Standard)

### From Least to Most Strict

| Level | Dirty Reads | Non-Repeatable Reads | Phantom Reads | Use Case |
|-------|-------------|---------------------|----------------|----------|
| **Read Uncommitted** | ✅ Allowed | ✅ Allowed | ✅ Allowed | Never use |
| **Read Committed** | ❌ Blocked | ✅ Allowed | ✅ Allowed | **Most common** |
| **Repeatable Read** | ❌ Blocked | ❌ Blocked | ✅ Allowed | Banking |
| **Serializable** | ❌ Blocked | ❌ Blocked | ❌ Blocked | **Maximum safety** |

### Key Terms
- **Dirty Read**: Reading uncommitted data from another transaction
- **Non-Repeatable Read**: Same row returns different values between reads
- **Phantom Read**: New rows appear/disappear in range queries

---

## 🔐 Lock Types

| Lock Type | Allows | Blocks | Purpose |
|-----------|--------|--------|---------|
| **Shared (S)** | Reading | Writing | Multiple readers |
| **Exclusive (X)** | Writing | Reading + Writing | Single writer |
| **Row-level** | Operations on other rows | Operations on same row | Concurrent operations |
| **Table-level** | Nothing else | All operations | Global serialization |

---

## 🎯 Your Implementation: SQLite

### Default Configuration
```sql
-- SQLite default behavior
BEGIN;  -- Actually BEGIN DEFERRED
-- Acquires shared lock on READ
-- Acquires exclusive lock on WRITE
```

**Problems:**
- **Race Condition**: Two transactions read same balance, both think they have enough money
- **Busy Timeout**: Second writer fails with `SQLITE_BUSY`

### Your Solution: BEGIN IMMEDIATE
```sql
BEGIN IMMEDIATE;  -- Your configuration
-- Acquires exclusive lock immediately
-- Serializes all write transactions
-- Prevents race conditions
```

**Benefits:**
- ✅ **No race conditions** - Transactions serialize
- ✅ **Consistent reads** - Each transaction sees updated balance
- ✅ **WAL mode** - Concurrent reads still allowed

---

## 🏦 Financial Systems: Read Committed vs Serializable

### Why Read Committed is Often Sufficient

**For single READ operations:**
```sql
-- This is safe with Read Committed
SELECT balance_cents FROM accounts WHERE id = 1;
UPDATE accounts SET balance_cents = balance_cents - 1000 WHERE id = 1;
```

**Why?**
- You want the **latest committed balance**
- Dirty reads are blocked (can't see uncommitted changes)
- Non-repeatable reads are acceptable (you only read once)

### When You Need Serializable

**For complex queries:**
```sql
-- This needs Serializable
SELECT SUM(balance_cents) FROM accounts WHERE region = 'EU';
-- If new accounts are added during transaction, sum changes
```

---

## 🚀 SQLite vs PostgreSQL

| Aspect | SQLite (Your Code) | PostgreSQL (Production) |
|--------|-------------------|-------------------------|
| **Lock Granularity** | Database-level | Row-level |
| **Concurrency** | Single writer globally | Multiple writers to different rows |
| **Isolation** | Always SERIALIZABLE | Configurable (default: Read Committed) |
| **SELECT FOR UPDATE** | Not supported | ✅ Supported |
| **Deadlocks** | Impossible (global lock) | Possible (need detection/retry) |
| **Scaling** | Single instance | Horizontal scaling |

---

## 💡 Interview Talking Points

### Why BEGIN IMMEDIATE?

> "SQLite's default BEGIN DEFERRED creates a race condition window. Two transactions can read the same balance, both think they have enough money, both try to debit. BEGIN IMMEDIATE acquires the write lock immediately, serializing transactions and preventing this race condition."

### Why Not PostgreSQL?

> "For this assessment, SQLite demonstrates transaction concepts perfectly. In production, I'd use PostgreSQL for row-level locking, allowing concurrent operations on different accounts. SQLite serializes all writes globally, which is fine for single instance but won't scale horizontally."

### Isolation Level Choice

> "For bulk transfers with single balance reads, Read Committed would be sufficient in PostgreSQL. We want the latest committed balance, and dirty reads are blocked. Serializable is overkill unless you have complex multi-table queries or need phantom read protection."

---

## 🎯 Key Takeaways

1. **Isolation levels** control how strict visibility rules are
2. **Locks** enforce isolation (stricter = more locking = less concurrency)
3. **SQLite** uses database-level locking (simple but limited)
4. **PostgreSQL** uses row-level locking (complex but scalable)
5. **Financial systems** often need Serializable for complex operations
6. **Your implementation** uses BEGIN IMMEDIATE to prevent race conditions

---

## 🔧 Your Transaction Flow

```sql
BEGIN IMMEDIATE;                    -- Acquire write lock immediately
SELECT balance_cents FROM accounts WHERE iban = ? AND bic = ?;
-- Business logic: Check sufficient funds
UPDATE accounts SET balance_cents = balance_cents - ? WHERE id = ?;
INSERT INTO transactions (...) VALUES (...), (...), (...);
COMMIT;                            -- Release lock, make changes visible
```

**Why this works:**
- ✅ **Atomic**: All operations succeed or fail together
- ✅ **Consistent**: Balance rules enforced by business logic
- ✅ **Isolated**: BEGIN IMMEDIATE prevents race conditions
- ✅ **Durable**: Changes persist after commit





