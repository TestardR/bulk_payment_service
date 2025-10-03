package integration

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"qonto/internal/sqlite"
)

type TestSuite struct {
	DB       *sql.DB
	DBPath   string
	Client   *sqlite.Client
	teardown func()
}

func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_payments.db")

	config := sqlite.Config{
		DatabasePath: dbPath,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		BusyTimeout:  30 * time.Second,
		EnableWAL:    true,
	}

	client, err := sqlite.NewClient(config)
	require.NoError(t, err, "failed to create test client")

	schema := `
		CREATE TABLE IF NOT EXISTS bank_accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_name TEXT NOT NULL,
			balance_cents INTEGER NOT NULL DEFAULT 0,
			iban TEXT NOT NULL,
			bic TEXT NOT NULL,
			UNIQUE(iban, bic)
		);

		CREATE TABLE IF NOT EXISTS transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			counterparty_name TEXT NOT NULL,
			counterparty_iban TEXT NOT NULL,
			counterparty_bic TEXT NOT NULL,
			amount_cents INTEGER NOT NULL,
			amount_currency TEXT NOT NULL DEFAULT 'EUR',
			bank_account_id INTEGER NOT NULL,
			description TEXT
		);
	`

	_, err = client.DB().Exec(schema)
	require.NoError(t, err, "failed to create schema")

	suite := &TestSuite{
		DB:     client.DB(),
		DBPath: dbPath,
		Client: client,
		teardown: func() {
			client.Close()
			os.Remove(dbPath)
		},
	}

	return suite
}

func (s *TestSuite) Teardown() {
	s.teardown()
}

func (s *TestSuite) SeedAccount(t *testing.T, orgName, iban, bic string, balanceCents int64) int64 {
	t.Helper()

	query := `
		INSERT INTO bank_accounts (organization_name, iban, bic, balance_cents)
		VALUES (?, ?, ?, ?)
	`

	result, err := s.DB.Exec(query, orgName, iban, bic, balanceCents)
	require.NoError(t, err, "failed to seed account")

	id, err := result.LastInsertId()
	require.NoError(t, err, "failed to get inserted account ID")

	return id
}

func (s *TestSuite) GetAccountBalance(t *testing.T, accountID int64) int64 {
	t.Helper()

	var balance int64
	err := s.DB.QueryRow("SELECT balance_cents FROM bank_accounts WHERE id = ?", accountID).Scan(&balance)
	require.NoError(t, err, "failed to get account balance")

	return balance
}

func (s *TestSuite) CountTransactions(t *testing.T, accountID int64) int {
	t.Helper()

	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM transactions WHERE bank_account_id = ?", accountID).Scan(&count)
	require.NoError(t, err, "failed to count transactions")

	return count
}

type Transaction struct {
	ID               int64
	CounterpartyName string
	CounterpartyIBAN string
	CounterpartyBIC  string
	AmountCents      int64
	Currency         string
	Description      string
}

func (s *TestSuite) GetTransactions(t *testing.T, accountID int64) []Transaction {
	t.Helper()

	query := `
		SELECT id, counterparty_name, counterparty_iban, counterparty_bic,
		       amount_cents, amount_currency, description
		FROM transactions
		WHERE bank_account_id = ?
		ORDER BY id
	`

	rows, err := s.DB.Query(query, accountID)
	require.NoError(t, err, "failed to query transactions")
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var tx Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.CounterpartyName,
			&tx.CounterpartyIBAN,
			&tx.CounterpartyBIC,
			&tx.AmountCents,
			&tx.Currency,
			&tx.Description,
		)
		require.NoError(t, err, "failed to scan transaction")
		transactions = append(transactions, tx)
	}

	require.NoError(t, rows.Err(), "error iterating transactions")
	return transactions
}
