package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccount_HasSufficientFunds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		balance       int64
		required      int64
		expectedValid bool
	}{
		{
			name:          "sufficient funds - exact amount",
			balance:       10000,
			required:      10000,
			expectedValid: true,
		},
		{
			name:          "sufficient funds - more than required",
			balance:       10000,
			required:      5000,
			expectedValid: true,
		},
		{
			name:          "insufficient funds - by 1 cent",
			balance:       9999,
			required:      10000,
			expectedValid: false,
		},
		{
			name:          "zero balance - zero required",
			balance:       0,
			required:      0,
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			account := &Account{
				BalanceCents: tt.balance,
			}

			got := account.HasSufficientFunds(tt.required)
			require.Equal(t, tt.expectedValid, got)
		})
	}
}

func TestAccount_Debit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		initialBalance  int64
		debitAmount     int64
		expectedBalance int64
		expectedError   error
	}{
		{
			name:            "successful debit - partial amount",
			initialBalance:  10000,
			debitAmount:     3000,
			expectedBalance: 7000,
		},
		{
			name:            "successful debit - exact amount",
			initialBalance:  10000,
			debitAmount:     10000,
			expectedBalance: 0,
		},
		{
			name:            "failed debit - insufficient funds",
			initialBalance:  5000,
			debitAmount:     10000,
			expectedBalance: 5000,
			expectedError:   ErrInsufficientFunds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			account := &Account{
				BalanceCents: tt.initialBalance,
			}

			err := account.Debit(tt.debitAmount)
			if (tt.expectedError != nil) || (err != nil) {
				require.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expectedBalance, account.BalanceCents)
		})
	}
}
