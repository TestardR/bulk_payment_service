package core

type Account struct {
	ID               int64
	OrganizationName string
	BalanceCents     int64
	IBAN             string
	BIC              string
}

func (a *Account) HasSufficientFunds(totalRequired int64) bool {
	return a.BalanceCents >= totalRequired
}

func (a *Account) Debit(amount int64) error {
	if !a.HasSufficientFunds(amount) {
		return ErrInsufficientFunds
	}

	a.BalanceCents -= amount
	return nil
}

type Transfer struct {
	ID               int64
	BankAccountID    int64
	CounterpartyName string
	CounterpartyIBAN string
	CounterpartyBIC  string
	AmountCents      int64
	Currency         string
	Description      string
}

type BulkTransfer struct {
	OrganizationBIC  string
	OrganizationIBAN string
	Transfers        []Transfer
}

func (bt BulkTransfer) TotalAmount() int64 {
	var total int64
	for _, t := range bt.Transfers {
		total += t.AmountCents
	}

	return total
}
