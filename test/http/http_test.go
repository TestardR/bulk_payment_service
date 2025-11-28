package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	httpHandler "payment/internal/http"
)

func TestBulkTransfer_E2E_HappyPath(t *testing.T) {
	suite := NewTestSuite(t)
	defer suite.Teardown()

	const (
		orgName        = "Test Organization"
		orgIBAN        = "FR10474608000002006107XXXXX"
		orgBIC         = "OIVUSCLQXXX"
		initialBalance = 1000000
	)

	accountID := suite.SeedAccount(t, orgName, orgIBAN, orgBIC, initialBalance)

	requestBody := httpHandler.BulkTransferRequest{
		OrganizationBIC:  orgBIC,
		OrganizationIBAN: orgIBAN,
		CreditTransfers: []httpHandler.CreditTransfer{
			{
				Amount:           "100.50",
				Currency:         "EUR",
				CounterpartyName: "Alice Smith",
				CounterpartyBIC:  "CRLYFRPPTOU",
				CounterpartyIBAN: "EE383680981021245685",
				Description:      "Payment to Alice",
			},
			{
				Amount:           "250.75",
				Currency:         "EUR",
				CounterpartyName: "Bob Jones",
				CounterpartyBIC:  "DEUTDEFF",
				CounterpartyIBAN: "DE89370400440532013000",
				Description:      "Payment to Bob",
			},
			{
				Amount:           "75.25",
				Currency:         "EUR",
				CounterpartyName: "Charlie Brown",
				CounterpartyBIC:  "BNPAFRPP",
				CounterpartyIBAN: "FR1420041010050500013M02606",
				Description:      "Payment to Charlie",
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/transfers/bulk", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.Handler.PostTransfers(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "expected 201 Created, got: %s", w.Body.String())

	expectedBalance := int64(initialBalance - 42650)
	actualBalance := suite.GetAccountBalance(t, accountID)
	require.Equal(t, expectedBalance, actualBalance, "account balance should be debited")

	transactions := suite.GetTransactions(t, accountID)
	require.Len(t, transactions, 3, "should have 3 transactions")

	expectedTransactions := []struct {
		name        string
		iban        string
		bic         string
		amountCents int64
		currency    string
		description string
	}{
		{"Alice Smith", "EE383680981021245685", "CRLYFRPPTOU", -10050, "EUR", "Payment to Alice"},
		{"Bob Jones", "DE89370400440532013000", "DEUTDEFF", -25075, "EUR", "Payment to Bob"},
		{"Charlie Brown", "FR1420041010050500013M02606", "BNPAFRPP", -7525, "EUR", "Payment to Charlie"},
	}

	for i, tx := range transactions {
		expected := expectedTransactions[i]
		require.Equal(t, expected.name, tx.CounterpartyName, "transaction %d: counterparty name mismatch", i)
		require.Equal(t, expected.iban, tx.CounterpartyIBAN, "transaction %d: counterparty IBAN mismatch", i)
		require.Equal(t, expected.bic, tx.CounterpartyBIC, "transaction %d: counterparty BIC mismatch", i)
		require.Equal(t, expected.amountCents, tx.AmountCents, "transaction %d: amount mismatch", i)
		require.Equal(t, expected.currency, tx.Currency, "transaction %d: currency mismatch", i)
		require.Equal(t, expected.description, tx.Description, "transaction %d: description mismatch", i)
	}
}
