package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"qonto/internal/core"
)

type Server struct {
	httpServer          *http.Server
	bulkTransferHandler Handler
	logger              core.Logger
}

func NewServer(
	bulkTransferProcessor BulkTransferProcessor,
	logger core.Logger,
	config Config,
) *Server {
	bulkTransferHandler := NewHandler(bulkTransferProcessor, logger)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /transfers/bulk", bulkTransferHandler.PostTransfers)

	httpServer := &http.Server{
		Addr:         config.Address,
		Handler:      mux,
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
	}

	return &Server{
		httpServer:          httpServer,
		bulkTransferHandler: bulkTransferHandler,
		logger:              logger,
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Starting HTTP server", "port", s.httpServer.Addr)

	go func() {
		<-ctx.Done()
		s.logger.InfoContext(ctx, "Shutting down HTTP server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.ErrorContext(ctx, "Error shutting down HTTP server", "error", err)
		}
	}()

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Stopping HTTP server")
	return s.httpServer.Shutdown(ctx)
}
