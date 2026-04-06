package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"clinic-platform/services/directory-service/internal/directory"
)

func TestCreatePatientReturnsCreatedPatient(t *testing.T) {
	repo := &stubDirectoryRepository{
		createPatientFn: func(_ context.Context, params directory.CreatePatientParams) (directory.Patient, error) {
			if params.FirstName != "Ada" {
				t.Fatalf("first_name = %q, want Ada", params.FirstName)
			}
			return directory.Patient{ID: "patient-1", FirstName: "Ada", LastName: "Lovelace", Document: "123", BirthDate: "1990-10-10", Phone: "555", Active: true}, nil
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"first_name":"Ada","last_name":"Lovelace","document":"123","birth_date":"1990-10-10","phone":"555"}`)
	request := httptest.NewRequest(http.MethodPost, "/patients", body)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}

	var patient directory.Patient
	if err := json.NewDecoder(recorder.Body).Decode(&patient); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if patient.ID != "patient-1" {
		t.Fatalf("id = %q, want patient-1", patient.ID)
	}
}

func TestPatientByIDReturnsNotFound(t *testing.T) {
	repo := &stubDirectoryRepository{
		getPatientByIDFn: func(context.Context, string) (directory.Patient, error) {
			return directory.Patient{}, directory.ErrNotFound
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/patients/missing-id", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestListProfessionalsReturnsItems(t *testing.T) {
	repo := &stubDirectoryRepository{
		listProfessionalsFn: func(context.Context) ([]directory.Professional, error) {
			return []directory.Professional{{ID: "pro-1", FirstName: "Ana", LastName: "Lopez", Specialty: "cardiology", Active: true}}, nil
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/professionals", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Items []directory.Professional `json:"items"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(response.Items))
	}
}

func TestCreateProfessionalReturnsBadRequestOnInvalidJSON(t *testing.T) {
	server := NewServer(testConfig(), &stubDirectoryRepository{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/professionals", bytes.NewBufferString(`{"first_name":`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHealthReturnsStatusOK(t *testing.T) {
	server := NewServer(testConfig(), &stubDirectoryRepository{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

type stubDirectoryRepository struct {
	createPatientFn       func(context.Context, directory.CreatePatientParams) (directory.Patient, error)
	listPatientsFn        func(context.Context) ([]directory.Patient, error)
	getPatientByIDFn      func(context.Context, string) (directory.Patient, error)
	createProfessionalFn  func(context.Context, directory.CreateProfessionalParams) (directory.Professional, error)
	listProfessionalsFn   func(context.Context) ([]directory.Professional, error)
	getProfessionalByIDFn func(context.Context, string) (directory.Professional, error)
}

func (s *stubDirectoryRepository) CreatePatient(ctx context.Context, params directory.CreatePatientParams) (directory.Patient, error) {
	if s.createPatientFn == nil {
		return directory.Patient{}, errors.New("unexpected CreatePatient call")
	}
	return s.createPatientFn(ctx, params)
}

func (s *stubDirectoryRepository) ListPatients(ctx context.Context) ([]directory.Patient, error) {
	if s.listPatientsFn == nil {
		return nil, errors.New("unexpected ListPatients call")
	}
	return s.listPatientsFn(ctx)
}

func (s *stubDirectoryRepository) GetPatientByID(ctx context.Context, id string) (directory.Patient, error) {
	if s.getPatientByIDFn == nil {
		return directory.Patient{}, errors.New("unexpected GetPatientByID call")
	}
	return s.getPatientByIDFn(ctx, id)
}

func (s *stubDirectoryRepository) CreateProfessional(ctx context.Context, params directory.CreateProfessionalParams) (directory.Professional, error) {
	if s.createProfessionalFn == nil {
		return directory.Professional{}, errors.New("unexpected CreateProfessional call")
	}
	return s.createProfessionalFn(ctx, params)
}

func (s *stubDirectoryRepository) ListProfessionals(ctx context.Context) ([]directory.Professional, error) {
	if s.listProfessionalsFn == nil {
		return nil, errors.New("unexpected ListProfessionals call")
	}
	return s.listProfessionalsFn(ctx)
}

func (s *stubDirectoryRepository) GetProfessionalByID(ctx context.Context, id string) (directory.Professional, error) {
	if s.getProfessionalByIDFn == nil {
		return directory.Professional{}, errors.New("unexpected GetProfessionalByID call")
	}
	return s.getProfessionalByIDFn(ctx, id)
}

func testConfig() Config {
	return Config{ServiceName: "directory-service", Version: "test", Environment: "test"}
}

func TestInfoReturnsMetadata(t *testing.T) {
	server := NewServer(testConfig(), &stubDirectoryRepository{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/info", nil)

	server.ServeHTTP(recorder, request)

	var response map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["service"] != "directory-service" {
		t.Fatalf("service = %q, want directory-service", response["service"])
	}
}

func TestGetProfessionalByIDReturnsProfessional(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubDirectoryRepository{
		getProfessionalByIDFn: func(context.Context, string) (directory.Professional, error) {
			return directory.Professional{ID: "pro-1", FirstName: "Ana", LastName: "Lopez", Specialty: "cardiology", Active: true, CreatedAt: now, UpdatedAt: now}, nil
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/professionals/pro-1", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}
