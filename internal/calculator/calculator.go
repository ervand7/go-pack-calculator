package calculator

import (
	"math"
	"sort"

	"github.com/rs/zerolog"
)

type Allocation struct {
	PackSize int `json:"packSize"`
	Quantity int `json:"quantity"`
}

type Result struct {
	ItemsOrdered int          `json:"itemsOrdered"`
	ItemsShipped int          `json:"itemsShipped"`
	TotalPacks   int          `json:"totalPacks"`
	Packs        []Allocation `json:"packs"`
}

// Calculate returns the pack allocation that ships the fewest possible items
// while fulfilling the order, then uses the fewest packs for that shipped total.
func Calculate(itemsOrdered int, packSizes []int) (Result, error) {
	return CalculateWithLogger(itemsOrdered, packSizes, zerolog.Nop())
}

// CalculateWithLogger is Calculate with detailed structured logs for callers
// that want visibility into validation, search bounds, and chosen allocation.
func CalculateWithLogger(itemsOrdered int, packSizes []int, logger zerolog.Logger) (Result, error) {
	logger.Debug().
		Int("items_ordered", itemsOrdered).
		Ints("pack_sizes", packSizes).
		Msg("starting pack calculation")

	if itemsOrdered < 0 {
		logger.Warn().Int("items_ordered", itemsOrdered).Err(ErrInvalidItemCount).Msg("invalid item count")
		return Result{}, ErrInvalidItemCount
	}

	sizes, err := NormalizePackSizes(packSizes)
	if err != nil {
		logger.Warn().Ints("pack_sizes", packSizes).Err(err).Msg("invalid pack sizes")
		return Result{}, err
	}
	logger.Debug().Ints("normalized_pack_sizes", sizes).Msg("normalized pack sizes")

	if itemsOrdered == 0 {
		logger.Info().Msg("zero item order requires no packs")
		return Result{
			ItemsOrdered: 0,
			ItemsShipped: 0,
			TotalPacks:   0,
			Packs:        []Allocation{},
		}, nil
	}

	maxPack := sizes[len(sizes)-1]
	limit := itemsOrdered + maxPack - 1
	logger.Debug().
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
		logger.Error().
			Int("items_ordered", itemsOrdered).
			Int("search_limit", limit).
			Msg("no reachable shipped total found")
		return Result{}, ErrNoPackSizes
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

	result := Result{
		ItemsOrdered: itemsOrdered,
		ItemsShipped: itemsShipped,
		TotalPacks:   minPacks[itemsShipped],
		Packs:        packs,
	}

	logger.Info().
		Int("items_ordered", result.ItemsOrdered).
		Int("items_shipped", result.ItemsShipped).
		Int("extra_items", result.ItemsShipped-result.ItemsOrdered).
		Int("total_packs", result.TotalPacks).
		Interface("packs", result.Packs).
		Msg("completed pack calculation")

	return result, nil
}

func NormalizePackSizes(packSizes []int) ([]int, error) {
	if len(packSizes) == 0 {
		return nil, ErrNoPackSizes
	}

	unique := make(map[int]struct{}, len(packSizes))
	for _, size := range packSizes {
		if size <= 0 {
			return nil, ErrInvalidPackSize
		}
		unique[size] = struct{}{}
	}

	sizes := make([]int, 0, len(unique))
	for size := range unique {
		sizes = append(sizes, size)
	}
	sort.Ints(sizes)

	return sizes, nil
}
