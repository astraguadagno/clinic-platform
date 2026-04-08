package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"clinic-platform/services/appointments-service/internal/appointments"
	"clinic-platform/services/appointments-service/internal/directory"
)

type Config struct {
	ServiceName string
	Version     string
	Environment string
}

type Server struct {
	config Config
	repo   appointmentsRepository
	dir    directoryLookup
	mux    *http.ServeMux
}

type appointmentsRepository interface {
	CreateSlotsBulk(ctx context.Context, params appointments.BulkCreateSlotsParams) ([]appointments.AvailabilitySlot, error)
	ListSlots(ctx context.Context, filters appointments.SlotFilters) ([]appointments.AvailabilitySlot, error)
	CreateAppointment(ctx context.Context, params appointments.CreateAppointmentParams) (appointments.Appointment, error)
	ListAppointments(ctx context.Context, filters appointments.AppointmentFilters) ([]appointments.Appointment, error)
	GetAppointmentByID(ctx context.Context, appointmentID string) (appointments.Appointment, error)
	CancelAppointment(ctx context.Context, appointmentID string) (appointments.Appointment, error)
}

type directoryLookup interface {
	CurrentUser(ctx context.Context, bearer string) (directory.User, error)
	ProfessionalExists(ctx context.Context, professionalID string) (bool, error)
	PatientExists(ctx context.Context, patientID string) (bool, error)
}

type ActorContext struct {
	UserID         string
	Role           string
	ProfessionalID string
}

func NewServer(config Config, repo appointmentsRepository, dir directoryLookup) *Server {
	server := &Server{
		config: config,
		repo:   repo,
		dir:    dir,
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

	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	var request appointments.BulkCreateSlotsParams
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	request.ProfessionalID, ok = enforceProfessionalScope(actor, request.ProfessionalID)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}
	if err := appointments.ValidateBulkCreateSlotsParams(request); errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid slot bulk request")
		return
	}

	professionalExists, err := s.dir.ProfessionalExists(r.Context(), request.ProfessionalID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "directory service unavailable")
		return
	}
	if !professionalExists {
		writeError(w, http.StatusBadRequest, "professional not found")
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

	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/appointments/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "cancel" {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}

	if ok := s.authorizeCancelAppointment(w, r, actor, parts[0]); !ok {
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
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	professionalID, ok := enforceProfessionalScope(actor, r.URL.Query().Get("professional_id"))
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	filters := appointments.SlotFilters{
		ProfessionalID: professionalID,
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
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	var request appointments.CreateAppointmentParams
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	request.ProfessionalID, ok = enforceProfessionalScope(actor, request.ProfessionalID)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}
	if err := appointments.ValidateCreateAppointmentParams(request); errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid appointment request")
		return
	}

	professionalExists, err := s.dir.ProfessionalExists(r.Context(), request.ProfessionalID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "directory service unavailable")
		return
	}
	if !professionalExists {
		writeError(w, http.StatusBadRequest, "professional not found")
		return
	}

	patientExists, err := s.dir.PatientExists(r.Context(), request.PatientID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "directory service unavailable")
		return
	}
	if !patientExists {
		writeError(w, http.StatusBadRequest, "patient not found")
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
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	professionalID, ok := enforceProfessionalScope(actor, r.URL.Query().Get("professional_id"))
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	filters := appointments.AppointmentFilters{
		ProfessionalID: professionalID,
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

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="appointments-service"`)
	writeError(w, http.StatusUnauthorized, "unauthorized")
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

func bearerTokenFromRequest(r *http.Request) (string, error) {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return "", directory.ErrUnauthorized
	}

	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", directory.ErrUnauthorized
	}

	return strings.TrimSpace(parts[1]), nil
}

func (s *Server) currentActor(r *http.Request) (ActorContext, error) {
	bearer, err := bearerTokenFromRequest(r)
	if err != nil {
		return ActorContext{}, err
	}

	user, err := s.dir.CurrentUser(r.Context(), bearer)
	if err != nil {
		return ActorContext{}, err
	}

	actor := ActorContext{
		UserID: user.ID,
		Role:   strings.TrimSpace(user.Role),
	}
	if user.ProfessionalID != nil {
		actor.ProfessionalID = strings.TrimSpace(*user.ProfessionalID)
	}

	return actor, nil
}

func (s *Server) requireAgendaActor(w http.ResponseWriter, r *http.Request) (ActorContext, bool) {
	actor, err := s.currentActor(r)
	if errors.Is(err, directory.ErrUnauthorized) {
		writeUnauthorized(w)
		return ActorContext{}, false
	}
	if errors.Is(err, directory.ErrUnavailable) {
		writeError(w, http.StatusServiceUnavailable, "directory service unavailable")
		return ActorContext{}, false
	}
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "directory service unavailable")
		return ActorContext{}, false
	}
	if !slices.Contains([]string{"admin", "secretary", "doctor"}, actor.Role) {
		writeError(w, http.StatusForbidden, "insufficient role")
		return ActorContext{}, false
	}
	if actor.Role == "doctor" && actor.ProfessionalID == "" {
		writeError(w, http.StatusForbidden, "professional profile required")
		return ActorContext{}, false
	}

	return actor, true
}

func enforceProfessionalScope(actor ActorContext, requestedProfessionalID string) (string, bool) {
	requestedProfessionalID = strings.TrimSpace(requestedProfessionalID)
	if actor.Role != "doctor" {
		return requestedProfessionalID, true
	}
	if requestedProfessionalID == "" {
		return actor.ProfessionalID, true
	}

	return requestedProfessionalID, requestedProfessionalID == actor.ProfessionalID
}

func (s *Server) authorizeCancelAppointment(w http.ResponseWriter, r *http.Request, actor ActorContext, appointmentID string) bool {
	if actor.Role != "doctor" {
		return true
	}

	appointment, err := s.repo.GetAppointmentByID(r.Context(), appointmentID)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid appointment id")
		return false
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "appointment not found")
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load appointment")
		return false
	}
	if strings.TrimSpace(appointment.ProfessionalID) != actor.ProfessionalID {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return false
	}

	return true
}
