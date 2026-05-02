package config

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rs/zerolog"
)

func TestStoreReturnsDefaultsWhenFileDoesNotExist(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "pack_sizes.json"), zerolog.Nop())

	got, err := store.PackSizes()
	if err != nil {
		t.Fatalf("PackSizes returned error: %v", err)
	}

	if !reflect.DeepEqual(got, DefaultPackSizes) {
		t.Fatalf("PackSizes = %v, want %v", got, DefaultPackSizes)
	}
}

func TestStorePersistsNormalizedPackSizes(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nested", "pack_sizes.json"), zerolog.Nop())

	saved, err := store.SavePackSizes([]int{53, 23, 31, 53})
	if err != nil {
		t.Fatalf("SavePackSizes returned error: %v", err)
	}

	want := []int{23, 31, 53}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("saved = %v, want %v", saved, want)
	}

	got, err := store.PackSizes()
	if err != nil {
		t.Fatalf("PackSizes returned error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PackSizes = %v, want %v", got, want)
	}
}
