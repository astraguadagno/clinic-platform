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
	"github.com/google/uuid"
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
	CreateTemplate(ctx context.Context, params appointments.CreateTemplateParams) (appointments.ScheduleTemplate, error)
	GetActiveTemplate(ctx context.Context, professionalID string, effectiveDate string) (appointments.ScheduleTemplateVersion, error)
	GetTemplate(ctx context.Context, templateID string) (appointments.ScheduleTemplate, error)
	ListTemplateVersions(ctx context.Context, templateID string) ([]appointments.ScheduleTemplateVersion, error)
	CreateScheduleBlock(ctx context.Context, params appointments.CreateScheduleBlockParams) (appointments.ScheduleBlock, error)
	GetScheduleBlock(ctx context.Context, blockID string) (appointments.ScheduleBlock, error)
	ListScheduleBlocks(ctx context.Context, filters appointments.ScheduleBlockFilters) ([]appointments.ScheduleBlock, error)
	UpdateScheduleBlock(ctx context.Context, blockID string, params appointments.UpdateScheduleBlockParams) (appointments.ScheduleBlock, error)
	DeleteScheduleBlock(ctx context.Context, blockID string) error
	CreateConsultation(ctx context.Context, params appointments.CreateConsultationParams) (appointments.Consultation, error)
	GetConsultation(ctx context.Context, consultationID string) (appointments.Consultation, error)
	ListConsultations(ctx context.Context, filters appointments.ConsultationFilters) ([]appointments.Consultation, error)
	UpdateConsultationStatus(ctx context.Context, consultationID string, params appointments.UpdateConsultationStatusParams) (appointments.Consultation, error)
	CreateAppointment(ctx context.Context, params appointments.CreateAppointmentParams) (appointments.Appointment, error)
	ListAppointments(ctx context.Context, filters appointments.AppointmentFilters) ([]appointments.Appointment, error)
	GetAppointmentByID(ctx context.Context, appointmentID string) (appointments.Appointment, error)
	CancelAppointment(ctx context.Context, appointmentID string) (appointments.Appointment, error)
}

type createScheduleRequest struct {
	ProfessionalID string          `json:"professional_id"`
	EffectiveFrom  string          `json:"effective_from"`
	Recurrence     json.RawMessage `json:"recurrence"`
	CreatedBy      *string         `json:"created_by,omitempty"`
	Reason         *string         `json:"reason,omitempty"`
}

type createConsultationRequest struct {
	SlotID         *string                         `json:"slot_id,omitempty"`
	ProfessionalID string                          `json:"professional_id"`
	PatientID      string                          `json:"patient_id"`
	Source         appointments.ConsultationSource `json:"source"`
	ScheduledStart *time.Time                      `json:"scheduled_start,omitempty"`
	ScheduledEnd   *time.Time                      `json:"scheduled_end,omitempty"`
	Notes          *string                         `json:"notes,omitempty"`
}

type createBlockRequest struct {
	ProfessionalID string  `json:"professional_id"`
	Scope          string  `json:"scope"`
	BlockDate      *string `json:"block_date,omitempty"`
	StartDate      *string `json:"start_date,omitempty"`
	EndDate        *string `json:"end_date,omitempty"`
	DayOfWeek      *int    `json:"day_of_week,omitempty"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
	TemplateID     *string `json:"template_id,omitempty"`
}

type updateBlockRequest struct {
	ProfessionalID string  `json:"professional_id,omitempty"`
	Scope          string  `json:"scope"`
	BlockDate      *string `json:"block_date,omitempty"`
	StartDate      *string `json:"start_date,omitempty"`
	EndDate        *string `json:"end_date,omitempty"`
	DayOfWeek      *int    `json:"day_of_week,omitempty"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
	TemplateID     *string `json:"template_id,omitempty"`
}

type updateConsultationStatusRequest struct {
	ID             string                           `json:"id"`
	Status         appointments.ConsultationStatus  `json:"status"`
	Source         *appointments.ConsultationSource `json:"source,omitempty"`
	CheckInTime    *time.Time                       `json:"check_in_time,omitempty"`
	ReceptionNotes *string                          `json:"reception_notes,omitempty"`
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
	s.mux.HandleFunc("/schedules", s.schedules)
	s.mux.HandleFunc("/schedules/versions", s.scheduleVersions)
	s.mux.HandleFunc("/blocks", s.blocks)
	s.mux.HandleFunc("/blocks/", s.blockByID)
	s.mux.HandleFunc("/consultations", s.consultations)
	s.mux.HandleFunc("/agenda/week", s.agendaWeek)
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

func (s *Server) schedules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getSchedule(w, r)
	case http.MethodPost:
		s.createSchedule(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) consultations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getConsultation(w, r)
	case http.MethodPost:
		s.createConsultation(w, r)
	case http.MethodPatch:
		s.patchConsultationStatus(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost, http.MethodPatch)
	}
}

