package core

import (
	"errors"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds for bulk transfer")
	ErrAccountNotFound   = errors.New("account not found")
)
