# Key Technical Challenges:

1. Race Conditions: Multiple requests could try to modify the same account simultaneously
2. Decimal Precision: Amounts are strings with up to 2 decimal places, need precise conversion to cents
3. Atomic Operations: Either all transfers succeed or none do
4. Database Consistency: Need proper transaction isolation levels


API Design:

1. Single endpoint: POST /transfers/bulk
2. Input validation for all fields
3. Proper HTTP status codes (201 for success, 422 for insufficient funds)

Transaction Safety:

1. Database-level transactions for atomicity
2. Pessimistic locking on account balance updates
3. Proper error handling and rollback mechanisms


Implementation Strategy:

1. Input Validation: Validate JSON structure, IBAN/BIC formats, amount formats
2. Account Lookup: Find account by BIC/IBAN combination
3. Balance Check: Calculate total required amount and verify sufficient funds
4. Atomic Processing: Use database transaction to insert transfers and update balance
5. Error Handling: Proper rollback on any failure


# Locking Strategy Analysis for Concurrent Transfer Prevention

Multiple bulk transfer requests arrive simultaneously for the same account
Each reads the current balance, calculates if sufficient funds exist
All might pass validation simultaneously, leading to overdrafts
Example: Account has 1000€, two requests for 600€ each arrive simultaneously

-- Lock the account row for the duration of the transaction
BEGIN TRANSACTION;
SELECT balance_cents FROM bank_accounts 
WHERE iban = ? AND bic = ? 
FOR UPDATE;  -- This locks the row

-- Calculate total required amount
-- Validate sufficient funds
-- Update balance and insert transfers
UPDATE bank_accounts SET balance_cents = balance_cents - ? WHERE id = ?;
INSERT INTO transactions (...) VALUES (...);
COMMIT;

Pros:
- Prevents race conditions completely
- Simple to implement and understand
- Guarantees consistency
- Works well with SQLite/PostgreSQL

Cons:
- Can cause deadlocks if not careful
- May reduce throughput under high concurrency
- Blocks other operations on the same account

2. Optimistic Locking

ALTER TABLE bank_accounts ADD COLUMN version INTEGER DEFAULT 1;

-- Update with version check
UPDATE bank_accounts 
SET balance_cents = balance_cents - ?, version = version + 1 
WHERE id = ? AND version = ?;  -- Only update if version matches

-- Check if update affected any rows
-- If 0 rows affected, retry or fail


Pros:
Better performance under low contention
No deadlock risk
Non-blocking reads

Cons:
More complex implementation
Requires retry logic
Can fail under high contention
Need to modify schema


Recommended Approach: Pessimistic Locking with Proper Transaction Isolation

For this banking application, I recommend pessimistic locking because:
Financial Safety First: We cannot afford any race conditions
Simplicity: Easier to reason about and debug
Database Support: SQLite and PostgreSQL support SELECT ... FOR UPDATE
Load Balancing: Works across multiple server instances


## Architecture Decision

This implementation follows hexagonal principles:

- **Core**: Pure business logic, no external dependencies
- **Infrastructure**: Concrete implementations (HTTP, SQLite)

### Why this approach?
- **Testability**: Core logic testable without database/HTTP
- **Flexibility**: Easy to swap SQLite for PostgreSQL
- **Clarity**: Clear separation of concerns
- **Production-ready**: Maintainable structure

### Trade-offs considered
- More files than simple monolithic approach
- Justified by testability and maintainability gains
- Right-sized for production service (not over-engineered)