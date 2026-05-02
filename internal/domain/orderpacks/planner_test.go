package orderpacks

import (
	"errors"
	"reflect"
	"testing"

	"github.com/rs/zerolog"
)

func TestShipmentPlannerDefaultExamples(t *testing.T) {
	planner := NewShipmentPlanner(zerolog.Nop())
	packSizes := []int{250, 500, 1000, 2000, 5000}

	tests := []struct {
		name         string
		itemsOrdered int
		itemsShipped int
		totalPacks   int
		packs        map[int]int
	}{
		{
			name:         "single item uses smallest pack",
			itemsOrdered: 1,
			itemsShipped: 250,
			totalPacks:   1,
			packs:        map[int]int{250: 1},
		},
		{
			name:         "exact smallest pack",
			itemsOrdered: 250,
			itemsShipped: 250,
			totalPacks:   1,
			packs:        map[int]int{250: 1},
		},
		{
			name:         "minimizes pack count after shipped items",
			itemsOrdered: 251,
			itemsShipped: 500,
			totalPacks:   1,
			packs:        map[int]int{500: 1},
		},
		{
			name:         "minimizes shipped items before pack count",
			itemsOrdered: 501,
			itemsShipped: 750,
			totalPacks:   2,
			packs:        map[int]int{250: 1, 500: 1},
		},
		{
			name:         "large example from brief",
			itemsOrdered: 12001,
			itemsShipped: 12250,
			totalPacks:   4,
			packs:        map[int]int{250: 1, 2000: 1, 5000: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := planner.Plan(tt.itemsOrdered, packSizes)
			if err != nil {
				t.Fatalf("Plan returned error: %v", err)
			}

			if plan.ItemsShipped != tt.itemsShipped {
				t.Fatalf("ItemsShipped = %d, want %d", plan.ItemsShipped, tt.itemsShipped)
			}
			if plan.TotalPacks != tt.totalPacks {
				t.Fatalf("TotalPacks = %d, want %d", plan.TotalPacks, tt.totalPacks)
			}
			if got := allocationMap(plan.Packs); !reflect.DeepEqual(got, tt.packs) {
				t.Fatalf("Packs = %#v, want %#v", got, tt.packs)
			}
		})
	}
}

func TestShipmentPlannerCustomLargeEdgeCase(t *testing.T) {
	planner := NewShipmentPlanner(zerolog.Nop())

	plan, err := planner.Plan(500000, []int{23, 31, 53})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	want := map[int]int{23: 2, 31: 7, 53: 9429}
	if plan.ItemsShipped != 500000 {
		t.Fatalf("ItemsShipped = %d, want 500000", plan.ItemsShipped)
	}
	if got := allocationMap(plan.Packs); !reflect.DeepEqual(got, want) {
		t.Fatalf("Packs = %#v, want %#v", got, want)
	}
}

func TestShipmentPlannerRulePriorityScenarios(t *testing.T) {
	planner := NewShipmentPlanner(zerolog.Nop())

	tests := []struct {
		name         string
		itemsOrdered int
		packSizes    []int
		itemsShipped int
		totalPacks   int
		packs        map[int]int
	}{
		{
			name:         "exact larger pack beats many smaller packs",
			itemsOrdered: 1000,
			packSizes:    []int{250, 500, 1000},
			itemsShipped: 1000,
			totalPacks:   1,
			packs:        map[int]int{1000: 1},
		},
		{
			name:         "minimum shipped items beats fewer packs",
			itemsOrdered: 11,
			packSizes:    []int{4, 5, 10},
			itemsShipped: 12,
			totalPacks:   3,
			packs:        map[int]int{4: 3},
		},
		{
			name:         "fewest packs wins for same shipped total",
			itemsOrdered: 6,
			packSizes:    []int{2, 3, 5},
			itemsShipped: 6,
			totalPacks:   2,
			packs:        map[int]int{3: 2},
		},
		{
			name:         "normalizes unsorted duplicate pack sizes",
			itemsOrdered: 9,
			packSizes:    []int{10, 5, 5, 4},
			itemsShipped: 9,
			totalPacks:   2,
			packs:        map[int]int{4: 1, 5: 1},
		},
		{
			name:         "single pack size rounds up to next reachable total",
			itemsOrdered: 15,
			packSizes:    []int{7},
			itemsShipped: 21,
			totalPacks:   3,
			packs:        map[int]int{7: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := planner.Plan(tt.itemsOrdered, tt.packSizes)
			if err != nil {
				t.Fatalf("Plan returned error: %v", err)
			}

			assertPlan(t, plan, tt.itemsOrdered, tt.itemsShipped, tt.totalPacks, tt.packs)
		})
	}
}

func TestShipmentPlannerZeroItems(t *testing.T) {
	planner := NewShipmentPlanner(zerolog.Nop())

	plan, err := planner.Plan(0, []int{250, 500})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	if plan.ItemsShipped != 0 || plan.TotalPacks != 0 || len(plan.Packs) != 0 {
		t.Fatalf("plan = %#v, want empty zero allocation", plan)
	}
}

func TestShipmentPlannerValidation(t *testing.T) {
	planner := NewShipmentPlanner(zerolog.Nop())

	tests := []struct {
		name         string
		itemsOrdered int
		packSizes    []int
		wantErr      error
	}{
		{name: "negative order", itemsOrdered: -1, packSizes: []int{250}, wantErr: ErrInvalidItemCount},
		{name: "empty packs", itemsOrdered: 1, packSizes: []int{}, wantErr: ErrNoPackSizes},
		{name: "zero pack", itemsOrdered: 1, packSizes: []int{0}, wantErr: ErrInvalidPackSize},
		{name: "negative pack", itemsOrdered: 1, packSizes: []int{-5}, wantErr: ErrInvalidPackSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := planner.Plan(tt.itemsOrdered, tt.packSizes)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizePackSizesSortsAndDeduplicates(t *testing.T) {
	got, err := NormalizePackSizes([]int{500, 250, 500, 1000})
	if err != nil {
		t.Fatalf("NormalizePackSizes returned error: %v", err)
	}

	want := []int{250, 500, 1000}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizePackSizes = %v, want %v", got, want)
	}
}

func allocationMap(packs []Allocation) map[int]int {
	result := make(map[int]int, len(packs))
	for _, pack := range packs {
		result[pack.PackSize] = pack.Quantity
	}
	return result
}

func assertPlan(t *testing.T, plan ShipmentPlan, itemsOrdered int, itemsShipped int, totalPacks int, packs map[int]int) {
	t.Helper()

	if plan.ItemsOrdered != itemsOrdered {
		t.Fatalf("ItemsOrdered = %d, want %d", plan.ItemsOrdered, itemsOrdered)
	}
	if plan.ItemsShipped != itemsShipped {
		t.Fatalf("ItemsShipped = %d, want %d", plan.ItemsShipped, itemsShipped)
	}
	if plan.TotalPacks != totalPacks {
		t.Fatalf("TotalPacks = %d, want %d", plan.TotalPacks, totalPacks)
	}
	if got := allocationMap(plan.Packs); !reflect.DeepEqual(got, packs) {
		t.Fatalf("Packs = %#v, want %#v", got, packs)
	}
}
