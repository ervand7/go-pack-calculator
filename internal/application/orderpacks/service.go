package orderpacks

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	domain "pack-calculator/internal/domain/orderpacks"
)

type PackSizeRepository interface {
	List(ctx context.Context) ([]int, error)
	Save(ctx context.Context, packSizes []int) ([]int, error)
}

type Service struct {
	repository PackSizeRepository
	planner    domain.ShipmentPlanner
	logger     zerolog.Logger
}

func NewService(repository PackSizeRepository, planner domain.ShipmentPlanner, logger zerolog.Logger) *Service {
	return &Service{
		repository: repository,
		planner:    planner,
		logger:     logger.With().Str("component", "orderpacks_application").Logger(),
	}
}

func (s *Service) GetPackSizes(ctx context.Context) ([]int, error) {
	s.logger.Debug().Msg("getting configured pack sizes")

	packSizes, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("get configured pack sizes: %w", err)
	}

	s.logger.Info().Ints("pack_sizes", packSizes).Msg("got configured pack sizes")
	return packSizes, nil
}

func (s *Service) UpdatePackSizes(ctx context.Context, packSizes []int) ([]int, error) {
	s.logger.Info().Ints("requested_pack_sizes", packSizes).Msg("updating configured pack sizes")

	normalized, err := domain.NormalizePackSizes(packSizes)
	if err != nil {
		return nil, fmt.Errorf("normalize requested pack sizes: %w", err)
	}

	saved, err := s.repository.Save(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("save configured pack sizes: %w", err)
	}

	s.logger.Info().Ints("pack_sizes", saved).Msg("updated configured pack sizes")
	return saved, nil
}

func (s *Service) Calculate(ctx context.Context, itemsOrdered int) (domain.ShipmentPlan, error) {
	s.logger.Info().Int("items_ordered", itemsOrdered).Msg("calculating shipment plan")

	packSizes, err := s.repository.List(ctx)
	if err != nil {
		return domain.ShipmentPlan{}, fmt.Errorf("get pack sizes for shipment calculation: %w", err)
	}

	plan, err := s.planner.Plan(itemsOrdered, packSizes)
	if err != nil {
		return domain.ShipmentPlan{}, fmt.Errorf("calculate shipment plan: %w", err)
	}

	s.logger.Info().
		Int("items_ordered", plan.ItemsOrdered).
		Int("items_shipped", plan.ItemsShipped).
		Int("total_packs", plan.TotalPacks).
		Msg("calculated shipment plan")

	return plan, nil
}
