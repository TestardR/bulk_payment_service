package http

import (
	"fmt"
	"strconv"
	"strings"

	"payment/internal/core"
)

type BulkTransferRequest struct {
	OrganizationBIC  string           `json:"organization_bic" validate:"required"`
	OrganizationIBAN string           `json:"organization_iban" validate:"required"`
	CreditTransfers  []CreditTransfer `json:"credit_transfers" validate:"required,min=1,dive"`
}

type CreditTransfer struct {
	Amount           string `json:"amount" validate:"required,gt=0"`
	Currency         string `json:"currency" validate:"required,eq=EUR"`
	CounterpartyName string `json:"counterparty_name" validate:"required"`
	CounterpartyBIC  string `json:"counterparty_bic" validate:"required"`
	CounterpartyIBAN string `json:"counterparty_iban" validate:"required"`
	Description      string `json:"description" validate:"required"`
}

func ParseAmountToCents(amount string) (int64, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return 0, fmt.Errorf("amount cannot be empty")
	}

	floatAmount, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format: %w", err)
	}

	if floatAmount < 0 {
		return 0, fmt.Errorf("amount cannot be negative")
	}

	cents := int64(floatAmount * 100)
	// 0.1 + 0.2 != 0.3

	return cents, nil
}

func (req BulkTransferRequest) ToDomain() (core.BulkTransfer, error) {
	transfers := make([]core.Transfer, 0, len(req.CreditTransfers))

	for _, ct := range req.CreditTransfers {
		amountCents, err := ParseAmountToCents(ct.Amount)
		if err != nil {
			return core.BulkTransfer{}, fmt.Errorf("invalid amount for transfer %s: %w", ct.Amount, err)
		}

		transfer := core.Transfer{
			CounterpartyName: ct.CounterpartyName,
			CounterpartyIBAN: ct.CounterpartyIBAN,
			CounterpartyBIC:  ct.CounterpartyBIC,
			AmountCents:      amountCents,
			Currency:         ct.Currency,
			Description:      ct.Description,
		}

		transfers = append(transfers, transfer)
	}

	return core.BulkTransfer{
		OrganizationBIC:  req.OrganizationBIC,
		OrganizationIBAN: req.OrganizationIBAN,
		Transfers:        transfers,
	}, nil
}
