package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rs/zerolog"

	app "pack-calculator/internal/application/orderpacks"
	domain "pack-calculator/internal/domain/orderpacks"
	"pack-calculator/internal/infrastructure/persistence/packsize"
)

func TestPackSizesCanBeChangedAndUsedForCalculation(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service, zerolog.Nop())

	putBody := bytes.NewBufferString(`{"packSizes":[23,31,53]}`)
	putRequest := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", putBody)
	putResponse := httptest.NewRecorder()
	router.ServeHTTP(putResponse, putRequest)

	if putResponse.Code != http.StatusOK {
		t.Fatalf("PUT /api/pack-sizes status = %d, body = %s", putResponse.Code, putResponse.Body.String())
	}

	calculateBody := bytes.NewBufferString(`{"items":500000}`)
	calculateRequest := httptest.NewRequest(http.MethodPost, "/api/calculate", calculateBody)
	calculateRecorder := httptest.NewRecorder()
	router.ServeHTTP(calculateRecorder, calculateRequest)

	if calculateRecorder.Code != http.StatusOK {
		t.Fatalf("POST /api/calculate status = %d, body = %s", calculateRecorder.Code, calculateRecorder.Body.String())
	}

	var result calculateResponse
	if err := json.NewDecoder(calculateRecorder.Body).Decode(&result); err != nil {
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

func TestGetPackSizesReturnsDefaults(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service, zerolog.Nop())

	request := httptest.NewRequest(http.MethodGet, "/api/pack-sizes", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var result packSizesRequest
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !reflect.DeepEqual(result.PackSizes, domain.DefaultPackSizes) {
		t.Fatalf("PackSizes = %v, want %v", result.PackSizes, domain.DefaultPackSizes)
	}
}

func TestPackSizesCanBeUpdatedWithPost(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service, zerolog.Nop())

	request := httptest.NewRequest(http.MethodPost, "/api/pack-sizes", bytes.NewBufferString(`{"packSizes":[100,50,100]}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var result packSizesRequest
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	want := []int{50, 100}
	if !reflect.DeepEqual(result.PackSizes, want) {
		t.Fatalf("PackSizes = %v, want %v", result.PackSizes, want)
	}
}

func TestCalculateRejectsInvalidItems(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service, zerolog.Nop())

	request := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewBufferString(`{"items":-1}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestPackSizesRejectInvalidSizes(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service, zerolog.Nop())

	request := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", bytes.NewBufferString(`{"packSizes":[250,0]}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCalculateUsesItemsOrderedWhenProvided(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service, zerolog.Nop())

	request := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewBufferString(`{"items":1,"itemsOrdered":251}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var result calculateResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.ItemsOrdered != 251 {
		t.Fatalf("ItemsOrdered = %d, want 251", result.ItemsOrdered)
	}
	if result.ItemsShipped != 500 {
		t.Fatalf("ItemsShipped = %d, want 500", result.ItemsShipped)
	}
}

func TestCalculateRejectsMalformedJSON(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service, zerolog.Nop())

	request := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewBufferString(`{bad-json`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusBadRequest, "invalid calculate JSON request")
}

func TestPackSizesRejectMalformedJSON(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service, zerolog.Nop())

	request := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", bytes.NewBufferString(`{bad-json`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusBadRequest, "invalid pack sizes JSON request")
}

func TestCalculateRejectsUnsupportedMethod(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service, zerolog.Nop())

	request := httptest.NewRequest(http.MethodGet, "/api/calculate", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusMethodNotAllowed, "method not allowed")
}

func TestPackSizesRejectUnsupportedMethod(t *testing.T) {
	service := newTestService(t)
	router := NewRouter(service, zerolog.Nop())

	request := httptest.NewRequest(http.MethodDelete, "/api/pack-sizes", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusMethodNotAllowed, "method not allowed for pack sizes endpoint")
}

func TestToCalculateResponseMapsDomainPlan(t *testing.T) {
	got := toCalculateResponse(domain.ShipmentPlan{
		ItemsOrdered: 42,
		ItemsShipped: 50,
		TotalPacks:   2,
		Packs: []domain.Allocation{
			{PackSize: 20, Quantity: 1},
			{PackSize: 30, Quantity: 1},
		},
	})

	want := calculateResponse{
		ItemsOrdered: 42,
		ItemsShipped: 50,
		TotalPacks:   2,
		Packs: []allocation{
			{PackSize: 20, Quantity: 1},
			{PackSize: 30, Quantity: 1},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("toCalculateResponse = %#v, want %#v", got, want)
	}
}

func TestWriteValidationErrorMapsDomainErrorsToBadRequest(t *testing.T) {
	response := httptest.NewRecorder()

	writeValidationError(response, domain.ErrInvalidPackSize, zerolog.Nop())

	assertErrorResponse(t, response, http.StatusBadRequest, domain.ErrInvalidPackSize.Error())
}

func TestWriteValidationErrorMapsUnexpectedErrorsToInternalServerError(t *testing.T) {
	response := httptest.NewRecorder()

	writeValidationError(response, errors.New("database down"), zerolog.Nop())

	assertErrorResponse(t, response, http.StatusInternalServerError, "unexpected server error")
}

func newTestService(t *testing.T) *app.Service {
	t.Helper()

	logger := zerolog.Nop()
	store := packsize.NewStore(filepath.Join(t.TempDir(), "pack_sizes.json"), logger)
	planner := domain.NewShipmentPlanner(logger)
	return app.NewService(store, planner, logger)
}

func allocationMap(packs []allocation) map[int]int {
	result := make(map[int]int, len(packs))
	for _, pack := range packs {
		result[pack.PackSize] = pack.Quantity
	}
	return result
}

func assertErrorResponse(t *testing.T, response *httptest.ResponseRecorder, status int, message string) {
	t.Helper()

	if response.Code != status {
		t.Fatalf("status = %d, want %d", response.Code, status)
	}

	var result errorResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if result.Error != message {
		t.Fatalf("error = %q, want %q", result.Error, message)
	}
}
