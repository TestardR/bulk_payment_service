package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"qonto/internal/core"
)

type AccountStore struct {
	db *sql.DB
	tx *sql.Tx
}

func NewAccountStore(db *sql.DB) AccountStore {
	return AccountStore{
		db: db,
	}
}

func (s AccountStore) GetAccountByID(ctx context.Context, iban string, bic string) (core.Account, error) {
	if s.tx == nil {
		return core.Account{}, errors.New("GetAccountByID must be called within Atomic transaction")
	}

	query := `
			SELECT id, organization_name, balance_cents, iban, bic
			FROM bank_accounts
			WHERE iban = ? AND bic = ?
		`

	var account core.Account
	err := s.tx.QueryRowContext(ctx, query, iban, bic).Scan(
		&account.ID,
		&account.OrganizationName,
		&account.BalanceCents,
		&account.IBAN,
		&account.BIC,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.Account{}, core.ErrAccountNotFound
		}

		return core.Account{}, fmt.Errorf("failed to get account: %w", err)
	}

	return account, nil
}

func (s AccountStore) UpdateBalance(ctx context.Context, account core.Account) error {
	if s.tx == nil {
		return errors.New("UpdateBalance must be called within Atomic transaction")
	}

	query := `
		UPDATE bank_accounts
		SET balance_cents = ?
		WHERE id = ?
	`

	result, err := s.tx.ExecContext(ctx, query, account.BalanceCents, account.ID)
	if err != nil {
		return fmt.Errorf("failed to execute update: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated for account ID %d", account.ID)
	}

	return nil
}

func (s AccountStore) AddTransfers(ctx context.Context, transfers []core.Transfer) error {
	if s.tx == nil {
		return errors.New("AddTransfers must be called within Atomic transaction")
	}

	// SQLite has a limit of 999 parameters (SQLITE_MAX_VARIABLE_NUMBER)
	// With 7 parameters per transfer, we can insert 142 transfers at once
	// We use 100 as a safe batch size
	const batchSize = 100
	for i := 0; i < len(transfers); i += batchSize {
		end := i + batchSize
		if end > len(transfers) {
			end = len(transfers)
		}
		if err := s.addTransfers(ctx, transfers[i:end]); err != nil {
			return err
		}
	}

	return nil
}

func (s AccountStore) addTransfers(ctx context.Context, transfers []core.Transfer) error {
	baseQuery := `
		INSERT INTO transactions (
			counterparty_name,
			counterparty_iban,
			counterparty_bic,
			amount_cents,
			amount_currency,
			bank_account_id,
			description
		) VALUES `

	valuePlaceholder := "(?, ?, ?, ?, ?, ?, ?)"

	query := baseQuery + valuePlaceholder
	for i := 1; i < len(transfers); i++ {
		query += ", " + valuePlaceholder
	}

	args := make([]interface{}, 0, len(transfers)*7)
	for _, transfer := range transfers {
		if transfer.BankAccountID == 0 {
			return fmt.Errorf("transfer missing bank_account_id")
		}

		amountCents := -transfer.AmountCents

		args = append(args,
			transfer.CounterpartyName,
			transfer.CounterpartyIBAN,
			transfer.CounterpartyBIC,
			amountCents,
			transfer.Currency,
			transfer.BankAccountID,
			transfer.Description,
		)
	}

	_, err := s.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk insert transfers: %w", err)
	}

	return nil
}

func (s AccountStore) Atomic(ctx context.Context, cb func(core.AccountRepository) error) error {
	// SQLite doesn't support SELECT FOR UPDATE, but we use BEGIN IMMEDIATE instead
	// (configured via _txlock=immediate in DSN)
	//
	// BEGIN IMMEDIATE acquires a RESERVED lock immediately:
	// - Prevents concurrent writes (serializes write transactions)
	// - Allows concurrent reads (with WAL mode)
	// - No race window between SELECT and UPDATE
	//
	// This is NOT BEGIN EXCLUSIVE (which would block all reads unnecessarily)
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelDefault,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	txStore := AccountStore{
		tx: tx,
	}

	if err = cb(txStore); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
