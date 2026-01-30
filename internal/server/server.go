package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/itstheanurag/executioner/internal/api"
	config "github.com/itstheanurag/executioner/internal/config"
	"github.com/itstheanurag/executioner/internal/database"
	"github.com/itstheanurag/executioner/internal/executor"
	"github.com/itstheanurag/executioner/internal/languages"
	"github.com/itstheanurag/executioner/internal/limiter"
	"github.com/itstheanurag/executioner/internal/queue"
	"github.com/itstheanurag/executioner/internal/sandbox"
	"github.com/itstheanurag/executioner/internal/worker"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

type Server struct {
	conf        *config.Config
	logger      *zerolog.Logger
	httpServer  *http.Server
	db          *database.Database
	registry    *languages.Registry
	sandbox     sandbox.Sandbox
	executor    *executor.Executor
	queue       *queue.Manager
	workers     []*worker.Worker
	rateLimiter *limiter.RateLimiter
	cancelFunc  context.CancelFunc
}

func New(
	conf *config.Config,
	logger *zerolog.Logger,
) (*Server, error) {

	db, err := database.New(conf, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Initialize components
	registry := languages.NewRegistry()
	sb, err := sandbox.NewDockerSandbox(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}

	exec := executor.NewExecutor(registry, sb)
	q := queue.NewManager(100)

	// Rate limiter: 100 req/sec global, 10 req/sec per IP, 50 concurrent executions
	rl := limiter.NewRateLimiter(100, 10, 20, 50)
	rl.StartCleanup(5 * time.Minute)

	handler := api.NewHandler(q)

	mux := http.NewServeMux()

	// health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// execution endpoint with rate limiting
	mux.HandleFunc("/execute", rl.Middleware(handler.Execute))

	httpServer := &http.Server{
		Addr:         ":" + conf.Server.Port,
		Handler:      mux,
		ReadTimeout:  time.Duration(conf.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(conf.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(conf.Server.IdleTimeout) * time.Second,
	}

	// Create workers
	numWorkers := 5
	workers := make([]*worker.Worker, numWorkers)

	for i := 0; i < numWorkers; i++ {
		workers[i] = worker.NewWorker(i, exec, q, logger)
	}

	s := &Server{
		conf:        conf,
		logger:      logger,
		httpServer:  httpServer,
		db:          db,
		registry:    registry,
		sandbox:     sb,
		executor:    exec,
		queue:       q,
		workers:     workers,
		rateLimiter: rl,
	}

	return s, nil
}

func (s *Server) Start() error {
	s.logger.Info().
		Str("port", s.conf.Server.Port).
		Msg("starting HTTP server")

	// Ensure all required images are pulled
	if err := s.ensureImages(context.Background()); err != nil {
		return fmt.Errorf("failed to ensure docker images: %w", err)
	}

	// Start workers
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel
	
	for _, w := range s.workers {
		go w.Start(ctx)
	}

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failed: %w", err)
	}

	return nil
}

func (s *Server) ensureImages(ctx context.Context) error {
	langs := s.registry.List()
	uniqueImages := make(map[string]bool)
	for _, l := range langs {
		uniqueImages[l.Config.Image] = true
	}

	for img := range uniqueImages {
		if err := s.sandbox.EnsureImage(ctx, img); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")

	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	if s.db != nil {
		s.db.Close()
	}

	return nil
}
