package core

import (
	"context"
)

//go:generate go tool go.uber.org/mock/mockgen -source=repository.go -destination=repository_mock.go -package=core

type AccountRepository interface {
	GetAccountByID(ctx context.Context, IBAN string, BIC string) (Account, error)
	AddTransfers(ctx context.Context, transfers []Transfer) error
	UpdateBalance(ctx context.Context, account Account) error
	Atomic(ctx context.Context, cb func(r AccountRepository) error) error
}
