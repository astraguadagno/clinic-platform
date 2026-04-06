package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	ServiceName string
	Version     string
	Environment string
}

type Server struct {
	config Config
	mux    *http.ServeMux
}

func NewServer(config Config) *Server {
	server := &Server{
		config: config,
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
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "list slots placeholder",
		})
	default:
		writeMethodNotAllowed(w, http.MethodGet)
	}
}

func (s *Server) bulkSlots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"message": "bulk slot creation not implemented yet",
	})
}

func (s *Server) appointments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "list appointments placeholder",
		})
	case http.MethodPost:
		writeJSON(w, http.StatusNotImplemented, map[string]any{
			"message": "create appointment not implemented yet",
		})
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
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "route not found"})
		return
	}

	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"message":        "cancel appointment not implemented yet",
		"appointment_id": parts[0],
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
		"message": "method not allowed",
	})
}
