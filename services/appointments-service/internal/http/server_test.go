package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"clinic-platform/services/appointments-service/internal/appointments"
	"clinic-platform/services/appointments-service/internal/directory"
)

func TestBulkSlotsReturnsCreatedItems(t *testing.T) {
	repoCalled := false
	repo := &stubAppointmentsRepository{
		createSlotsBulkFn: func(_ context.Context, params appointments.BulkCreateSlotsParams) ([]appointments.AvailabilitySlot, error) {
			repoCalled = true
			if params.SlotDurationMinutes != 30 {
				t.Fatalf("slot_duration_minutes = %d, want 30", params.SlotDurationMinutes)
			}
			return []appointments.AvailabilitySlot{{ID: "slot-1", ProfessionalID: params.ProfessionalID, Status: "available"}}, nil
		},
	}

	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{professionalExists: true})
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"professional_id":"550e8400-e29b-41d4-a716-446655440000","date":"2026-04-10","start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}`)
	request := httptest.NewRequest(http.MethodPost, "/slots/bulk", body)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if !repoCalled {
		t.Fatal("expected repo to be called")
	}
}

func TestCreateAppointmentPostScenarios(t *testing.T) {
	const (
		validSlotID         = "550e8400-e29b-41d4-a716-446655440000"
		validPatientID      = "550e8400-e29b-41d4-a716-446655440001"
		validProfessionalID = "550e8400-e29b-41d4-a716-446655440002"
	)

	validBody := `{"slot_id":"` + validSlotID + `","patient_id":"` + validPatientID + `","professional_id":"` + validProfessionalID + `"}`
	createdAt := time.Date(2026, time.April, 10, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)
	createdAppointment := appointments.Appointment{
		ID:             "appt-123",
		SlotID:         validSlotID,
		PatientID:      validPatientID,
		ProfessionalID: validProfessionalID,
		Status:         "booked",
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	tests := []struct {
		name                  string
		body                  string
		directory             stubDirectoryLookup
		repoErr               error
		repoAppointment       appointments.Appointment
		wantStatus            int
		wantError             string
		wantAppointment       *appointments.Appointment
		wantProfessionalCalls int
		wantPatientCalls      int
		wantCreateRepoCalls   int
	}{
		{
			name:                  "invalid json body returns bad request",
			body:                  `{"slot_id":`,
			wantStatus:            http.StatusBadRequest,
			wantError:             "invalid json body",
			wantProfessionalCalls: 0,
			wantPatientCalls:      0,
			wantCreateRepoCalls:   0,
		},
		{
			name:                  "invalid appointment params returns bad request",
			body:                  `{"slot_id":"not-a-uuid","patient_id":"` + validPatientID + `","professional_id":"` + validProfessionalID + `"}`,
			wantStatus:            http.StatusBadRequest,
			wantError:             "invalid appointment request",
			wantProfessionalCalls: 0,
			wantPatientCalls:      0,
			wantCreateRepoCalls:   0,
		},
		{
			name: "professional availability lookup error returns service unavailable",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalErr: directory.ErrUnavailable,
			},
			wantStatus:            http.StatusServiceUnavailable,
			wantError:             "directory service unavailable",
			wantProfessionalCalls: 1,
			wantPatientCalls:      0,
			wantCreateRepoCalls:   0,
		},
		{
			name: "missing professional short circuits before repo",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: false,
			},
			wantStatus:            http.StatusBadRequest,
			wantError:             "professional not found",
			wantProfessionalCalls: 1,
			wantPatientCalls:      0,
			wantCreateRepoCalls:   0,
		},
		{
			name: "missing patient short circuits before repo",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
				patientExists:      false,
			},
			wantStatus:            http.StatusBadRequest,
			wantError:             "patient not found",
			wantProfessionalCalls: 1,
			wantPatientCalls:      1,
			wantCreateRepoCalls:   0,
		},
		{
			name: "repo validation error returns bad request",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
				patientExists:      true,
			},
			repoErr:               appointments.ErrValidation,
			wantStatus:            http.StatusBadRequest,
			wantError:             "invalid appointment request",
			wantProfessionalCalls: 1,
			wantPatientCalls:      1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "repo not found returns not found",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
				patientExists:      true,
			},
			repoErr:               appointments.ErrNotFound,
			wantStatus:            http.StatusNotFound,
			wantError:             "slot not found",
			wantProfessionalCalls: 1,
			wantPatientCalls:      1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "repo conflict returns conflict",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
				patientExists:      true,
			},
			repoErr:               appointments.ErrConflict,
			wantStatus:            http.StatusConflict,
			wantError:             "slot is not available",
			wantProfessionalCalls: 1,
			wantPatientCalls:      1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "repo generic error returns internal server error",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
				patientExists:      true,
			},
			repoErr:               errors.New("db down"),
			wantStatus:            http.StatusInternalServerError,
			wantError:             "failed to create appointment",
			wantProfessionalCalls: 1,
			wantPatientCalls:      1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "success returns created appointment",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
				patientExists:      true,
			},
			repoAppointment:       createdAppointment,
			wantStatus:            http.StatusCreated,
			wantAppointment:       &createdAppointment,
			wantProfessionalCalls: 1,
			wantPatientCalls:      1,
			wantCreateRepoCalls:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.directory
			repo := &stubAppointmentsRepository{
				createAppointmentFn: func(_ context.Context, params appointments.CreateAppointmentParams) (appointments.Appointment, error) {
					if params.SlotID != validSlotID {
						t.Fatalf("slot_id = %q, want %q", params.SlotID, validSlotID)
					}
					if params.PatientID != validPatientID {
						t.Fatalf("patient_id = %q, want %q", params.PatientID, validPatientID)
					}
					if params.ProfessionalID != validProfessionalID {
						t.Fatalf("professional_id = %q, want %q", params.ProfessionalID, validProfessionalID)
					}
					return tt.repoAppointment, tt.repoErr
				},
			}

			server := NewServer(testAppointmentsConfig(), repo, &dir)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/appointments", bytes.NewBufferString(tt.body))

			server.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if dir.professionalCalls != tt.wantProfessionalCalls {
				t.Fatalf("ProfessionalExists calls = %d, want %d", dir.professionalCalls, tt.wantProfessionalCalls)
			}
			if dir.patientCalls != tt.wantPatientCalls {
				t.Fatalf("PatientExists calls = %d, want %d", dir.patientCalls, tt.wantPatientCalls)
			}
			if repo.createAppointmentCalls != tt.wantCreateRepoCalls {
				t.Fatalf("CreateAppointment calls = %d, want %d", repo.createAppointmentCalls, tt.wantCreateRepoCalls)
			}

			if tt.wantError != "" {
				assertErrorResponse(t, recorder.Body, tt.wantError)
				return
			}

			var response appointments.Appointment
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response != *tt.wantAppointment {
				t.Fatalf("response = %+v, want %+v", response, *tt.wantAppointment)
			}
		})
	}
}

