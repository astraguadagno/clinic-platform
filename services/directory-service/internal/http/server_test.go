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

func TestCreateEncounterReturnsUnauthorizedWithoutBearerToken(t *testing.T) {
	server := NewServer(testConfig(), &stubDirectoryRepository{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/patients/0f0f6c4d-7bbb-4d8e-94f9-f13fca1d16ca/encounters", bytes.NewBufferString(`{"note":"Paciente estable"}`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestCreateEncounterReturnsForbiddenWithoutProfessionalProfile(t *testing.T) {
	repo := &stubDirectoryRepository{
		getUserBySessionTokenFn: func(context.Context, string, time.Time) (directory.User, error) {
			return directory.User{ID: "user-1", Email: "admin@clinic.local", Role: "admin", Active: true}, nil
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/patients/0f0f6c4d-7bbb-4d8e-94f9-f13fca1d16ca/encounters", bytes.NewBufferString(`{"note":"Paciente estable"}`))
	request.Header.Set("Authorization", "Bearer test-token")

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCreateEncounterReturnsCreatedEncounter(t *testing.T) {
	professionalID := "f58d7e2f-c5fc-4884-b7bb-a3d14577a995"
	patientID := "0f0f6c4d-7bbb-4d8e-94f9-f13fca1d16ca"
	occurredAt := time.Date(2026, 4, 7, 14, 30, 0, 0, time.UTC)
	now := time.Date(2026, 4, 7, 14, 31, 0, 0, time.UTC)

	repo := &stubDirectoryRepository{
		getUserBySessionTokenFn: func(context.Context, string, time.Time) (directory.User, error) {
			return directory.User{ID: "user-1", Email: "doctor@clinic.local", Role: "doctor", ProfessionalID: &professionalID, Active: true}, nil
		},
		createEncounterFn: func(_ context.Context, params directory.CreateEncounterParams) (directory.Encounter, error) {
			if params.PatientID != patientID {
				t.Fatalf("patientID = %q, want %q", params.PatientID, patientID)
			}
			if params.ProfessionalID != professionalID {
				t.Fatalf("professionalID = %q, want %q", params.ProfessionalID, professionalID)
			}
			if params.Note != "Paciente estable" {
				t.Fatalf("note = %q, want Paciente estable", params.Note)
			}
			return directory.Encounter{
				ID:             "enc-1",
				ChartID:        "chart-1",
				PatientID:      patientID,
				ProfessionalID: professionalID,
				OccurredAt:     occurredAt,
				CreatedAt:      now,
				UpdatedAt:      now,
				InitialNote: directory.ClinicalNote{
					ID:             "note-1",
					EncounterID:    "enc-1",
					ChartID:        "chart-1",
					PatientID:      patientID,
					ProfessionalID: professionalID,
					Kind:           "initial",
					Content:        "Paciente estable",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			}, nil
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/patients/"+patientID+"/encounters", bytes.NewBufferString(`{"occurred_at":"2026-04-07T14:30:00Z","note":"Paciente estable"}`))
	request.Header.Set("Authorization", "Bearer test-token")

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}

	var encounter directory.Encounter
	if err := json.NewDecoder(recorder.Body).Decode(&encounter); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if encounter.ID != "enc-1" {
		t.Fatalf("id = %q, want enc-1", encounter.ID)
	}
	if encounter.InitialNote.Content != "Paciente estable" {
		t.Fatalf("initial_note.content = %q, want Paciente estable", encounter.InitialNote.Content)
	}
}

func TestListPatientEncountersReturnsItemsForCurrentProfessional(t *testing.T) {
	professionalID := "f58d7e2f-c5fc-4884-b7bb-a3d14577a995"
	patientID := "0f0f6c4d-7bbb-4d8e-94f9-f13fca1d16ca"
	now := time.Date(2026, 4, 7, 14, 31, 0, 0, time.UTC)

	repo := &stubDirectoryRepository{
		getUserBySessionTokenFn: func(context.Context, string, time.Time) (directory.User, error) {
			return directory.User{ID: "user-1", Email: "doctor@clinic.local", Role: "doctor", ProfessionalID: &professionalID, Active: true}, nil
		},
		listPatientEncountersFn: func(_ context.Context, gotPatientID, gotProfessionalID string) ([]directory.Encounter, error) {
			if gotPatientID != patientID {
				t.Fatalf("patientID = %q, want %q", gotPatientID, patientID)
			}
			if gotProfessionalID != professionalID {
				t.Fatalf("professionalID = %q, want %q", gotProfessionalID, professionalID)
			}
			return []directory.Encounter{{
				ID:             "enc-1",
				ChartID:        "chart-1",
				PatientID:      patientID,
				ProfessionalID: professionalID,
				OccurredAt:     now,
				CreatedAt:      now,
				UpdatedAt:      now,
				InitialNote: directory.ClinicalNote{
					ID:             "note-1",
					EncounterID:    "enc-1",
					ChartID:        "chart-1",
					PatientID:      patientID,
					ProfessionalID: professionalID,
					Kind:           "initial",
					Content:        "Paciente estable",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			}}, nil
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/patients/"+patientID+"/encounters", nil)
	request.Header.Set("Authorization", "Bearer test-token")

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Items []directory.Encounter `json:"items"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(response.Items))
	}
	if response.Items[0].ProfessionalID != professionalID {
		t.Fatalf("professional_id = %q, want %q", response.Items[0].ProfessionalID, professionalID)
	}
}

func TestLoginReturnsAccessToken(t *testing.T) {
	repo := &stubDirectoryRepository{
		authenticateUserFn: func(_ context.Context, email, password string) (directory.User, error) {
			if email != "admin@clinic.local" {
				t.Fatalf("email = %q, want admin@clinic.local", email)
			}
			if password != "admin123" {
				t.Fatalf("password = %q, want admin123", password)
			}
			return directory.User{ID: "user-1", Email: email, Role: "admin", Active: true}, nil
		},
		createSessionFn: func(_ context.Context, userID, tokenHash string, expiresAt time.Time) error {
			if userID != "user-1" {
				t.Fatalf("userID = %q, want user-1", userID)
			}
			if tokenHash == "" {
				t.Fatal("tokenHash should not be empty")
			}
			if expiresAt.IsZero() {
				t.Fatal("expiresAt should not be zero")
			}
			return nil
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"admin@clinic.local","password":"admin123"}`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response loginResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.AccessToken == "" {
		t.Fatal("access token should not be empty")
	}
	if response.TokenType != "Bearer" {
		t.Fatalf("token_type = %q, want Bearer", response.TokenType)
	}
	if response.User.Email != "admin@clinic.local" {
		t.Fatalf("user.email = %q, want admin@clinic.local", response.User.Email)
	}
	if response.ExpiresAt.IsZero() {
		t.Fatal("expires_at should not be zero")
	}
}

func TestLoginReturnsUnauthorizedOnInvalidCredentials(t *testing.T) {
	repo := &stubDirectoryRepository{
		authenticateUserFn: func(context.Context, string, string) (directory.User, error) {
			return directory.User{}, directory.ErrUnauthorized
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"admin@clinic.local","password":"bad"}`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestLoginReturnsBadRequestOnMissingCredentials(t *testing.T) {
	repo := &stubDirectoryRepository{
		authenticateUserFn: func(context.Context, string, string) (directory.User, error) {
			return directory.User{}, directory.ErrValidation
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{"email":"","password":""}`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestMeReturnsCurrentUser(t *testing.T) {
	repo := &stubDirectoryRepository{
		getUserBySessionTokenFn: func(_ context.Context, tokenHash string, _ time.Time) (directory.User, error) {
			if tokenHash == "" {
				t.Fatal("tokenHash should not be empty")
			}
			return directory.User{ID: "user-1", Email: "admin@clinic.local", Role: "admin", Active: true}, nil
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	request.Header.Set("Authorization", "Bearer test-token")

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response directory.User
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Email != "admin@clinic.local" {
		t.Fatalf("email = %q, want admin@clinic.local", response.Email)
	}
}

func TestMeReturnsUnauthorizedWithoutBearerToken(t *testing.T) {
	server := NewServer(testConfig(), &stubDirectoryRepository{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestMeReturnsUnauthorizedOnUnknownSession(t *testing.T) {
	repo := &stubDirectoryRepository{
		getUserBySessionTokenFn: func(context.Context, string, time.Time) (directory.User, error) {
			return directory.User{}, directory.ErrUnauthorized
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	request.Header.Set("Authorization", "Bearer missing-token")

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
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

func TestCreatePatientInvalidBirthDateReturnsBadRequest(t *testing.T) {
	repo := &stubDirectoryRepository{
		createPatientFn: func(context.Context, directory.CreatePatientParams) (directory.Patient, error) {
			return directory.Patient{}, directory.ErrValidation
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/patients", bytes.NewBufferString(`{"first_name":"Ada","last_name":"Lovelace","document":"123","birth_date":"10-10-1990","phone":"555"}`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestCreatePatientMissingRequiredFieldReturnsBadRequest(t *testing.T) {
	repo := &stubDirectoryRepository{
		createPatientFn: func(context.Context, directory.CreatePatientParams) (directory.Patient, error) {
			return directory.Patient{}, directory.ErrValidation
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/patients", bytes.NewBufferString(`{"first_name":"Ada","last_name":"","document":"123","birth_date":"1990-10-10","phone":"555"}`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestCreatePatientUnexpectedRepoErrorReturnsInternalServerError(t *testing.T) {
	repo := &stubDirectoryRepository{
		createPatientFn: func(context.Context, directory.CreatePatientParams) (directory.Patient, error) {
			return directory.Patient{}, errors.New("db down")
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/patients", bytes.NewBufferString(`{"first_name":"Ada","last_name":"Lovelace","document":"123","birth_date":"1990-10-10","phone":"555"}`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestCreateProfessionalValidReturnsCreated(t *testing.T) {
	repo := &stubDirectoryRepository{
		createProfessionalFn: func(_ context.Context, params directory.CreateProfessionalParams) (directory.Professional, error) {
			if params.Specialty != "cardiology" {
				t.Fatalf("specialty = %q, want cardiology", params.Specialty)
			}
			return directory.Professional{ID: "pro-1", FirstName: "Ana", LastName: "Lopez", Specialty: "cardiology", Active: true}, nil
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/professionals", bytes.NewBufferString(`{"first_name":"Ana","last_name":"Lopez","specialty":"cardiology"}`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
}

func TestCreateProfessionalMissingSpecialtyReturnsBadRequest(t *testing.T) {
	repo := &stubDirectoryRepository{
		createProfessionalFn: func(context.Context, directory.CreateProfessionalParams) (directory.Professional, error) {
			return directory.Professional{}, directory.ErrValidation
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/professionals", bytes.NewBufferString(`{"first_name":"Ana","last_name":"Lopez","specialty":""}`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestCreateProfessionalUnexpectedRepoErrorReturnsInternalServerError(t *testing.T) {
	repo := &stubDirectoryRepository{
		createProfessionalFn: func(context.Context, directory.CreateProfessionalParams) (directory.Professional, error) {
			return directory.Professional{}, errors.New("db down")
		},
	}

	server := NewServer(testConfig(), repo)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/professionals", bytes.NewBufferString(`{"first_name":"Ana","last_name":"Lopez","specialty":"cardiology"}`))

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
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
	createPatientFn         func(context.Context, directory.CreatePatientParams) (directory.Patient, error)
	listPatientsFn          func(context.Context) ([]directory.Patient, error)
	getPatientByIDFn        func(context.Context, string) (directory.Patient, error)
	createEncounterFn       func(context.Context, directory.CreateEncounterParams) (directory.Encounter, error)
	listPatientEncountersFn func(context.Context, string, string) ([]directory.Encounter, error)
	createProfessionalFn    func(context.Context, directory.CreateProfessionalParams) (directory.Professional, error)
	listProfessionalsFn     func(context.Context) ([]directory.Professional, error)
	getProfessionalByIDFn   func(context.Context, string) (directory.Professional, error)
	authenticateUserFn      func(context.Context, string, string) (directory.User, error)
	createSessionFn         func(context.Context, string, string, time.Time) error
	getUserBySessionTokenFn func(context.Context, string, time.Time) (directory.User, error)
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

func (s *stubDirectoryRepository) CreateEncounter(ctx context.Context, params directory.CreateEncounterParams) (directory.Encounter, error) {
	if s.createEncounterFn == nil {
		return directory.Encounter{}, errors.New("unexpected CreateEncounter call")
	}
	return s.createEncounterFn(ctx, params)
}

func (s *stubDirectoryRepository) ListPatientEncounters(ctx context.Context, patientID, professionalID string) ([]directory.Encounter, error) {
	if s.listPatientEncountersFn == nil {
		return nil, errors.New("unexpected ListPatientEncounters call")
	}
	return s.listPatientEncountersFn(ctx, patientID, professionalID)
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

func (s *stubDirectoryRepository) AuthenticateUser(ctx context.Context, email, password string) (directory.User, error) {
	if s.authenticateUserFn == nil {
		return directory.User{}, errors.New("unexpected AuthenticateUser call")
	}
	return s.authenticateUserFn(ctx, email, password)
}

func (s *stubDirectoryRepository) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	if s.createSessionFn == nil {
		return errors.New("unexpected CreateSession call")
	}
	return s.createSessionFn(ctx, userID, tokenHash, expiresAt)
}

func (s *stubDirectoryRepository) GetUserBySessionToken(ctx context.Context, tokenHash string, now time.Time) (directory.User, error) {
	if s.getUserBySessionTokenFn == nil {
		return directory.User{}, errors.New("unexpected GetUserBySessionToken call")
	}
	return s.getUserBySessionTokenFn(ctx, tokenHash, now)
}

func testConfig() Config {
	return Config{ServiceName: "directory-service", Version: "test", Environment: "test", AuthTokenTTL: time.Hour}
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
