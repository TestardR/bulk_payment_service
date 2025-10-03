package http

import (
	"testing"

	"github.com/stretchr/testify/require"

	"qonto/internal/core"
)

func TestParseAmountToCents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		amount        string
		expected      int64
		expectedError bool
	}{
		{
			name:     "whole_number",
			amount:   "999",
			expected: 99900,
		},
		{
			name:     "decimal_with_one_place",
			amount:   "14.5",
			expected: 1450,
		},
		{
			name:     "decimal_with_two_places",
			amount:   "13.22",
			expected: 1322,
		},
		{
			name:     "large_amount",
			amount:   "61238",
			expected: 6123800,
		},
		{
			name:     "zero",
			amount:   "0",
			expected: 0,
		},
		{
			name:     "zero_decimal",
			amount:   "0.00",
			expected: 0,
		},
		{
			name:     "small_amount",
			amount:   "0.01",
			expected: 1,
		},
		{
			name:     "amount_with_spaces",
			amount:   "  100.50  ",
			expected: 10050,
		},
		{
			name:          "empty_string",
			amount:        "",
			expectedError: true,
		},
		{
			name:          "invalid_format",
			amount:        "abc",
			expectedError: true,
		},
		{
			name:          "negative_amount",
			amount:        "-10.50",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseAmountToCents(tt.amount)

			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestBulkTransferRequest_ToDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		request  BulkTransferRequest
		expected func(t *testing.T, result core.BulkTransfer)
		wantErr  bool
	}{
		{
			name: "maps_all_fields_correctly",
			request: BulkTransferRequest{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				CreditTransfers: []CreditTransfer{
					{
						Amount:           "14.5",
						Currency:         "EUR",
						CounterpartyName: "Bip Bip",
						CounterpartyBIC:  "CRLYFRPPTOU",
						CounterpartyIBAN: "EE383680981021245685",
						Description:      "Wonderland/4410",
					},
				},
			},
			expected: func(t *testing.T, result core.BulkTransfer) {
				require.Equal(t, "OIVUSCLQXXX", result.OrganizationBIC)
				require.Equal(t, "FR10474608000002006107XXXXX", result.OrganizationIBAN)
				require.Len(t, result.Transfers, 1)

				transfer := result.Transfers[0]
				require.Equal(t, int64(1450), transfer.AmountCents)
				require.Equal(t, "EUR", transfer.Currency)
				require.Equal(t, "Bip Bip", transfer.CounterpartyName)
				require.Equal(t, "CRLYFRPPTOU", transfer.CounterpartyBIC)
				require.Equal(t, "EE383680981021245685", transfer.CounterpartyIBAN)
				require.Equal(t, "Wonderland/4410", transfer.Description)
			},
		},
		{
			name: "maps_multiple_transfers",
			request: BulkTransferRequest{
				OrganizationBIC:  "TESTBIC",
				OrganizationIBAN: "TEST123",
				CreditTransfers: []CreditTransfer{
					{
						Amount:           "100.00",
						Currency:         "EUR",
						CounterpartyName: "First",
						CounterpartyBIC:  "BIC1",
						CounterpartyIBAN: "IBAN1",
						Description:      "First transfer",
					},
					{
						Amount:           "200.50",
						Currency:         "EUR",
						CounterpartyName: "Second",
						CounterpartyBIC:  "BIC2",
						CounterpartyIBAN: "IBAN2",
						Description:      "Second transfer",
					},
				},
			},
			expected: func(t *testing.T, result core.BulkTransfer) {
				require.Len(t, result.Transfers, 2)
				require.Equal(t, int64(10000), result.Transfers[0].AmountCents)
				require.Equal(t, "First", result.Transfers[0].CounterpartyName)
				require.Equal(t, int64(20050), result.Transfers[1].AmountCents)
				require.Equal(t, "Second", result.Transfers[1].CounterpartyName)
			},
		},
		{
			name: "empty_transfers_list",
			request: BulkTransferRequest{
				OrganizationBIC:  "TESTBIC",
				OrganizationIBAN: "TEST123",
				CreditTransfers:  []CreditTransfer{},
			},
			expected: func(t *testing.T, result core.BulkTransfer) {
				require.Len(t, result.Transfers, 0)
			},
		},
		{
			name: "invalid_amount_returns_error",
			request: BulkTransferRequest{
				OrganizationBIC:  "TESTBIC",
				OrganizationIBAN: "TEST123",
				CreditTransfers: []CreditTransfer{
					{
						Amount:           "not-a-number",
						Currency:         "EUR",
						CounterpartyName: "Test",
						CounterpartyBIC:  "BIC",
						CounterpartyIBAN: "IBAN",
						Description:      "Test",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := tt.request.ToDomain()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.expected != nil {
				tt.expected(t, result)
			}
		})
	}
}
