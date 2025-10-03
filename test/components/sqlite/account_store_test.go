package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"qonto/internal/core"
	"qonto/internal/sqlite"
)

func TestAccountStore_GetAccountByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupAccount  func(suite *TestSuite) (iban, bic string)
		expectedError error
	}{
		{
			name: "existing_account_returns_account",
			setupAccount: func(suite *TestSuite) (string, string) {
				iban := "FR1420041010050500013M02606"
				bic := "PSSTFRPPMON"
				suite.SeedAccount(t, "Acme Corp", iban, bic, 1000000)
				return iban, bic
			},
		},
		{
			name: "non_existent_account_returns_error",
			setupAccount: func(suite *TestSuite) (string, string) {
				return "FR9999999999999999999999999", "INVALIDBIC"
			},
			expectedError: core.ErrAccountNotFound,
		},
		{
			name: "partial_match_iban_only_returns_error",
			setupAccount: func(suite *TestSuite) (string, string) {
				iban := "FR2220041010050500013M02607"
				bic := "PSSTFRPPMON"
				suite.SeedAccount(t, "Acme Corp", iban, bic, 1000000)
				return iban, "WRONGBIC"
			},
			expectedError: core.ErrAccountNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := NewTestSuite(t)
			defer suite.Teardown()

			store := sqlite.NewAccountStore(suite.DB)
			iban, bic := tt.setupAccount(suite)

			var result core.Account
			err := store.Atomic(context.Background(), func(r core.AccountRepository) error {
				account, err := r.GetAccountByID(context.Background(), iban, bic)
				result = account
				return err
			})

			if tt.expectedError != nil {
				require.ErrorIs(t, err, tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, iban, result.IBAN)
			require.Equal(t, bic, result.BIC)
			require.NotZero(t, result.ID)
		})
	}
}

func TestAccountStore_UpdateBalance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		initialBalance int64
		newBalance     int64
	}{
		{
			name:           "update_to_lower_balance",
			initialBalance: 1000000,
			newBalance:     500000,
		},
		{
			name:           "update_to_higher_balance",
			initialBalance: 1000000,
			newBalance:     2000000,
		},
		{
			name:           "update_to_zero",
			initialBalance: 1000000,
			newBalance:     0,
		},
		{
			name:           "update_negative_balance",
			initialBalance: 1000000,
			newBalance:     -50000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := NewTestSuite(t)
			defer suite.Teardown()

			store := sqlite.NewAccountStore(suite.DB)

			iban := "FR1420041010050500013M02606"
			bic := "PSSTFRPPMON"
			accountID := suite.SeedAccount(t, "Test Org", iban, bic, tt.initialBalance)

			err := store.Atomic(context.Background(), func(r core.AccountRepository) error {
				account, err := r.GetAccountByID(context.Background(), iban, bic)
				if err != nil {
					return err
				}

				account.BalanceCents = tt.newBalance
				return r.UpdateBalance(context.Background(), account)
			})
			require.NoError(t, err)

			actualBalance := suite.GetAccountBalance(t, accountID)
			require.Equal(t, tt.newBalance, actualBalance)
		})
	}
}

func TestAccountStore_AddTransfers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		transferCount    int
		expectedDBAmount func(transfer core.Transfer) int64
	}{
		{
			name:          "single_transfer",
			transferCount: 1,
			expectedDBAmount: func(t core.Transfer) int64 {
				return -t.AmountCents
			},
		},
		{
			name:          "multiple_transfers",
			transferCount: 5,
			expectedDBAmount: func(t core.Transfer) int64 {
				return -t.AmountCents
			},
		},
		{
			name:          "batch_size_boundary",
			transferCount: 100, // Exactly at batch size
			expectedDBAmount: func(t core.Transfer) int64 {
				return -t.AmountCents
			},
		},
		{
			name:          "exceeds_batch_size",
			transferCount: 150, // Requires multiple batches
			expectedDBAmount: func(t core.Transfer) int64 {
				return -t.AmountCents
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			suite := NewTestSuite(t)
			defer suite.Teardown()

			store := sqlite.NewAccountStore(suite.DB)

			iban := "FR1420041010050500013M02606"
			bic := "PSSTFRPPMON"
			accountID := suite.SeedAccount(t, "Test Org", iban, bic, 10000000)

			transfers := make([]core.Transfer, tt.transferCount)
			for i := 0; i < tt.transferCount; i++ {
				transfers[i] = core.Transfer{
					BankAccountID:    accountID,
					CounterpartyName: "Recipient",
					CounterpartyIBAN: "GB33BUKB20201555555555",
					CounterpartyBIC:  "BUKBGB22",
					AmountCents:      10000,
					Currency:         "EUR",
					Description:      "Payment",
				}
			}

			err := store.Atomic(context.Background(), func(r core.AccountRepository) error {
				return r.AddTransfers(context.Background(), transfers)
			})
			require.NoError(t, err)

			count := suite.CountTransactions(t, accountID)
			require.Equal(t, tt.transferCount, count)

			dbTransfers := suite.GetTransactions(t, accountID)
			require.Len(t, dbTransfers, tt.transferCount)

			for i, got := range dbTransfers {
				expectedAmount := tt.expectedDBAmount(transfers[i])
				require.Equal(t, expectedAmount, got.AmountCents, "transfer %d: expected amount %d, got %d", i, expectedAmount, got.AmountCents)
			}
		})
	}
}

