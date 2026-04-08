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
	request := newAuthenticatedRequest(http.MethodPost, "/slots/bulk", body)

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
			request := newAuthenticatedRequest(http.MethodPost, "/appointments", bytes.NewBufferString(tt.body))

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
	request := newAuthenticatedRequest(http.MethodGet, "/appointments?status=booked", nil)

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
	request := newAuthenticatedRequest(http.MethodPatch, "/appointments/appt-1/cancel", nil)

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
	request := newAuthenticatedRequest(http.MethodGet, "/slots?date=bad-date", nil)

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
	request := newAuthenticatedRequest(http.MethodPost, "/slots/bulk", body)

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
	request := newAuthenticatedRequest(http.MethodPost, "/slots/bulk", body)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestAgendaEndpointsRejectAnonymousRequests(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		body   io.Reader
	}{
		{name: "list slots", method: http.MethodGet, target: "/slots?professional_id=550e8400-e29b-41d4-a716-446655440000"},
		{name: "bulk slots", method: http.MethodPost, target: "/slots/bulk", body: bytes.NewBufferString(`{"professional_id":"550e8400-e29b-41d4-a716-446655440000","date":"2026-04-10","start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}`)},
		{name: "list appointments", method: http.MethodGet, target: "/appointments?professional_id=550e8400-e29b-41d4-a716-446655440000"},
		{name: "create appointment", method: http.MethodPost, target: "/appointments", body: bytes.NewBufferString(`{"slot_id":"550e8400-e29b-41d4-a716-446655440000","patient_id":"550e8400-e29b-41d4-a716-446655440001","professional_id":"550e8400-e29b-41d4-a716-446655440002"}`)},
		{name: "cancel appointment", method: http.MethodPatch, target: "/appointments/550e8400-e29b-41d4-a716-446655440099/cancel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := &stubDirectoryLookup{}
			server := NewServer(testAppointmentsConfig(), &stubAppointmentsRepository{}, dir)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.target, tt.body))

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
			if dir.currentUserCalls != 0 {
				t.Fatalf("CurrentUser calls = %d, want 0", dir.currentUserCalls)
			}
		})
	}
}

func TestAgendaEndpointsRejectUnsupportedRole(t *testing.T) {
	dir := &stubDirectoryLookup{currentUser: directory.User{ID: "user-1", Role: "assistant", Active: true}}
	server := NewServer(testAppointmentsConfig(), &stubAppointmentsRepository{}, dir)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, "/appointments?professional_id=550e8400-e29b-41d4-a716-446655440000", nil))

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	assertErrorResponse(t, recorder.Body, "insufficient role")
}

func TestDoctorWithoutProfessionalProfileGetsForbidden(t *testing.T) {
	dir := &stubDirectoryLookup{currentUser: directory.User{ID: "user-1", Role: "doctor", Active: true}}
	server := NewServer(testAppointmentsConfig(), &stubAppointmentsRepository{}, dir)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, "/slots", nil))

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	assertErrorResponse(t, recorder.Body, "professional profile required")
}

