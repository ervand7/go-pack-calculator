package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"pack-calculator/internal/calculator"
	"pack-calculator/internal/config"
)

type calculateRequest struct {
	Items        int `json:"items"`
	ItemsOrdered int `json:"itemsOrdered"`
}

type packSizesRequest struct {
	PackSizes []int `json:"packSizes"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewRouter(store *config.Store, logger zerolog.Logger) *http.ServeMux {
	logger = logger.With().Str("component", "http_api").Logger()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/pack-sizes", withRequestLogging(handlePackSizes(store, logger), logger))
	mux.HandleFunc("/api/calculate", withRequestLogging(handleCalculate(store, logger), logger))
	return mux
}

func handlePackSizes(store *config.Store, logger zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug().Str("method", r.Method).Msg("handling pack sizes request")
		switch r.Method {
		case http.MethodGet:
			packSizes, err := store.PackSizes()
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
			packSizes, err := store.SavePackSizes(req.PackSizes)
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

func handleCalculate(store *config.Store, logger zerolog.Logger) http.HandlerFunc {
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

		packSizes, err := store.PackSizes()
		if err != nil {
			logger.Error().Err(err).Msg("could not read pack sizes for calculation")
			writeError(w, http.StatusInternalServerError, "could not read pack sizes", logger)
			return
		}

		result, err := calculator.CalculateWithLogger(itemsOrdered, packSizes, logger.With().Str("component", "calculator").Logger())
		if err != nil {
			writeValidationError(w, err, logger)
			return
		}

		writeJSON(w, http.StatusOK, result, logger)
	}
}

func writeValidationError(w http.ResponseWriter, err error, logger zerolog.Logger) {
	switch {
	case errors.Is(err, calculator.ErrInvalidItemCount),
		errors.Is(err, calculator.ErrNoPackSizes),
		errors.Is(err, calculator.ErrInvalidPackSize):
		logger.Warn().Err(err).Msg("validation error")
		writeError(w, http.StatusBadRequest, err.Error(), logger)
	default:
		logger.Error().Err(err).Msg("unexpected server error")
		writeError(w, http.StatusInternalServerError, "unexpected server error", logger)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any, logger zerolog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logger.Error().Err(err).Int("status", status).Msg("failed to write JSON response")
	}
}

func writeError(w http.ResponseWriter, status int, message string, logger zerolog.Logger) {
	writeJSON(w, status, errorResponse{Error: message}, logger)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

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
