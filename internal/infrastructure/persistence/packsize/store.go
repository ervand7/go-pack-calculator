package packsize

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"

	domain "pack-calculator/internal/domain/orderpacks"
)

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

// List returns configured pack sizes, falling back to domain defaults when no
// file exists.
func (s *Store) List(ctx context.Context) ([]int, error) {
	if err := ctx.Err(); err != nil {
		s.logger.Warn().Err(err).Msg("context canceled before reading pack sizes")
		return nil, err
	}

	s.logger.Debug().Msg("reading pack sizes")
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.logger.Info().Ints("pack_sizes", domain.DefaultPackSizes).Msg("pack sizes file missing, using defaults")
		return append([]int(nil), domain.DefaultPackSizes...), nil
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

	normalized, err := domain.NormalizePackSizes(stored.PackSizes)
	if err != nil {
		s.logger.Error().Err(err).Ints("pack_sizes", stored.PackSizes).Msg("stored pack sizes are invalid")
		return nil, err
	}

	s.logger.Debug().Ints("pack_sizes", normalized).Msg("read pack sizes")
	return normalized, nil
}

// Save normalizes and persists pack sizes to the configured JSON file.
func (s *Store) Save(ctx context.Context, packSizes []int) ([]int, error) {
	if err := ctx.Err(); err != nil {
		s.logger.Warn().Err(err).Msg("context canceled before saving pack sizes")
		return nil, err
	}

	s.logger.Info().Ints("requested_pack_sizes", packSizes).Msg("saving pack sizes")
	normalized, err := domain.NormalizePackSizes(packSizes)
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