func TestDoctorCannotCrossProfessionalScope(t *testing.T) {
	doctorProfessionalID := "550e8400-e29b-41d4-a716-446655440010"
	dir := &stubDirectoryLookup{currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true}}
	server := NewServer(testAppointmentsConfig(), &stubAppointmentsRepository{}, dir)
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"slot_id":"550e8400-e29b-41d4-a716-446655440000","patient_id":"550e8400-e29b-41d4-a716-446655440001","professional_id":"550e8400-e29b-41d4-a716-446655440999"}`)

	server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodPost, "/appointments", body))

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	assertErrorResponse(t, recorder.Body, "forbidden professional scope")
}

func TestDoctorCanAccessOwnAgenda(t *testing.T) {
	doctorProfessionalID := "550e8400-e29b-41d4-a716-446655440010"
	createdAt := time.Date(2026, time.April, 10, 9, 0, 0, 0, time.UTC)
	cancelledAt := createdAt.Add(30 * time.Minute)

	t.Run("list slots uses doctor professional scope", func(t *testing.T) {
		repoCalled := false
		repo := &stubAppointmentsRepository{
			listSlotsFn: func(_ context.Context, filters appointments.SlotFilters) ([]appointments.AvailabilitySlot, error) {
				repoCalled = true
				if filters.ProfessionalID != doctorProfessionalID {
					t.Fatalf("filters.ProfessionalID = %q, want %q", filters.ProfessionalID, doctorProfessionalID)
				}
				return []appointments.AvailabilitySlot{{ID: "slot-1", ProfessionalID: doctorProfessionalID, Status: "available"}}, nil
			},
		}
		dir := &stubDirectoryLookup{currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true}}
		server := NewServer(testAppointmentsConfig(), repo, dir)
		recorder := httptest.NewRecorder()

		server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, "/slots?professional_id="+doctorProfessionalID, nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if !repoCalled {
			t.Fatal("expected repo to be called")
		}
	})

	t.Run("create appointment accepts doctor professional scope", func(t *testing.T) {
		repo := &stubAppointmentsRepository{
			createAppointmentFn: func(_ context.Context, params appointments.CreateAppointmentParams) (appointments.Appointment, error) {
				if params.ProfessionalID != doctorProfessionalID {
					t.Fatalf("params.ProfessionalID = %q, want %q", params.ProfessionalID, doctorProfessionalID)
				}
				return appointments.Appointment{
					ID:             "appt-1",
					SlotID:         "550e8400-e29b-41d4-a716-446655440000",
					PatientID:      "550e8400-e29b-41d4-a716-446655440001",
					ProfessionalID: doctorProfessionalID,
					Status:         "booked",
					CreatedAt:      createdAt,
					UpdatedAt:      createdAt,
				}, nil
			},
		}
		dir := &stubDirectoryLookup{
			currentUser:        directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true},
			professionalExists: true,
			patientExists:      true,
		}
		server := NewServer(testAppointmentsConfig(), repo, dir)
		recorder := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"slot_id":"550e8400-e29b-41d4-a716-446655440000","patient_id":"550e8400-e29b-41d4-a716-446655440001","professional_id":"` + doctorProfessionalID + `"}`)

		server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodPost, "/appointments", body))

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
		}
		if dir.professionalCalls != 1 {
			t.Fatalf("ProfessionalExists calls = %d, want 1", dir.professionalCalls)
		}
		if dir.patientCalls != 1 {
			t.Fatalf("PatientExists calls = %d, want 1", dir.patientCalls)
		}
	})

	t.Run("cancel appointment allows doctor owned appointment", func(t *testing.T) {
		repo := &stubAppointmentsRepository{
			getAppointmentByIDFn: func(_ context.Context, appointmentID string) (appointments.Appointment, error) {
				if appointmentID != "550e8400-e29b-41d4-a716-446655440099" {
					t.Fatalf("appointmentID = %q, want owned appointment", appointmentID)
				}
				return appointments.Appointment{ID: appointmentID, ProfessionalID: doctorProfessionalID, Status: "booked"}, nil
			},
			cancelAppointmentFn: func(_ context.Context, appointmentID string) (appointments.Appointment, error) {
				return appointments.Appointment{ID: appointmentID, ProfessionalID: doctorProfessionalID, Status: "cancelled", CancelledAt: &cancelledAt}, nil
			},
		}
		dir := &stubDirectoryLookup{currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true}}
		server := NewServer(testAppointmentsConfig(), repo, dir)
		recorder := httptest.NewRecorder()

		server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodPatch, "/appointments/550e8400-e29b-41d4-a716-446655440099/cancel", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if repo.cancelAppointmentCalls != 1 {
			t.Fatalf("CancelAppointment calls = %d, want 1", repo.cancelAppointmentCalls)
		}
	})
}

func TestSecretaryCanAccessSharedAgenda(t *testing.T) {
	sharedProfessionalID := "550e8400-e29b-41d4-a716-446655440222"
	repoCalled := false
	repo := &stubAppointmentsRepository{
		listAppointmentsFn: func(_ context.Context, filters appointments.AppointmentFilters) ([]appointments.Appointment, error) {
			repoCalled = true
			if filters.ProfessionalID != sharedProfessionalID {
				t.Fatalf("filters.ProfessionalID = %q, want %q", filters.ProfessionalID, sharedProfessionalID)
			}
			return []appointments.Appointment{{ID: "appt-1", ProfessionalID: sharedProfessionalID, Status: "booked"}}, nil
		},
	}
	dir := &stubDirectoryLookup{currentUser: directory.User{ID: "user-2", Role: "secretary", Active: true}}
	server := NewServer(testAppointmentsConfig(), repo, dir)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, "/appointments?professional_id="+sharedProfessionalID, nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !repoCalled {
		t.Fatal("expected repo to be called")
	}
}

