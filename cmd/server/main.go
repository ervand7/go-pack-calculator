package main

import (
	"context"
	_ "embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	app "pack-calculator/internal/application/orderpacks"
	domain "pack-calculator/internal/domain/orderpacks"
	"pack-calculator/internal/infrastructure/config"
	"pack-calculator/internal/infrastructure/persistence/packsize"
	"pack-calculator/internal/interfaces/httpapi"
)

//go:embed static/index.html
var indexHTML string

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := newLogger(cfg.LogLevel)
	logger.Info().
		Str("pack_sizes_file", cfg.PackSizesFile).
		Str("port", cfg.Port).
		Str("log_level", cfg.LogLevel.String()).
		Dur("shutdown_timeout", cfg.ShutdownTimeout).
		Dur("read_header_timeout", cfg.ReadHeaderTimeout).
		Msg("starting pack calculator")

	store := packsize.NewStore(cfg.PackSizesFile, logger)
	planner := domain.NewShipmentPlanner(logger)
	service := app.NewService(store, planner, logger)
	router := httpapi.NewRouter(service, logger)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			logger.Error().Str("path", r.URL.Path).Msg("static route not found")
			http.NotFound(w, r)
			return
		}
		logger.Debug().Str("remote_addr", r.RemoteAddr).Msg("serving UI")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte(indexHTML)); err != nil {
			logger.Error().Err(err).Msg("failed to write UI response")
		}
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info().Str("addr", server.Addr).Msg("HTTP server listening")
		errCh <- server.ListenAndServe()
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopCh)

	select {
	case signalValue := <-stopCh:
		logger.Info().Str("signal", signalValue.String()).Msg("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	logger.Info().Msg("graceful shutdown started")
	if err := server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("graceful shutdown failed, forcing close")
		if closeErr := server.Close(); closeErr != nil {
			logger.Fatal().Err(closeErr).Msg("forced server close failed")
		}
		os.Exit(1)
	}

	logger.Info().Msg("graceful shutdown completed")
}

func newLogger(level zerolog.Level) zerolog.Logger {
	zerolog.SetGlobalLevel(level)

	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", "pack-calculator").
		Logger()
}