func (s *Server) blocks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listBlocks(w, r)
	case http.MethodPost:
		s.createBlock(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) blockByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getBlock(w, r)
	case http.MethodPatch:
		s.updateBlock(w, r)
	case http.MethodDelete:
		s.deleteBlock(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPatch, http.MethodDelete)
	}
}

func (s *Server) agendaWeek(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	s.getAgendaWeek(w, r)
}

func (s *Server) scheduleVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	s.getScheduleVersions(w, r)
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

func (s *Server) createSchedule(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}
	if actor.Role == "secretary" {
		writeError(w, http.StatusForbidden, "insufficient role")
		return
	}

	var request createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	request.ProfessionalID, ok = enforceProfessionalScope(actor, request.ProfessionalID)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	params := appointments.CreateTemplateParams{
		ProfessionalID: request.ProfessionalID,
		EffectiveFrom:  request.EffectiveFrom,
		Recurrence:     request.Recurrence,
		CreatedBy:      request.CreatedBy,
		Reason:         request.Reason,
	}
	if err := validateCreateScheduleRequest(params); errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid schedule request")
		return
	}

	professionalExists, err := s.dir.ProfessionalExists(r.Context(), params.ProfessionalID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "directory service unavailable")
		return
	}
	if !professionalExists {
		writeError(w, http.StatusBadRequest, "professional not found")
		return
	}

	template, err := s.repo.CreateTemplate(r.Context(), params)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid schedule request")
		return
	}
	if errors.Is(err, appointments.ErrConflict) {
		writeError(w, http.StatusConflict, "schedule version already exists for effective date")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create schedule")
		return
	}

	writeJSON(w, http.StatusCreated, template)
}

func (s *Server) getSchedule(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	professionalID, ok := enforceProfessionalScope(actor, r.URL.Query().Get("professional_id"))
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	effectiveDate := strings.TrimSpace(r.URL.Query().Get("effective_date"))
	schedule, err := appointments.NewScheduleService(s.repo).GetSchedule(r.Context(), professionalID, effectiveDate)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid schedule filters")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "schedule not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load schedule")
		return
	}

	writeJSON(w, http.StatusOK, schedule)
}

func (s *Server) getScheduleVersions(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	templateID := strings.TrimSpace(r.URL.Query().Get("template_id"))
	template, err := s.repo.GetTemplate(r.Context(), templateID)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid schedule filters")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "schedule not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load schedule versions")
		return
	}

	if _, ok := enforceProfessionalScope(actor, template.ProfessionalID); !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	versions, err := s.repo.ListTemplateVersions(r.Context(), templateID)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid schedule filters")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "schedule not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load schedule versions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": versions})
}

func (s *Server) createBlock(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	var request createBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	request.ProfessionalID, ok = enforceProfessionalScope(actor, request.ProfessionalID)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	params := appointments.CreateScheduleBlockParams{
		ProfessionalID: request.ProfessionalID,
		Scope:          request.Scope,
		BlockDate:      request.BlockDate,
		StartDate:      request.StartDate,
		EndDate:        request.EndDate,
		DayOfWeek:      request.DayOfWeek,
		StartTime:      request.StartTime,
		EndTime:        request.EndTime,
		TemplateID:     request.TemplateID,
	}
	if err := validateCreateBlockRequest(params); errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid block request")
		return
	}

	professionalExists, err := s.dir.ProfessionalExists(r.Context(), params.ProfessionalID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "directory service unavailable")
		return
	}
	if !professionalExists {
		writeError(w, http.StatusBadRequest, "professional not found")
		return
	}

	block, err := s.repo.CreateScheduleBlock(r.Context(), params)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid block request")
		return
	}
	if errors.Is(err, appointments.ErrConflict) {
		writeError(w, http.StatusConflict, "block conflicts with existing schedule")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create block")
		return
	}

	writeJSON(w, http.StatusCreated, block)
}

