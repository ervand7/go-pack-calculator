package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	app "pack-calculator/internal/application/orderpacks"
	domain "pack-calculator/internal/domain/orderpacks"
)

type calculateRequest struct {
	Items        int `json:"items"`
	ItemsOrdered int `json:"itemsOrdered"`
}

type calculateResponse struct {
	ItemsOrdered int          `json:"itemsOrdered"`
	ItemsShipped int          `json:"itemsShipped"`
	TotalPacks   int          `json:"totalPacks"`
	Packs        []allocation `json:"packs"`
}

type allocation struct {
	PackSize int `json:"packSize"`
	Quantity int `json:"quantity"`
}

type packSizesRequest struct {
	PackSizes []int `json:"packSizes"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// NewRouter registers HTTP routes for pack-size configuration and calculations.
func NewRouter(service *app.Service, logger zerolog.Logger) *http.ServeMux {
	logger = logger.With().Str("component", "http_api").Logger()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/pack-sizes", withRequestLogging(handlePackSizes(service, logger), logger))
	mux.HandleFunc("/api/calculate", withRequestLogging(handleCalculate(service, logger), logger))
	return mux
}

// handlePackSizes exposes configured pack sizes for reads and updates.
func handlePackSizes(service *app.Service, logger zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug().Str("method", r.Method).Msg("handling pack sizes request")
		switch r.Method {
		case http.MethodGet:
			packSizes, err := service.GetPackSizes(r.Context())
			if err != nil {
				logger.Error().Err(err).Msg("could not read pack sizes")
				writeError(w, http.StatusInternalServerError, "could not read pack sizes", logger)
				return
			}
			logger.Info().Ints("pack_sizes", packSizes).Msg("returning pack sizes")
			writeJSON(w, http.StatusOK, packSizesRequest{PackSizes: packSizes}, logger)
		case http.MethodPut, http.MethodPost:
			var req packSizesRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				logger.Warn().Err(err).Msg("invalid pack sizes JSON request")
				writeError(w, http.StatusBadRequest, "request body must be valid JSON", logger)
				return
			}
			packSizes, err := service.UpdatePackSizes(r.Context(), req.PackSizes)
			if err != nil {
				writeValidationError(w, err, logger)
				return
			}
			writeJSON(w, http.StatusOK, packSizesRequest{PackSizes: packSizes}, logger)
		default:
			logger.Warn().Str("method", r.Method).Msg("method not allowed for pack sizes endpoint")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", logger)
		}
	}
}

// handleCalculate calculates a shipment plan for the requested item count.
func handleCalculate(service *app.Service, logger zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			logger.Warn().Str("method", r.Method).Msg("method not allowed for calculate endpoint")
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", logger)
			return
		}

		var req calculateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn().Err(err).Msg("invalid calculate JSON request")
			writeError(w, http.StatusBadRequest, "request body must be valid JSON", logger)
			return
		}

		itemsOrdered := req.Items
		if req.ItemsOrdered != 0 {
			itemsOrdered = req.ItemsOrdered
			logger.Debug().
				Int("items", req.Items).
				Int("items_ordered", req.ItemsOrdered).
				Msg("using itemsOrdered request field")
		}
		logger.Info().Int("items_ordered", itemsOrdered).Msg("received calculation request")

		plan, err := service.Calculate(r.Context(), itemsOrdered)
		if err != nil {
			writeValidationError(w, err, logger)
			return
		}

		writeJSON(w, http.StatusOK, toCalculateResponse(plan), logger)
	}
}

// toCalculateResponse maps the domain shipment plan to the public API shape.
func toCalculateResponse(plan domain.ShipmentPlan) calculateResponse {
	packs := make([]allocation, 0, len(plan.Packs))
	for _, pack := range plan.Packs {
		packs = append(packs, allocation{
			PackSize: pack.PackSize,
			Quantity: pack.Quantity,
		})
	}

	return calculateResponse{
		ItemsOrdered: plan.ItemsOrdered,
		ItemsShipped: plan.ItemsShipped,
		TotalPacks:   plan.TotalPacks,
		Packs:        packs,
	}
}

// writeValidationError maps known domain validation errors to HTTP 400 responses.
func writeValidationError(w http.ResponseWriter, err error, logger zerolog.Logger) {
	switch {
	case errors.Is(err, domain.ErrInvalidItemCount),
		errors.Is(err, domain.ErrNoPackSizes),
		errors.Is(err, domain.ErrInvalidPackSize):
		logger.Warn().Err(err).Msg("validation error")
		writeError(w, http.StatusBadRequest, err.Error(), logger)
	default:
		logger.Error().Err(err).Msg("unexpected server error")
		writeError(w, http.StatusInternalServerError, "unexpected server error", logger)
	}
}

// writeJSON sends a JSON response and logs encoding failures.
func writeJSON(w http.ResponseWriter, status int, payload any, logger zerolog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logger.Error().Err(err).Int("status", status).Msg("failed to write JSON response")
	}
}

// writeError sends the standard API error response.
func writeError(w http.ResponseWriter, status int, message string, logger zerolog.Logger) {
	writeJSON(w, status, errorResponse{Error: message}, logger)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader records the status code before delegating to the response writer.
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// withRequestLogging logs each request with status and duration.
func withRequestLogging(next http.HandlerFunc, logger zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		logger.Debug().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Msg("request started")

		next(recorder, r)

		event := logger.Info()
		if recorder.status >= 500 {
			event = logger.Error()
		} else if recorder.status >= 400 {
			event = logger.Warn()
		}

		event.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", recorder.status).
			Dur("duration", time.Since(start)).
			Str("remote_addr", r.RemoteAddr).
			Msg("request completed")
	}
}
