package main

import (
	"context"
	_ "embed"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"pack-calculator/internal/config"
	"pack-calculator/internal/httpapi"
)

//go:embed static/index.html
var indexHTML string

func main() {
	packSizesFile := getEnvOrDefault("PACK_SIZES_FILE", "data/pack_sizes.json")
	port := getEnvOrDefault("PORT", "8080")
	shutdownTimeout := 10 * time.Second

	logger := newLogger()
	logger.Info().
		Str("pack_sizes_file", packSizesFile).
		Str("port", port).
		Dur("shutdown_timeout", shutdownTimeout).
		Msg("starting pack calculator")

	store := config.NewStore(packSizesFile, logger)
	router := httpapi.NewRouter(store, logger)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			logger.Warn().Str("path", r.URL.Path).Msg("static route not found")
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
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
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

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
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

func getEnvOrDefault(key, defaultVal string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal
	}

	return value
}

func newLogger() zerolog.Logger {
	levelValue := strings.ToLower(getEnvOrDefault("LOG_LEVEL", "debug"))
	level, err := zerolog.ParseLevel(levelValue)
	if err != nil {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", "pack-calculator").
		Logger()
}
