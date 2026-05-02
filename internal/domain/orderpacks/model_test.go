package orderpacks

import (
	"errors"
	"reflect"
	"testing"
)

func TestDefaultPackSizes(t *testing.T) {
	want := []int{250, 500, 1000, 2000, 5000}

	if !reflect.DeepEqual(DefaultPackSizes, want) {
		t.Fatalf("DefaultPackSizes = %v, want %v", DefaultPackSizes, want)
	}
}

func TestNormalizePackSizesModelRules(t *testing.T) {
	tests := []struct {
		name      string
		packSizes []int
		want      []int
	}{
		{
			name:      "keeps already sorted unique sizes",
			packSizes: []int{23, 31, 53},
			want:      []int{23, 31, 53},
		},
		{
			name:      "sorts ascending",
			packSizes: []int{1000, 250, 500},
			want:      []int{250, 500, 1000},
		},
		{
			name:      "deduplicates repeated sizes",
			packSizes: []int{500, 250, 500, 1000, 250},
			want:      []int{250, 500, 1000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePackSizes(tt.packSizes)
			if err != nil {
				t.Fatalf("NormalizePackSizes returned error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("NormalizePackSizes = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizePackSizesValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		packSizes []int
		wantErr   error
	}{
		{
			name:      "empty list",
			packSizes: []int{},
			wantErr:   ErrNoPackSizes,
		},
		{
			name:      "zero size",
			packSizes: []int{250, 0},
			wantErr:   ErrInvalidPackSize,
		},
		{
			name:      "negative size",
			packSizes: []int{250, -500},
			wantErr:   ErrInvalidPackSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizePackSizes(tt.packSizes)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
