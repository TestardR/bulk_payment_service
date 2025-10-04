package http

import (
	"context"
	"errors"
	"net/http"
)

type Logger interface {
	InfoContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

func loggingMiddleware(logger Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.InfoContext(
			r.Context(),
			"request",
			"method", r.Method,
			"path", r.URL.Path,
		)

		next.ServeHTTP(w, r)
	})
}

type Server struct {
	httpServer          *http.Server
	bulkTransferHandler Handler
	logger              Logger
}

func NewServer(
	bulkTransferProcessor BulkTransferProcessor,
	logger Logger,
	config Config,
) *Server {
	bulkTransferHandler := NewHandler(bulkTransferProcessor, logger)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /transfers/bulk", bulkTransferHandler.PostTransfers)

	handler := loggingMiddleware(logger, mux)

	httpServer := &http.Server{
		Addr:         config.Address,
		Handler:      handler,
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
	s.logger.InfoContext(ctx, "Starting HTTP server", "address", s.httpServer.Addr)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.ErrorContext(ctx, "HTTP server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Stopping HTTP server")
	return s.httpServer.Shutdown(ctx)
}
