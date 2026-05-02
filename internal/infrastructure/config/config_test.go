package config

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestLoadFromEnvUsesDefaults(t *testing.T) {
	cfg, err := LoadFromEnv(emptyEnv)
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if cfg.Port != defaultPort {
		t.Fatalf("Port = %q, want %q", cfg.Port, defaultPort)
	}
	if cfg.PackSizesFile != defaultPackSizesFile {
		t.Fatalf("PackSizesFile = %q, want %q", cfg.PackSizesFile, defaultPackSizesFile)
	}
	if cfg.LogLevel != zerolog.DebugLevel {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, zerolog.DebugLevel)
	}
	if cfg.ShutdownTimeout != defaultShutdownTimeout {
		t.Fatalf("ShutdownTimeout = %v, want %v", cfg.ShutdownTimeout, defaultShutdownTimeout)
	}
	if cfg.ReadHeaderTimeout != defaultReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %v, want %v", cfg.ReadHeaderTimeout, defaultReadHeaderTimeout)
	}
}

func TestLoadFromEnvUsesProvidedValues(t *testing.T) {
	env := map[string]string{
		"PORT":                "9090",
		"PACK_SIZES_FILE":     "/tmp/pack_sizes.json",
		"LOG_LEVEL":           "warn",
		"SHUTDOWN_TIMEOUT":    "15s",
		"READ_HEADER_TIMEOUT": "3s",
	}

	cfg, err := LoadFromEnv(mapLookup(env))
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.PackSizesFile != "/tmp/pack_sizes.json" {
		t.Fatalf("PackSizesFile = %q, want /tmp/pack_sizes.json", cfg.PackSizesFile)
	}
	if cfg.LogLevel != zerolog.WarnLevel {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, zerolog.WarnLevel)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Fatalf("ShutdownTimeout = %v, want 15s", cfg.ShutdownTimeout)
	}
	if cfg.ReadHeaderTimeout != 3*time.Second {
		t.Fatalf("ReadHeaderTimeout = %v, want 3s", cfg.ReadHeaderTimeout)
	}
}

func TestLoadFromEnvRejectsInvalidLogLevel(t *testing.T) {
	_, err := LoadFromEnv(mapLookup(map[string]string{"LOG_LEVEL": "chatty"}))
	if err == nil {
		t.Fatal("LoadFromEnv returned nil error, want invalid log level error")
	}
}

func TestLoadFromEnvRejectsInvalidDuration(t *testing.T) {
	_, err := LoadFromEnv(mapLookup(map[string]string{"SHUTDOWN_TIMEOUT": "soon"}))
	if err == nil {
		t.Fatal("LoadFromEnv returned nil error, want invalid duration error")
	}
}

func TestLoadFromEnvRejectsNonPositiveDuration(t *testing.T) {
	_, err := LoadFromEnv(mapLookup(map[string]string{"READ_HEADER_TIMEOUT": "0s"}))
	if err == nil {
		t.Fatal("LoadFromEnv returned nil error, want non-positive duration error")
	}
}

func emptyEnv(string) (string, bool) {
	return "", false
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
