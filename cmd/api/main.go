package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/itstheanurag/executioner/internal/config"

	"github.com/itstheanurag/executioner/internal/server"
	"github.com/rs/zerolog"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	conf, err := config.LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load config")
	}

	srv, err := server.New(conf, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
	}

	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal().Err(err).Msg("server crashed")
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		logger.Error().Err(err).Msg("graceful shutdown failed")
	}
}
