package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	defaultPort              = "8080"
	defaultPackSizesFile     = "data/pack_sizes.json"
	defaultLogLevel          = "debug"
	defaultShutdownTimeout   = 10 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
)

type Config struct {
	Port              string
	PackSizesFile     string
	LogLevel          zerolog.Level
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
}

func Load() (Config, error) {
	return LoadFromEnv(os.LookupEnv)
}

func LoadFromEnv(lookup func(string) (string, bool)) (Config, error) {
	logLevel, err := parseLogLevel(getEnvOrDefault(lookup, "LOG_LEVEL", defaultLogLevel))
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := parseDuration(lookup, "SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	readHeaderTimeout, err := parseDuration(lookup, "READ_HEADER_TIMEOUT", defaultReadHeaderTimeout)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:              getEnvOrDefault(lookup, "PORT", defaultPort),
		PackSizesFile:     getEnvOrDefault(lookup, "PACK_SIZES_FILE", defaultPackSizesFile),
		LogLevel:          logLevel,
		ShutdownTimeout:   shutdownTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
	}, nil
}

func getEnvOrDefault(lookup func(string) (string, bool), key string, defaultValue string) string {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func parseLogLevel(value string) (zerolog.Level, error) {
	level, err := zerolog.ParseLevel(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return zerolog.NoLevel, fmt.Errorf("invalid LOG_LEVEL %q: %w", value, err)
	}
	return level, nil
}

func parseDuration(lookup func(string) (string, bool), key string, defaultValue time.Duration) (time.Duration, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return defaultValue, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", key, value, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("invalid %s %q: duration must be positive", key, value)
	}

	return duration, nil
}
