package calculator

import (
	"errors"
	"reflect"
	"testing"
)

func TestCalculateDefaultExamples(t *testing.T) {
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
			result, err := Calculate(tt.itemsOrdered, packSizes)
			if err != nil {
				t.Fatalf("Calculate returned error: %v", err)
			}

			if result.ItemsShipped != tt.itemsShipped {
				t.Fatalf("ItemsShipped = %d, want %d", result.ItemsShipped, tt.itemsShipped)
			}
			if result.TotalPacks != tt.totalPacks {
				t.Fatalf("TotalPacks = %d, want %d", result.TotalPacks, tt.totalPacks)
			}
			if got := allocationMap(result.Packs); !reflect.DeepEqual(got, tt.packs) {
				t.Fatalf("Packs = %#v, want %#v", got, tt.packs)
			}
		})
	}
}

func TestCalculateCustomLargeEdgeCase(t *testing.T) {
	result, err := Calculate(500000, []int{23, 31, 53})
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}

	want := map[int]int{23: 2, 31: 7, 53: 9429}
	if result.ItemsShipped != 500000 {
		t.Fatalf("ItemsShipped = %d, want 500000", result.ItemsShipped)
	}
	if got := allocationMap(result.Packs); !reflect.DeepEqual(got, want) {
		t.Fatalf("Packs = %#v, want %#v", got, want)
	}
}

func TestCalculateZeroItems(t *testing.T) {
	result, err := Calculate(0, []int{250, 500})
	if err != nil {
		t.Fatalf("Calculate returned error: %v", err)
	}

	if result.ItemsShipped != 0 || result.TotalPacks != 0 || len(result.Packs) != 0 {
		t.Fatalf("result = %#v, want empty zero allocation", result)
	}
}

func TestCalculateValidation(t *testing.T) {
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
			_, err := Calculate(tt.itemsOrdered, tt.packSizes)
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