func TestAgendaEndpointsFailClosedWhenDirectorySessionValidationFails(t *testing.T) {
	dir := &stubDirectoryLookup{currentUserErr: directory.ErrUnavailable}
	server := NewServer(testAppointmentsConfig(), &stubAppointmentsRepository{}, dir)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, "/appointments", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	assertErrorResponse(t, recorder.Body, "directory service unavailable")
}

func TestDoctorCancelChecksOwnershipBeforeMutation(t *testing.T) {
	doctorProfessionalID := "550e8400-e29b-41d4-a716-446655440010"
	repo := &stubAppointmentsRepository{
		getAppointmentByIDFn: func(context.Context, string) (appointments.Appointment, error) {
			return appointments.Appointment{ID: "550e8400-e29b-41d4-a716-446655440099", ProfessionalID: "550e8400-e29b-41d4-a716-446655440999"}, nil
		},
		cancelAppointmentFn: func(context.Context, string) (appointments.Appointment, error) {
			return appointments.Appointment{}, errors.New("cancel should not be called")
		},
	}
	dir := &stubDirectoryLookup{currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true}}
	server := NewServer(testAppointmentsConfig(), repo, dir)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodPatch, "/appointments/550e8400-e29b-41d4-a716-446655440099/cancel", nil))

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if repo.cancelAppointmentCalls != 0 {
		t.Fatalf("CancelAppointment calls = %d, want 0", repo.cancelAppointmentCalls)
	}
	assertErrorResponse(t, recorder.Body, "forbidden professional scope")
}

type stubAppointmentsRepository struct {
	createSlotsBulkFn      func(context.Context, appointments.BulkCreateSlotsParams) ([]appointments.AvailabilitySlot, error)
	listSlotsFn            func(context.Context, appointments.SlotFilters) ([]appointments.AvailabilitySlot, error)
	createAppointmentFn    func(context.Context, appointments.CreateAppointmentParams) (appointments.Appointment, error)
	listAppointmentsFn     func(context.Context, appointments.AppointmentFilters) ([]appointments.Appointment, error)
	getAppointmentByIDFn   func(context.Context, string) (appointments.Appointment, error)
	cancelAppointmentFn    func(context.Context, string) (appointments.Appointment, error)
	createAppointmentCalls int
	cancelAppointmentCalls int
}

type stubDirectoryLookup struct {
	currentUser        directory.User
	currentUserErr     error
	professionalExists bool
	professionalErr    error
	patientExists      bool
	patientErr         error
	currentUserCalls   int
	professionalCalls  int
	patientCalls       int
}

func (s *stubDirectoryLookup) CurrentUser(context.Context, string) (directory.User, error) {
	s.currentUserCalls++
	if s.currentUserErr != nil {
		return directory.User{}, s.currentUserErr
	}
	if s.currentUser.Role == "" {
		return directory.User{ID: "user-1", Role: "admin", Active: true}, nil
	}
	return s.currentUser, nil
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
	s.cancelAppointmentCalls++
	if s.cancelAppointmentFn == nil {
		return appointments.Appointment{}, errors.New("unexpected CancelAppointment call")
	}
	return s.cancelAppointmentFn(ctx, appointmentID)
}

func (s *stubAppointmentsRepository) GetAppointmentByID(ctx context.Context, appointmentID string) (appointments.Appointment, error) {
	if s.getAppointmentByIDFn == nil {
		return appointments.Appointment{}, errors.New("unexpected GetAppointmentByID call")
	}
	return s.getAppointmentByIDFn(ctx, appointmentID)
}

func newAuthenticatedRequest(method, target string, body io.Reader) *http.Request {
	request := httptest.NewRequest(method, target, body)
	request.Header.Set("Authorization", "Bearer test-token")
	return request
}

func testAppointmentsConfig() Config {
	return Config{ServiceName: "appointments-service", Version: "test", Environment: "test"}
}
