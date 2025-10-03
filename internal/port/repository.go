package port

import (
	"context"

	"qonto/internal/core"
)

type AccountRepository interface {
	ProcessBulkTransfer(context.Context, core.BulkTransfer) error
}
