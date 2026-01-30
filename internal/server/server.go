package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	config "github.com/itstheanurag/executioner/internal/config"
	"github.com/rs/zerolog"
)

type Server struct {
	conf       *config.Config
	logger     *zerolog.Logger
	httpServer *http.Server
}

func New(
	conf *config.Config,
	logger *zerolog.Logger,
) (*Server, error) {

	mux := http.NewServeMux()

	// health check — always add this early
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	httpServer := &http.Server{
		Addr:         ":" + conf.Server.Port,
		Handler:      mux,
		ReadTimeout:  time.Duration(conf.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(conf.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(conf.Server.IdleTimeout) * time.Second,
	}

	s := &Server{
		conf:       conf,
		logger:     logger,
		httpServer: httpServer,
	}

	return s, nil
}

func (s *Server) Start() error {
	s.logger.Info().
		Str("port", s.conf.Server.Port).
		Msg("starting HTTP server")

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failed: %w", err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	// Future:
	// - close DB
	// - stop job queue
	// - stop workers

	return nil
}