func (s *Server) listBlocks(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	professionalID, ok := enforceProfessionalScope(actor, r.URL.Query().Get("professional_id"))
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	filters := appointments.ScheduleBlockFilters{
		ProfessionalID: professionalID,
		TemplateID:     strings.TrimSpace(r.URL.Query().Get("template_id")),
		Scope:          strings.TrimSpace(r.URL.Query().Get("scope")),
	}

	blocks, err := s.repo.ListScheduleBlocks(r.Context(), filters)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid block filters")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list blocks")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": blocks})
}

func (s *Server) getBlock(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	block, err := s.repo.GetScheduleBlock(r.Context(), blockIDFromPath(r.URL.Path))
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid block id")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "block not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load block")
		return
	}

	if _, ok := enforceProfessionalScope(actor, block.ProfessionalID); !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	writeJSON(w, http.StatusOK, block)
}

func (s *Server) updateBlock(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	blockID := blockIDFromPath(r.URL.Path)
	existing, err := s.repo.GetScheduleBlock(r.Context(), blockID)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid block id")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "block not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load block")
		return
	}

	if _, ok := enforceProfessionalScope(actor, existing.ProfessionalID); !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	var request updateBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	professionalID := request.ProfessionalID
	if strings.TrimSpace(professionalID) == "" {
		professionalID = existing.ProfessionalID
	}
	professionalID, ok = enforceProfessionalScope(actor, professionalID)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	params := appointments.UpdateScheduleBlockParams{
		ProfessionalID: professionalID,
		Scope:          request.Scope,
		BlockDate:      request.BlockDate,
		StartDate:      request.StartDate,
		EndDate:        request.EndDate,
		DayOfWeek:      request.DayOfWeek,
		StartTime:      request.StartTime,
		EndTime:        request.EndTime,
		TemplateID:     request.TemplateID,
	}
	if err := validateUpdateBlockRequest(params); errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid block request")
		return
	}

	professionalExists, err := s.dir.ProfessionalExists(r.Context(), params.ProfessionalID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "directory service unavailable")
		return
	}
	if !professionalExists {
		writeError(w, http.StatusBadRequest, "professional not found")
		return
	}

	updated, err := s.repo.UpdateScheduleBlock(r.Context(), blockID, params)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid block request")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "block not found")
		return
	}
	if errors.Is(err, appointments.ErrConflict) {
		writeError(w, http.StatusConflict, "block conflicts with existing schedule")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update block")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteBlock(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	blockID := blockIDFromPath(r.URL.Path)
	block, err := s.repo.GetScheduleBlock(r.Context(), blockID)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid block id")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "block not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load block")
		return
	}

	if _, ok := enforceProfessionalScope(actor, block.ProfessionalID); !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	err = s.repo.DeleteScheduleBlock(r.Context(), blockID)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid block id")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "block not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete block")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createConsultation(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	var request createConsultationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	request.ProfessionalID, ok = enforceProfessionalScope(actor, request.ProfessionalID)
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	params := appointments.CreateConsultationParams{
		SlotID:         request.SlotID,
		ProfessionalID: request.ProfessionalID,
		PatientID:      request.PatientID,
		Source:         request.Source,
		ScheduledStart: request.ScheduledStart,
		ScheduledEnd:   request.ScheduledEnd,
		Notes:          request.Notes,
	}

	if _, err := validateCreateConsultationRequest(params); errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid consultation request")
		return
	}

	professionalExists, err := s.dir.ProfessionalExists(r.Context(), params.ProfessionalID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "directory service unavailable")
		return
	}
	if !professionalExists {
		writeError(w, http.StatusBadRequest, "professional not found")
		return
	}

	patientExists, err := s.dir.PatientExists(r.Context(), params.PatientID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "directory service unavailable")
		return
	}
	if !patientExists {
		writeError(w, http.StatusBadRequest, "patient not found")
		return
	}

	consultation, err := s.repo.CreateConsultation(r.Context(), params)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid consultation request")
		return
	}
	if errors.Is(err, appointments.ErrConflict) {
		writeError(w, http.StatusConflict, "consultation conflicts with existing schedule")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create consultation")
		return
	}

	writeJSON(w, http.StatusCreated, consultation)
}

