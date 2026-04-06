package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"clinic-platform/services/appointments-service/internal/appointments"
)

type Config struct {
	ServiceName string
	Version     string
	Environment string
}

type Server struct {
	config Config
	repo   appointmentsRepository
	mux    *http.ServeMux
}

type appointmentsRepository interface {
	CreateSlotsBulk(ctx context.Context, params appointments.BulkCreateSlotsParams) ([]appointments.AvailabilitySlot, error)
	ListSlots(ctx context.Context, filters appointments.SlotFilters) ([]appointments.AvailabilitySlot, error)
	CreateAppointment(ctx context.Context, params appointments.CreateAppointmentParams) (appointments.Appointment, error)
	ListAppointments(ctx context.Context, filters appointments.AppointmentFilters) ([]appointments.Appointment, error)
	CancelAppointment(ctx context.Context, appointmentID string) (appointments.Appointment, error)
}

func NewServer(config Config, repo appointmentsRepository) *Server {
	server := &Server{
		config: config,
		repo:   repo,
		mux:    http.NewServeMux(),
	}

	server.registerRoutes()

	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.health)
	s.mux.HandleFunc("/info", s.info)
	s.mux.HandleFunc("/slots", s.slots)
	s.mux.HandleFunc("/slots/bulk", s.bulkSlots)
	s.mux.HandleFunc("/appointments", s.appointments)
	s.mux.HandleFunc("/appointments/", s.appointmentByIDAction)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": s.config.ServiceName,
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) info(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service":     s.config.ServiceName,
		"version":     s.config.Version,
		"environment": s.config.Environment,
	})
}

func (s *Server) slots(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listSlots(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet)
	}
}

func (s *Server) bulkSlots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var request appointments.BulkCreateSlotsParams
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	slots, err := s.repo.CreateSlotsBulk(r.Context(), request)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid slot bulk request")
		return
	}
	if errors.Is(err, appointments.ErrConflict) {
		writeError(w, http.StatusConflict, "slot range conflicts with existing slots")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create slots")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"items": slots})
}

func (s *Server) appointments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listAppointments(w, r)
	case http.MethodPost:
		s.createAppointment(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) appointmentByIDAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeMethodNotAllowed(w, http.MethodPatch)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/appointments/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "cancel" {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}

	appointment, err := s.repo.CancelAppointment(r.Context(), parts[0])
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid appointment id")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "appointment not found")
		return
	}
	if errors.Is(err, appointments.ErrConflict) {
		writeError(w, http.StatusConflict, "appointment already cancelled")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to cancel appointment")
		return
	}

	writeJSON(w, http.StatusOK, appointment)
}

func (s *Server) listSlots(w http.ResponseWriter, r *http.Request) {
	filters := appointments.SlotFilters{
		ProfessionalID: r.URL.Query().Get("professional_id"),
		Status:         r.URL.Query().Get("status"),
		Date:           r.URL.Query().Get("date"),
	}

	slots, err := s.repo.ListSlots(r.Context(), filters)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid slot filters")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list slots")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": slots})
}

func (s *Server) createAppointment(w http.ResponseWriter, r *http.Request) {
	var request appointments.CreateAppointmentParams
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	appointment, err := s.repo.CreateAppointment(r.Context(), request)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid appointment request")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "slot not found")
		return
	}
	if errors.Is(err, appointments.ErrConflict) {
		writeError(w, http.StatusConflict, "slot is not available")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create appointment")
		return
	}

	writeJSON(w, http.StatusCreated, appointment)
}

func (s *Server) listAppointments(w http.ResponseWriter, r *http.Request) {
	filters := appointments.AppointmentFilters{
		ProfessionalID: r.URL.Query().Get("professional_id"),
		PatientID:      r.URL.Query().Get("patient_id"),
		Status:         r.URL.Query().Get("status"),
		Date:           r.URL.Query().Get("date"),
	}

	items, err := s.repo.ListAppointments(r.Context(), filters)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid appointment filters")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list appointments")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": message,
	})
}
