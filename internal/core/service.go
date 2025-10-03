package core

import (
	"context"
)

type Service struct {
	accountRepository AccountRepository
}

func NewService(accountRepo AccountRepository) Service {
	return Service{
		accountRepository: accountRepo,
	}
}

func (s Service) ProcessBulkTransfer(ctx context.Context, bulkTransfer BulkTransfer) error {
	if len(bulkTransfer.Transfers) == 0 {
		return nil
	}

	transactionCallback := func(r AccountRepository) error {
		account, err := r.GetAccountByID(ctx, bulkTransfer.OrganizationIBAN, bulkTransfer.OrganizationBIC)
		if err != nil {
			return err
		}

		if err = account.Debit(bulkTransfer.TotalAmount()); err != nil {
			return err
		}

		if err = r.UpdateBalance(ctx, account); err != nil {
			return err
		}

		transfers := make([]Transfer, len(bulkTransfer.Transfers))
		for i, transfer := range bulkTransfer.Transfers {
			transfer.BankAccountID = account.ID
			transfers[i] = transfer
		}

		return r.AddTransfers(ctx, transfers)
	}

	return s.accountRepository.Atomic(ctx, transactionCallback)
}