func TestListAppointmentsReturnsItems(t *testing.T) {
	repo := &stubAppointmentsRepository{
		listAppointmentsFn: func(context.Context, appointments.AppointmentFilters) ([]appointments.Appointment, error) {
			return []appointments.Appointment{{ID: "appt-1", Status: "booked"}}, nil
		},
	}

	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/appointments?status=booked", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Items []appointments.Appointment `json:"items"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(response.Items))
	}
}

func TestCancelAppointmentReturnsAppointment(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubAppointmentsRepository{
		cancelAppointmentFn: func(context.Context, string) (appointments.Appointment, error) {
			return appointments.Appointment{ID: "appt-1", Status: "cancelled", CancelledAt: &now}, nil
		},
	}

	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/appointments/appt-1/cancel", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestListSlotsReturnsBadRequestOnInvalidFilters(t *testing.T) {
	repo := &stubAppointmentsRepository{
		listSlotsFn: func(context.Context, appointments.SlotFilters) ([]appointments.AvailabilitySlot, error) {
			return nil, appointments.ErrValidation
		},
	}

	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/slots?date=bad-date", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestBulkSlotsReturnsBadRequestWhenProfessionalMissing(t *testing.T) {
	repo := &stubAppointmentsRepository{}
	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{professionalExists: false})
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"professional_id":"550e8400-e29b-41d4-a716-446655440000","date":"2026-04-10","start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}`)
	request := httptest.NewRequest(http.MethodPost, "/slots/bulk", body)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestBulkSlotsReturnsServiceUnavailableWhenDirectoryFails(t *testing.T) {
	repo := &stubAppointmentsRepository{}
	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{professionalErr: directory.ErrUnavailable})
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"professional_id":"550e8400-e29b-41d4-a716-446655440000","date":"2026-04-10","start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}`)
	request := httptest.NewRequest(http.MethodPost, "/slots/bulk", body)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

