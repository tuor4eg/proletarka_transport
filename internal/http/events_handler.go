package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"proletarka_transport/internal/config"
	"proletarka_transport/internal/domain"
	"proletarka_transport/internal/events"
)

type EventsHandler struct {
	config config.Config
	logger *slog.Logger
	dispatcher *events.Dispatcher
}

func NewEventsHandler(cfg config.Config, logger *slog.Logger, dispatcher *events.Dispatcher) http.Handler {
	handler := &EventsHandler{
		config:     cfg,
		logger:     logger,
		dispatcher: dispatcher,
	}

	mux := http.NewServeMux()
	mux.Handle("/events", handler)
	return mux
}

func (h *EventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/events" {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not found",
		})
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return
	}

	if r.Header.Get("x-outbound-events-secret") != h.config.Inbound.EventsSecret {
		h.logger.Warn("rejected inbound event with invalid secret", "remote_addr", r.RemoteAddr)
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid secret",
		})
		return
	}

	defer r.Body.Close()

	var event domain.Event
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json body",
		})
		return
	}

	if decoder.More() {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "request body must contain a single JSON object",
		})
		return
	}

	if err := event.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if err := h.dispatcher.Dispatch(r.Context(), event); err != nil {
		h.logger.Error("failed to process inbound event", "event", event.Event, "resource_kind", event.Resource.Kind, "resource_id", event.Resource.ID, "error", err.Error())
		status := http.StatusInternalServerError
		if errors.Is(err, events.ErrInvalidEvent) || errors.Is(err, events.ErrUnsupportedEvent) {
			status = http.StatusBadRequest
		}

		writeJSON(w, status, map[string]string{
			"error": err.Error(),
		})
		return
	}

	h.logger.Info("processed inbound event", "event", event.Event, "resource_kind", event.Resource.Kind, "resource_id", event.Resource.ID)
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
