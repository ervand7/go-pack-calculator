package orderpacks

import "sort"

var DefaultPackSizes = []int{250, 500, 1000, 2000, 5000}

type Allocation struct {
	PackSize int
	Quantity int
}

type ShipmentPlan struct {
	ItemsOrdered int
	ItemsShipped int
	TotalPacks   int
	Packs        []Allocation
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
