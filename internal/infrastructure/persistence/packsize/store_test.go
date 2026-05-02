package packsize

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rs/zerolog"

	domain "pack-calculator/internal/domain/orderpacks"
)

func TestStoreReturnsDefaultsWhenFileDoesNotExist(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "pack_sizes.json"), zerolog.Nop())

	got, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if !reflect.DeepEqual(got, domain.DefaultPackSizes) {
		t.Fatalf("List = %v, want %v", got, domain.DefaultPackSizes)
	}
}

func TestStorePersistsNormalizedPackSizes(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nested", "pack_sizes.json"), zerolog.Nop())

	saved, err := store.Save(context.Background(), []int{53, 23, 31, 53})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	want := []int{23, 31, 53}
	if !reflect.DeepEqual(saved, want) {
		t.Fatalf("saved = %v, want %v", saved, want)
	}

	got, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List = %v, want %v", got, want)
	}
}

func TestStoreWritesExpectedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pack_sizes.json")
	store := NewStore(path, zerolog.Nop())

	_, err := store.Save(context.Background(), []int{1000, 250, 500})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	want := "{\n  \"packSizes\": [\n    250,\n    500,\n    1000\n  ]\n}"
	if string(data) != want {
		t.Fatalf("stored JSON = %q, want %q", string(data), want)
	}
}

func TestStoreListRejectsMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pack_sizes.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	store := NewStore(path, zerolog.Nop())

	_, err := store.List(context.Background())
	if err == nil {
		t.Fatal("List returned nil error, want malformed JSON error")
	}
}

func TestStoreListRejectsInvalidStoredPackSizes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pack_sizes.json")
	if err := os.WriteFile(path, []byte(`{"packSizes":[250,0]}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	store := NewStore(path, zerolog.Nop())

	_, err := store.List(context.Background())
	if !errors.Is(err, domain.ErrInvalidPackSize) {
		t.Fatalf("error = %v, want %v", err, domain.ErrInvalidPackSize)
	}
}

func TestStoreSaveRejectsInvalidPackSizes(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "pack_sizes.json"), zerolog.Nop())

	_, err := store.Save(context.Background(), []int{250, -1})
	if !errors.Is(err, domain.ErrInvalidPackSize) {
		t.Fatalf("error = %v, want %v", err, domain.ErrInvalidPackSize)
	}
}

func TestStoreListReturnsContextErrorWhenCanceled(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "pack_sizes.json"), zerolog.Nop())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.List(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want %v", err, context.Canceled)
	}
}

func TestStoreSaveReturnsContextErrorWhenCanceled(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "pack_sizes.json"), zerolog.Nop())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Save(ctx, []int{250})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want %v", err, context.Canceled)
	}
}