func (s *Server) getConsultation(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	consultation, err := s.repo.GetConsultation(r.Context(), strings.TrimSpace(r.URL.Query().Get("id")))
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid consultation filters")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "consultation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load consultation")
		return
	}

	if _, ok := enforceProfessionalScope(actor, consultation.ProfessionalID); !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	writeJSON(w, http.StatusOK, consultation)
}

func (s *Server) patchConsultationStatus(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	var request updateConsultationStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	consultation, err := s.repo.GetConsultation(r.Context(), request.ID)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid consultation status update")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "consultation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load consultation")
		return
	}

	if _, ok := enforceProfessionalScope(actor, consultation.ProfessionalID); !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}
	if actor.Role == "secretary" && request.Status == appointments.ConsultationStatusCompleted {
		writeError(w, http.StatusForbidden, "insufficient role")
		return
	}

	updated, err := appointments.NewConsultationService(s.repo).UpdateStatus(r.Context(), request.ID, appointments.ConsultationStatusUpdateParams{
		Status:         request.Status,
		Source:         request.Source,
		ActorRole:      consultationActorRoleFromHTTPActor(actor.Role),
		CheckInTime:    request.CheckInTime,
		ReceptionNotes: request.ReceptionNotes,
	})
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid consultation status update")
		return
	}
	if errors.Is(err, appointments.ErrNotFound) {
		writeError(w, http.StatusNotFound, "consultation not found")
		return
	}
	if errors.Is(err, appointments.ErrConflict) {
		writeError(w, http.StatusConflict, "consultation status update conflicts with existing state")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update consultation")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) getAgendaWeek(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAgendaActor(w, r)
	if !ok {
		return
	}

	professionalID, ok := enforceProfessionalScope(actor, r.URL.Query().Get("professional_id"))
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden professional scope")
		return
	}

	weekStart, err := validateAgendaWeekFilters(professionalID, r.URL.Query().Get("week_start"))
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid agenda week filters")
		return
	}

	blocks, err := s.repo.ListScheduleBlocks(r.Context(), appointments.ScheduleBlockFilters{ProfessionalID: professionalID})
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid agenda week filters")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agenda week")
		return
	}

	consultations, err := s.repo.ListConsultations(r.Context(), appointments.ConsultationFilters{
		ProfessionalID: professionalID,
		WeekStart:      weekStart.Format("2006-01-02"),
	})
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid agenda week filters")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agenda week")
		return
	}

	templates, err := s.loadWeekAgendaTemplates(r.Context(), professionalID, weekStart)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid agenda week filters")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agenda week")
		return
	}

	agenda, err := appointments.ComposeWeekAgenda(professionalID, weekStart, templates, blocks, consultations)
	if errors.Is(err, appointments.ErrValidation) {
		writeError(w, http.StatusBadRequest, "invalid agenda week filters")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load agenda week")
		return
	}

	writeJSON(w, http.StatusOK, agenda)
}