type stubAppointmentsRepository struct {
	createSlotsBulkFn      func(context.Context, appointments.BulkCreateSlotsParams) ([]appointments.AvailabilitySlot, error)
	listSlotsFn            func(context.Context, appointments.SlotFilters) ([]appointments.AvailabilitySlot, error)
	createAppointmentFn    func(context.Context, appointments.CreateAppointmentParams) (appointments.Appointment, error)
	listAppointmentsFn     func(context.Context, appointments.AppointmentFilters) ([]appointments.Appointment, error)
	cancelAppointmentFn    func(context.Context, string) (appointments.Appointment, error)
	createAppointmentCalls int
}

type stubDirectoryLookup struct {
	professionalExists bool
	professionalErr    error
	patientExists      bool
	patientErr         error
	professionalCalls  int
	patientCalls       int
}

func (s *stubDirectoryLookup) ProfessionalExists(context.Context, string) (bool, error) {
	s.professionalCalls++
	if s.professionalErr != nil {
		return false, s.professionalErr
	}
	return s.professionalExists, nil
}

func (s *stubDirectoryLookup) PatientExists(context.Context, string) (bool, error) {
	s.patientCalls++
	if s.patientErr != nil {
		return false, s.patientErr
	}
	return s.patientExists, nil
}

func (s *stubAppointmentsRepository) CreateSlotsBulk(ctx context.Context, params appointments.BulkCreateSlotsParams) ([]appointments.AvailabilitySlot, error) {
	if s.createSlotsBulkFn == nil {
		return nil, errors.New("unexpected CreateSlotsBulk call")
	}
	return s.createSlotsBulkFn(ctx, params)
}

func (s *stubAppointmentsRepository) ListSlots(ctx context.Context, filters appointments.SlotFilters) ([]appointments.AvailabilitySlot, error) {
	if s.listSlotsFn == nil {
		return nil, errors.New("unexpected ListSlots call")
	}
	return s.listSlotsFn(ctx, filters)
}

func (s *stubAppointmentsRepository) CreateAppointment(ctx context.Context, params appointments.CreateAppointmentParams) (appointments.Appointment, error) {
	s.createAppointmentCalls++
	if s.createAppointmentFn == nil {
		return appointments.Appointment{}, errors.New("unexpected CreateAppointment call")
	}
	return s.createAppointmentFn(ctx, params)
}

func assertErrorResponse(t *testing.T, body io.Reader, want string) {
	t.Helper()

	var response struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error != want {
		t.Fatalf("error = %q, want %q", response.Error, want)
	}
}

func (s *stubAppointmentsRepository) ListAppointments(ctx context.Context, filters appointments.AppointmentFilters) ([]appointments.Appointment, error) {
	if s.listAppointmentsFn == nil {
		return nil, errors.New("unexpected ListAppointments call")
	}
	return s.listAppointmentsFn(ctx, filters)
}

func (s *stubAppointmentsRepository) CancelAppointment(ctx context.Context, appointmentID string) (appointments.Appointment, error) {
	if s.cancelAppointmentFn == nil {
		return appointments.Appointment{}, errors.New("unexpected CancelAppointment call")
	}
	return s.cancelAppointmentFn(ctx, appointmentID)
}

func testAppointmentsConfig() Config {
	return Config{ServiceName: "appointments-service", Version: "test", Environment: "test"}
}
