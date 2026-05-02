package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rs/zerolog"

	"pack-calculator/internal/calculator"
	"pack-calculator/internal/config"
)

func TestPackSizesCanBeChangedAndUsedForCalculation(t *testing.T) {
	store := config.NewStore(filepath.Join(t.TempDir(), "pack_sizes.json"), zerolog.Nop())
	router := NewRouter(store, zerolog.Nop())

	putBody := bytes.NewBufferString(`{"packSizes":[23,31,53]}`)
	putRequest := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", putBody)
	putResponse := httptest.NewRecorder()
	router.ServeHTTP(putResponse, putRequest)

	if putResponse.Code != http.StatusOK {
		t.Fatalf("PUT /api/pack-sizes status = %d, body = %s", putResponse.Code, putResponse.Body.String())
	}

	calculateBody := bytes.NewBufferString(`{"items":500000}`)
	calculateRequest := httptest.NewRequest(http.MethodPost, "/api/calculate", calculateBody)
	calculateResponse := httptest.NewRecorder()
	router.ServeHTTP(calculateResponse, calculateRequest)

	if calculateResponse.Code != http.StatusOK {
		t.Fatalf("POST /api/calculate status = %d, body = %s", calculateResponse.Code, calculateResponse.Body.String())
	}

	var result calculator.Result
	if err := json.NewDecoder(calculateResponse.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := map[int]int{23: 2, 31: 7, 53: 9429}
	if result.ItemsShipped != 500000 {
		t.Fatalf("ItemsShipped = %d, want 500000", result.ItemsShipped)
	}
	if got := allocationMap(result.Packs); !reflect.DeepEqual(got, want) {
		t.Fatalf("Packs = %#v, want %#v", got, want)
	}
}

func TestCalculateRejectsInvalidItems(t *testing.T) {
	store := config.NewStore(filepath.Join(t.TempDir(), "pack_sizes.json"), zerolog.Nop())
	router := NewRouter(store, zerolog.Nop())

	request := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewBufferString(`{"items":-1}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestPackSizesRejectInvalidSizes(t *testing.T) {
	store := config.NewStore(filepath.Join(t.TempDir(), "pack_sizes.json"), zerolog.Nop())
	router := NewRouter(store, zerolog.Nop())

	request := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", bytes.NewBufferString(`{"packSizes":[250,0]}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func allocationMap(packs []calculator.Allocation) map[int]int {
	result := make(map[int]int, len(packs))
	for _, pack := range packs {
		result[pack.PackSize] = pack.Quantity
	}
	return result
}
