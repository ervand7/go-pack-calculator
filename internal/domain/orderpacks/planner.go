package orderpacks

import (
	"fmt"
	"math"

	"github.com/rs/zerolog"
)

type ShipmentPlanner struct {
	logger zerolog.Logger
}

func NewShipmentPlanner(logger zerolog.Logger) ShipmentPlanner {
	return ShipmentPlanner{
		logger: logger.With().Str("component", "shipment_planner").Logger(),
	}
}

// Plan returns the allocation that ships the fewest possible items while
// fulfilling the order, then uses the fewest packs for that shipped total.
func (p ShipmentPlanner) Plan(itemsOrdered int, packSizes []int) (ShipmentPlan, error) {
	p.logger.Debug().
		Int("items_ordered", itemsOrdered).
		Ints("pack_sizes", packSizes).
		Msg("starting shipment planning")

	if itemsOrdered < 0 {
		return ShipmentPlan{}, fmt.Errorf("validate item count: %w", ErrInvalidItemCount)
	}

	sizes, err := NormalizePackSizes(packSizes)
	if err != nil {
		return ShipmentPlan{}, fmt.Errorf("normalize pack sizes: %w", err)
	}
	p.logger.Debug().Ints("normalized_pack_sizes", sizes).Msg("normalized pack sizes")

	if itemsOrdered == 0 {
		p.logger.Info().Msg("zero item order requires no packs")
		return ShipmentPlan{
			ItemsOrdered: 0,
			ItemsShipped: 0,
			TotalPacks:   0,
			Packs:        []Allocation{},
		}, nil
	}

	maxPack := sizes[len(sizes)-1]
	limit := itemsOrdered + maxPack - 1
	p.logger.Debug().
		Int("max_pack_size", maxPack).
		Int("search_limit", limit).
		Msg("prepared dynamic programming search range")

	minPacks := make([]int, limit+1)
	previousPack := make([]int, limit+1)

	for i := 1; i <= limit; i++ {
		minPacks[i] = math.MaxInt32
	}

	for total := 1; total <= limit; total++ {
		for _, size := range sizes {
			if size > total {
				break
			}
			if minPacks[total-size] == math.MaxInt32 {
				continue
			}
			candidate := minPacks[total-size] + 1
			if candidate < minPacks[total] {
				minPacks[total] = candidate
				previousPack[total] = size
			}
		}
	}

	itemsShipped := -1
	for total := itemsOrdered; total <= limit; total++ {
		if minPacks[total] != math.MaxInt32 {
			itemsShipped = total
			break
		}
	}
	if itemsShipped == -1 {
		p.logger.Error().
			Int("items_ordered", itemsOrdered).
			Int("search_limit", limit).
			Msg("no reachable shipped total found")
		return ShipmentPlan{}, fmt.Errorf("find reachable shipped total: %w", ErrNoPackSizes)
	}

	counts := map[int]int{}
	for total := itemsShipped; total > 0; {
		size := previousPack[total]
		counts[size]++
		total -= size
	}

	packs := make([]Allocation, 0, len(counts))
	for _, size := range sizes {
		if quantity := counts[size]; quantity > 0 {
			packs = append(packs, Allocation{PackSize: size, Quantity: quantity})
		}
	}

	plan := ShipmentPlan{
		ItemsOrdered: itemsOrdered,
		ItemsShipped: itemsShipped,
		TotalPacks:   minPacks[itemsShipped],
		Packs:        packs,
	}

	p.logger.Info().
		Int("items_ordered", plan.ItemsOrdered).
		Int("items_shipped", plan.ItemsShipped).
		Int("extra_items", plan.ItemsShipped-plan.ItemsOrdered).
		Int("total_packs", plan.TotalPacks).
		Interface("packs", plan.Packs).
		Msg("completed shipment planning")

	return plan, nil
}