func (s *Server) loadWeekAgendaTemplates(ctx context.Context, professionalID string, weekStart time.Time) ([]appointments.ScheduleTemplate, error) {
	effectiveDate := weekStart.AddDate(0, 0, 6).Format("2006-01-02")
	activeVersion, err := s.repo.GetActiveTemplate(ctx, professionalID, effectiveDate)
	if errors.Is(err, appointments.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	template, err := s.repo.GetTemplate(ctx, activeVersion.TemplateID)
	if err != nil {
		return nil, err
	}

	versions, err := s.repo.ListTemplateVersions(ctx, template.ID)
	if err != nil {
		return nil, err
	}
	template.Versions = versions

	return []appointments.ScheduleTemplate{template}, nil
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

func validateCreateScheduleRequest(params appointments.CreateTemplateParams) error {
	if _, err := uuid.Parse(strings.TrimSpace(params.ProfessionalID)); err != nil {
		return appointments.ErrValidation
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(params.EffectiveFrom)); err != nil {
		return appointments.ErrValidation
	}
	if !json.Valid(params.Recurrence) {
		return appointments.ErrValidation
	}
	var recurrence map[string]any
	if err := json.Unmarshal(params.Recurrence, &recurrence); err != nil || recurrence == nil {
		return appointments.ErrValidation
	}
	if params.CreatedBy != nil {
		if _, err := uuid.Parse(strings.TrimSpace(*params.CreatedBy)); err != nil {
			return appointments.ErrValidation
		}
	}
	return nil
}

func validateCreateConsultationRequest(params appointments.CreateConsultationParams) (appointments.CreateConsultationParams, error) {
	professionalID := strings.TrimSpace(params.ProfessionalID)
	patientID := strings.TrimSpace(params.PatientID)
	if _, err := uuid.Parse(professionalID); err != nil {
		return appointments.CreateConsultationParams{}, appointments.ErrValidation
	}
	if _, err := uuid.Parse(patientID); err != nil {
		return appointments.CreateConsultationParams{}, appointments.ErrValidation
	}
	if !params.Source.IsValid() {
		return appointments.CreateConsultationParams{}, appointments.ErrValidation
	}
	if params.SlotID != nil && strings.TrimSpace(*params.SlotID) == "" {
		return appointments.CreateConsultationParams{}, appointments.ErrValidation
	}
	if params.SlotID != nil && (params.ScheduledStart != nil || params.ScheduledEnd != nil) {
		return appointments.CreateConsultationParams{}, appointments.ErrValidation
	}
	if params.SlotID == nil {
		if params.ScheduledStart == nil || params.ScheduledEnd == nil {
			return appointments.CreateConsultationParams{}, appointments.ErrValidation
		}
		scheduledStart := params.ScheduledStart.UTC()
		scheduledEnd := params.ScheduledEnd.UTC()
		if !scheduledStart.Before(scheduledEnd) {
			return appointments.CreateConsultationParams{}, appointments.ErrValidation
		}
		params.ScheduledStart = &scheduledStart
		params.ScheduledEnd = &scheduledEnd
	}
	params.ProfessionalID = professionalID
	params.PatientID = patientID
	if params.Notes != nil {
		notes := strings.TrimSpace(*params.Notes)
		params.Notes = &notes
	}

	return params, nil
}

func validateCreateBlockRequest(params appointments.CreateScheduleBlockParams) error {
	_, err := normalizeBlockParams(params.ProfessionalID, params.Scope, params.BlockDate, params.StartDate, params.EndDate, params.DayOfWeek, params.StartTime, params.EndTime, params.TemplateID)
	return err
}

func validateUpdateBlockRequest(params appointments.UpdateScheduleBlockParams) error {
	_, err := normalizeBlockParams(params.ProfessionalID, params.Scope, params.BlockDate, params.StartDate, params.EndDate, params.DayOfWeek, params.StartTime, params.EndTime, params.TemplateID)
	return err
}

func normalizeBlockParams(professionalIDValue, scopeValue string, blockDateValue, startDateValue, endDateValue *string, dayOfWeekValue *int, startTimeValue, endTimeValue string, templateIDValue *string) (appointments.CreateScheduleBlockParams, error) {
	params := appointments.CreateScheduleBlockParams{
		ProfessionalID: strings.TrimSpace(professionalIDValue),
		Scope:          strings.TrimSpace(scopeValue),
		BlockDate:      trimOptionalString(blockDateValue),
		StartDate:      trimOptionalString(startDateValue),
		EndDate:        trimOptionalString(endDateValue),
		DayOfWeek:      dayOfWeekValue,
		StartTime:      strings.TrimSpace(startTimeValue),
		EndTime:        strings.TrimSpace(endTimeValue),
		TemplateID:     trimOptionalString(templateIDValue),
	}

	blockParams, err := appointments.ValidateScheduleBlockParams(params)
	if err != nil {
		return appointments.CreateScheduleBlockParams{}, err
	}

	return blockParams, nil
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func validateAgendaWeekFilters(professionalIDValue, weekStartValue string) (time.Time, error) {
	if _, err := uuid.Parse(strings.TrimSpace(professionalIDValue)); err != nil {
		return time.Time{}, appointments.ErrValidation
	}

	weekStart, err := time.Parse("2006-01-02", strings.TrimSpace(weekStartValue))
	if err != nil {
		return time.Time{}, appointments.ErrValidation
	}

	return weekStart.UTC(), nil
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

func consultationActorRoleFromHTTPActor(role string) appointments.ConsultationActorRole {
	switch role {
	case "doctor":
		return appointments.ConsultationActorRoleDoctor
	case "secretary":
		return appointments.ConsultationActorRoleSecretary
	default:
		return appointments.ConsultationActorRoleSecretary
	}
}

func blockIDFromPath(path string) string {
	return strings.Trim(strings.TrimPrefix(path, "/blocks/"), "/")
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