func TestAccountStore_Atomic_CommitSuccess(t *testing.T) {
	t.Parallel()

	suite := NewTestSuite(t)
	defer suite.Teardown()

	store := sqlite.NewAccountStore(suite.DB)

	iban := "FR1420041010050500013M02606"
	bic := "PSSTFRPPMON"
	accountID := suite.SeedAccount(t, "Test Org", iban, bic, 1000000)

	err := store.Atomic(context.Background(), func(r core.AccountRepository) error {
		account, err := r.GetAccountByID(context.Background(), iban, bic)
		if err != nil {
			return err
		}

		account.BalanceCents = 500000
		if err := r.UpdateBalance(context.Background(), account); err != nil {
			return err
		}

		transfers := []core.Transfer{
			{
				BankAccountID:    accountID,
				CounterpartyName: "Recipient",
				CounterpartyIBAN: "GB33BUKB20201555555555",
				CounterpartyBIC:  "BUKBGB22",
				AmountCents:      500000,
				Currency:         "EUR",
				Description:      "Payment",
			},
		}

		return r.AddTransfers(context.Background(), transfers)
	})
	require.NoError(t, err)

	balance := suite.GetAccountBalance(t, accountID)
	require.Equal(t, int64(500000), balance)

	count := suite.CountTransactions(t, accountID)
	require.Equal(t, 1, count)
}

func TestAccountStore_Atomic_ConcurrentWrites(t *testing.T) {
	t.Parallel()

	suite := NewTestSuite(t)
	defer suite.Teardown()

	store := sqlite.NewAccountStore(suite.DB)

	iban := "FR1420041010050500013M02606"
	bic := "PSSTFRPPMON"
	initialBalance := int64(1_000_000)
	accountID := suite.SeedAccount(t, "Test Org", iban, bic, initialBalance)

	// Simulate concurrent bulk transfer processing
	// With BEGIN IMMEDIATE, these should serialize (not race)
	const numConcurrent = 5
	const debitAmount = 100_000

	errChan := make(chan error, numConcurrent)
	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			err := store.Atomic(context.Background(), func(r core.AccountRepository) error {
				account, err := r.GetAccountByID(context.Background(), iban, bic)
				if err != nil {
					return err
				}

				account.BalanceCents -= debitAmount
				if err := r.UpdateBalance(context.Background(), account); err != nil {
					return err
				}

				transfers := []core.Transfer{
					{
						BankAccountID:    accountID,
						CounterpartyName: "Recipient",
						CounterpartyIBAN: "GB33BUKB20201555555555",
						CounterpartyBIC:  "BUKBGB22",
						AmountCents:      debitAmount,
						Currency:         "EUR",
						Description:      "Concurrent test",
					},
				}

				return r.AddTransfers(context.Background(), transfers)
			})
			errChan <- err
		}(i)
	}

	for i := 0; i < numConcurrent; i++ {
		err := <-errChan
		require.NoError(t, err, "concurrent transaction %d failed", i)
	}

	expectedBalance := initialBalance - (numConcurrent * debitAmount)
	actualBalance := suite.GetAccountBalance(t, accountID)
	require.Equal(t, expectedBalance, actualBalance, "balance should reflect all %d concurrent debits", numConcurrent)

	count := suite.CountTransactions(t, accountID)
	require.Equal(t, numConcurrent, count, "should have %d transfers from concurrent operations", numConcurrent)
}
