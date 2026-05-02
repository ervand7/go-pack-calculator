package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"

	"pack-calculator/internal/calculator"
)

var DefaultPackSizes = []int{250, 500, 1000, 2000, 5000}

type Store struct {
	path   string
	logger zerolog.Logger
	mu     sync.RWMutex
}

type fileData struct {
	PackSizes []int `json:"packSizes"`
}

func NewStore(path string, logger zerolog.Logger) *Store {
	return &Store{
		path:   path,
		logger: logger.With().Str("component", "pack_size_store").Str("path", path).Logger(),
	}
}

func (s *Store) PackSizes() ([]int, error) {
	s.logger.Debug().Msg("reading pack sizes")
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.logger.Info().Ints("pack_sizes", DefaultPackSizes).Msg("pack sizes file missing, using defaults")
		return append([]int(nil), DefaultPackSizes...), nil
	}
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to read pack sizes file")
		return nil, err
	}

	var stored fileData
	if err := json.Unmarshal(data, &stored); err != nil {
		s.logger.Error().Err(err).Msg("failed to parse pack sizes file")
		return nil, err
	}

	normalized, err := calculator.NormalizePackSizes(stored.PackSizes)
	if err != nil {
		s.logger.Error().Err(err).Ints("pack_sizes", stored.PackSizes).Msg("stored pack sizes are invalid")
		return nil, err
	}

	s.logger.Debug().Ints("pack_sizes", normalized).Msg("read pack sizes")
	return normalized, nil
}

func (s *Store) SavePackSizes(packSizes []int) ([]int, error) {
	s.logger.Info().Ints("requested_pack_sizes", packSizes).Msg("saving pack sizes")
	normalized, err := calculator.NormalizePackSizes(packSizes)
	if err != nil {
		s.logger.Warn().Err(err).Ints("requested_pack_sizes", packSizes).Msg("rejected invalid pack sizes")
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		s.logger.Error().Err(err).Msg("failed to create pack sizes directory")
		return nil, err
	}

	data, err := json.MarshalIndent(fileData{PackSizes: normalized}, "", "  ")
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to encode pack sizes")
		return nil, err
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		s.logger.Error().Err(err).Msg("failed to write pack sizes file")
		return nil, err
	}

	s.logger.Info().Ints("pack_sizes", normalized).Msg("saved pack sizes")
	return append([]int(nil), normalized...), nil
}
