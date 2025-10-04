package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"qonto/internal/core"
)

func TestHandler_PostTransfers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		requestBody      BulkTransferRequest
		setupMock        func(mock *MockBulkTransferProcessor)
		expectedStatus   int
		expectedBodyPart string
	}{
		{
			name: "successful_transfer_returns_201",
			requestBody: BulkTransferRequest{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				CreditTransfers: []CreditTransfer{
					{
						Amount:           "14.5",
						Currency:         "EUR",
						CounterpartyName: "Bip Bip",
						CounterpartyBIC:  "CRLYFRPPTOU",
						CounterpartyIBAN: "EE383680981021245685",
						Description:      "Test payment",
					},
				},
			},
			setupMock: func(mock *MockBulkTransferProcessor) {
				mock.EXPECT().
					ProcessBulkTransfer(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "insufficient_funds_returns_422",
			requestBody: BulkTransferRequest{
				OrganizationBIC:  "TESTBIC",
				OrganizationIBAN: "TESTIBAN",
				CreditTransfers: []CreditTransfer{
					{
						Amount:           "1000.00",
						Currency:         "EUR",
						CounterpartyName: "Test",
						CounterpartyBIC:  "BIC",
						CounterpartyIBAN: "IBAN",
						Description:      "Test",
					},
				},
			},
			setupMock: func(mock *MockBulkTransferProcessor) {
				mock.EXPECT().
					ProcessBulkTransfer(gomock.Any(), gomock.Any()).
					Return(core.ErrInsufficientFunds).
					Times(1)
			},
			expectedStatus:   http.StatusUnprocessableEntity,
			expectedBodyPart: "Insufficient funds",
		},
		{
			name: "account_not_found_returns_404",
			requestBody: BulkTransferRequest{
				OrganizationBIC:  "TESTBIC",
				OrganizationIBAN: "TESTIBAN",
				CreditTransfers: []CreditTransfer{
					{
						Amount:           "100.00",
						Currency:         "EUR",
						CounterpartyName: "Test",
						CounterpartyBIC:  "BIC",
						CounterpartyIBAN: "IBAN",
						Description:      "Test",
					},
				},
			},
			setupMock: func(mock *MockBulkTransferProcessor) {
				mock.EXPECT().
					ProcessBulkTransfer(gomock.Any(), gomock.Any()).
					Return(core.ErrAccountNotFound).
					Times(1)
			},
			expectedStatus:   http.StatusNotFound,
			expectedBodyPart: "Account not found",
		},
		{
			name: "generic_error_returns_500",
			requestBody: BulkTransferRequest{
				OrganizationBIC:  "TESTBIC",
				OrganizationIBAN: "TESTIBAN",
				CreditTransfers: []CreditTransfer{
					{
						Amount:           "100.00",
						Currency:         "EUR",
						CounterpartyName: "Test",
						CounterpartyBIC:  "BIC",
						CounterpartyIBAN: "IBAN",
						Description:      "Test",
					},
				},
			},
			setupMock: func(mock *MockBulkTransferProcessor) {
				mock.EXPECT().
					ProcessBulkTransfer(gomock.Any(), gomock.Any()).
					Return(errors.New("database connection failed")).
					Times(1)
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedBodyPart: "Failed to process bulk transfer",
		},
		{
			name: "validation_error_returns_400",
			requestBody: BulkTransferRequest{
				OrganizationBIC:  "TESTBIC",
				OrganizationIBAN: "", // Empty required field
				CreditTransfers: []CreditTransfer{
					{
						Amount:           "100.00",
						Currency:         "EUR",
						CounterpartyName: "Test",
						CounterpartyBIC:  "BIC",
						CounterpartyIBAN: "IBAN",
						Description:      "Test",
					},
				},
			},
			setupMock:        func(mock *MockBulkTransferProcessor) {},
			expectedStatus:   http.StatusBadRequest,
			expectedBodyPart: "Validation failed",
		},
		{
			name: "invalid_amount_format_returns_400",
			requestBody: BulkTransferRequest{
				OrganizationBIC:  "TESTBIC",
				OrganizationIBAN: "TESTIBAN",
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
			setupMock:        func(mock *MockBulkTransferProcessor) {},
			expectedStatus:   http.StatusBadRequest,
			expectedBodyPart: "invalid amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockProcessor := NewMockBulkTransferProcessor(ctrl)
			tt.setupMock(mockProcessor)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			handler := NewHandler(mockProcessor, logger)

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/transfers/bulk", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.PostTransfers(w, req)
			require.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBodyPart != "" {
				require.Contains(t, w.Body.String(), tt.expectedBodyPart)
			}
		})
	}
}
