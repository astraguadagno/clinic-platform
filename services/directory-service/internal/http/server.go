package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"clinic-platform/services/directory-service/internal/directory"
)

type Config struct {
	ServiceName string
	Version     string
	Environment string
}

type Server struct {
	config Config
	repo   directoryRepository
	mux    *http.ServeMux
}

type directoryRepository interface {
	CreatePatient(ctx context.Context, params directory.CreatePatientParams) (directory.Patient, error)
	ListPatients(ctx context.Context) ([]directory.Patient, error)
	GetPatientByID(ctx context.Context, id string) (directory.Patient, error)
	CreateProfessional(ctx context.Context, params directory.CreateProfessionalParams) (directory.Professional, error)
	ListProfessionals(ctx context.Context) ([]directory.Professional, error)
	GetProfessionalByID(ctx context.Context, id string) (directory.Professional, error)
}

func NewServer(config Config, repo directoryRepository) *Server {
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
	s.mux.HandleFunc("/patients", s.patients)
	s.mux.HandleFunc("/patients/", s.patientByID)
	s.mux.HandleFunc("/professionals", s.professionals)
	s.mux.HandleFunc("/professionals/", s.professionalByID)
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

func (s *Server) patients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listPatients(w, r)
	case http.MethodPost:
		s.createPatient(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) patientByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/patients/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "patient id is required")
		return
	}

	patient, err := s.repo.GetPatientByID(r.Context(), id)
	if errors.Is(err, directory.ErrNotFound) {
		writeError(w, http.StatusNotFound, "patient not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load patient")
		return
	}

	writeJSON(w, http.StatusOK, patient)
}

func (s *Server) professionals(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listProfessionals(w, r)
	case http.MethodPost:
		s.createProfessional(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) professionalByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/professionals/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "professional id is required")
		return
	}

	professional, err := s.repo.GetProfessionalByID(r.Context(), id)
	if errors.Is(err, directory.ErrNotFound) {
		writeError(w, http.StatusNotFound, "professional not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load professional")
		return
	}

	writeJSON(w, http.StatusOK, professional)
}

func (s *Server) createPatient(w http.ResponseWriter, r *http.Request) {
	var request directory.CreatePatientParams
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	patient, err := s.repo.CreatePatient(r.Context(), request)
	if errors.Is(err, directory.ErrValidation) {
		writeError(w, http.StatusBadRequest, "failed to create patient")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create patient")
		return
	}

	writeJSON(w, http.StatusCreated, patient)
}

func (s *Server) listPatients(w http.ResponseWriter, r *http.Request) {
	patients, err := s.repo.ListPatients(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list patients")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": patients})
}

func (s *Server) createProfessional(w http.ResponseWriter, r *http.Request) {
	var request directory.CreateProfessionalParams
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	professional, err := s.repo.CreateProfessional(r.Context(), request)
	if errors.Is(err, directory.ErrValidation) {
		writeError(w, http.StatusBadRequest, "failed to create professional")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create professional")
		return
	}

	writeJSON(w, http.StatusCreated, professional)
}

func (s *Server) listProfessionals(w http.ResponseWriter, r *http.Request) {
	professionals, err := s.repo.ListProfessionals(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list professionals")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": professionals})
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
