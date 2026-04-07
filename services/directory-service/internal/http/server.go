package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	serviceauth "clinic-platform/services/directory-service/internal/auth"
	"clinic-platform/services/directory-service/internal/directory"
)

type Config struct {
	ServiceName  string
	Version      string
	Environment  string
	AuthTokenTTL time.Duration
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
	AuthenticateUser(ctx context.Context, email, password string) (directory.User, error)
	CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	GetUserBySessionToken(ctx context.Context, tokenHash string, now time.Time) (directory.User, error)
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
	s.mux.HandleFunc("/auth/login", s.login)
	s.mux.HandleFunc("/auth/me", s.me)
	s.mux.HandleFunc("/patients", s.patients)
	s.mux.HandleFunc("/patients/", s.patientByID)
	s.mux.HandleFunc("/professionals", s.professionals)
	s.mux.HandleFunc("/professionals/", s.professionalByID)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string         `json:"access_token"`
	TokenType   string         `json:"token_type"`
	ExpiresAt   time.Time      `json:"expires_at"`
	User        directory.User `json:"user"`
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

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var request loginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	user, err := s.repo.AuthenticateUser(r.Context(), request.Email, request.Password)
	if errors.Is(err, directory.ErrValidation) {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if errors.Is(err, directory.ErrUnauthorized) {
		writeUnauthorized(w)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to authenticate user")
		return
	}

	accessToken, tokenHash, err := serviceauth.GenerateSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	expiresAt := time.Now().UTC().Add(s.authTokenTTL())
	if err := s.repo.CreateSession(r.Context(), user.ID, tokenHash, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        user,
	})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	token, err := bearerTokenFromRequest(r)
	if err != nil {
		writeUnauthorized(w)
		return
	}

	user, err := s.repo.GetUserBySessionToken(r.Context(), serviceauth.HashSessionToken(token), time.Now().UTC())
	if errors.Is(err, directory.ErrUnauthorized) {
		writeUnauthorized(w)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load current user")
		return
	}

	writeJSON(w, http.StatusOK, user)
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

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="directory-service"`)
	writeError(w, http.StatusUnauthorized, "unauthorized")
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

func (s *Server) authTokenTTL() time.Duration {
	if s.config.AuthTokenTTL <= 0 {
		return 24 * time.Hour
	}

	return s.config.AuthTokenTTL
}
