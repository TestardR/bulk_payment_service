package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"qonto/internal/core"
)

//go:generate go tool go.uber.org/mock/mockgen -source=post_transfers.go -destination=service_mock.go -package=http

type BulkTransferProcessor interface {
	ProcessBulkTransfer(ctx context.Context, bulkTransfer core.BulkTransfer) error
}

type Handler struct {
	bulkTransferProcessor BulkTransferProcessor
	logger                core.Logger
}

func NewHandler(bulkTransferProcessor BulkTransferProcessor, logger core.Logger) Handler {
	return Handler{
		bulkTransferProcessor: bulkTransferProcessor,
		logger:                logger,
	}
}

func (h Handler) PostTransfers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req BulkTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	bulkTransfer, err := req.ToDomain()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.bulkTransferProcessor.ProcessBulkTransfer(ctx, bulkTransfer); err != nil {
		if errors.Is(err, core.ErrAccountNotFound) {
			http.Error(w, "Account not found", http.StatusNotFound)
			return
		}

		if errors.Is(err, core.ErrInsufficientFunds) {
			http.Error(w, "Insufficient funds for bulk transfer", http.StatusUnprocessableEntity)
			return
		}

		h.logger.ErrorContext(ctx, "Failed to process bulk transfer", "error", err)
		http.Error(w, "Failed to process bulk transfer", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
