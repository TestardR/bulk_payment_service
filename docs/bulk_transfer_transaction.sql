BEGIN; -- lock READ COMMITTED

-- Step 1: Fetch account by IBAN and BIC
SELECT id, organization_name, balance_cents, iban, bic --
FROM bank_accounts FOR UPDATE
WHERE iban = 'FR10474608000002006107XXXXX' 
  AND bic = 'OIVUSCLQXXX';

-- Step 2: Validate sufficient funds

-- Step 3: Update account balance 
UPDATE bank_accounts
SET balance_cents = balance_cents - 42650
WHERE id = 1;

-- Step 4: Insert all transfers
INSERT INTO transactions (
    counterparty_name,
    counterparty_iban,
    counterparty_bic,
    amount_cents,       
    amount_currency,
    bank_account_id,
    description
) VALUES
    ('Charlie Brown', 'FR1420041010050500013M02606', 'BNPAFRPP', -7525, 'EUR', 1, 'Payment to Charlie');

-- Step 5: Commit transaction
COMMIT;

-- 10ms
-- SQLITE
-- 100 transfers => 1000ms 100t/s


-- POSTGRES + index on bank_accounts (iban, bic)
-- 100 transfers
-- 1 acc => 100 t/s
-- 10 acc => 1000 t/s
-- 100 acc => 10 000 t/s --


