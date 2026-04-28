package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCreateSchedulePostScenarios(t *testing.T) {
	const (
		validProfessionalID = "550e8400-e29b-41d4-a716-446655440002"
		validCreatedBy      = "550e8400-e29b-41d4-a716-446655440099"
	)

	validBody := `{"professional_id":"` + validProfessionalID + `","effective_from":"2026-05-01","recurrence":{"days":[1,3],"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30},"created_by":"` + validCreatedBy + `","reason":"extended hours"}`
	createdAt := time.Date(2026, time.April, 10, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)
	createdBy := validCreatedBy
	createdTemplate := appointments.ScheduleTemplate{
		ID:             "550e8400-e29b-41d4-a716-446655440010",
		ProfessionalID: validProfessionalID,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		Versions: []appointments.ScheduleTemplateVersion{{
			ID:            "550e8400-e29b-41d4-a716-446655440011",
			TemplateID:    "550e8400-e29b-41d4-a716-446655440010",
			VersionNumber: 1,
			EffectiveFrom: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
			Recurrence:    json.RawMessage(`{"days":[1,3],"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}`),
			CreatedAt:     createdAt,
			CreatedBy:     &createdBy,
			Reason:        ptrToString("extended hours"),
		}},
	}

	doctorProfessionalID := validProfessionalID

	tests := []struct {
		name                  string
		body                  string
		directory             stubDirectoryLookup
		repoErr               error
		repoTemplate          appointments.ScheduleTemplate
		wantStatus            int
		wantError             string
		wantTemplate          *appointments.ScheduleTemplate
		wantProfessionalCalls int
		wantCreateRepoCalls   int
	}{
		{
			name:                "invalid json body returns bad request",
			body:                `{"professional_id":`,
			wantStatus:          http.StatusBadRequest,
			wantError:           "invalid json body",
			wantCreateRepoCalls: 0,
		},
		{
			name: "secretary cannot create schedules",
			body: validBody,
			directory: stubDirectoryLookup{
				currentUser: directory.User{ID: "user-1", Role: "secretary", Active: true},
			},
			wantStatus:          http.StatusForbidden,
			wantError:           "insufficient role",
			wantCreateRepoCalls: 0,
		},
		{
			name:                "invalid request returns bad request",
			body:                `{"professional_id":"not-a-uuid","effective_from":"2026-05-01","recurrence":{"days":[1]}}`,
			wantStatus:          http.StatusBadRequest,
			wantError:           "invalid schedule request",
			wantCreateRepoCalls: 0,
		},
		{
			name: "doctor scope is enforced before create",
			body: validBody,
			directory: stubDirectoryLookup{
				currentUser:        directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true},
				professionalExists: true,
			},
			repoTemplate:          createdTemplate,
			wantStatus:            http.StatusCreated,
			wantTemplate:          &createdTemplate,
			wantProfessionalCalls: 1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "directory failure returns service unavailable",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalErr: directory.ErrUnavailable,
			},
			wantStatus:            http.StatusServiceUnavailable,
			wantError:             "directory service unavailable",
			wantProfessionalCalls: 1,
			wantCreateRepoCalls:   0,
		},
		{
			name: "missing professional returns bad request",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: false,
			},
			wantStatus:            http.StatusBadRequest,
			wantError:             "professional not found",
			wantProfessionalCalls: 1,
			wantCreateRepoCalls:   0,
		},
		{
			name: "repo validation error returns bad request",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
			},
			repoErr:               appointments.ErrValidation,
			wantStatus:            http.StatusBadRequest,
			wantError:             "invalid schedule request",
			wantProfessionalCalls: 1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "repo conflict is treated as unexpected create failure",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
			},
			repoErr:               appointments.ErrConflict,
			wantStatus:            http.StatusInternalServerError,
			wantError:             "failed to create schedule",
			wantProfessionalCalls: 1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "success returns created schedule",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
			},
			repoTemplate:          createdTemplate,
			wantStatus:            http.StatusCreated,
			wantTemplate:          &createdTemplate,
			wantProfessionalCalls: 1,
			wantCreateRepoCalls:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.directory
			repo := &stubAppointmentsRepository{
				createTemplateFn: func(_ context.Context, params appointments.CreateTemplateParams) (appointments.ScheduleTemplate, error) {
					if params.ProfessionalID != validProfessionalID {
						t.Fatalf("professional_id = %q, want %q", params.ProfessionalID, validProfessionalID)
					}
					if params.EffectiveFrom != "2026-05-01" {
						t.Fatalf("effective_from = %q, want %q", params.EffectiveFrom, "2026-05-01")
					}
					if string(params.Recurrence) != `{"days":[1,3],"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}` {
						t.Fatalf("recurrence = %s, want expected recurrence", params.Recurrence)
					}
					if params.CreatedBy == nil || *params.CreatedBy != validCreatedBy {
						t.Fatalf("created_by = %v, want %q", params.CreatedBy, validCreatedBy)
					}
					if params.Reason == nil || *params.Reason != "extended hours" {
						t.Fatalf("reason = %v, want %q", params.Reason, "extended hours")
					}
					return tt.repoTemplate, tt.repoErr
				},
			}

			server := NewServer(testAppointmentsConfig(), repo, &dir)
			recorder := httptest.NewRecorder()
			request := newAuthenticatedRequest(http.MethodPost, "/schedules", bytes.NewBufferString(tt.body))

			server.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if dir.professionalCalls != tt.wantProfessionalCalls {
				t.Fatalf("ProfessionalExists calls = %d, want %d", dir.professionalCalls, tt.wantProfessionalCalls)
			}
			if repo.createTemplateCalls != tt.wantCreateRepoCalls {
				t.Fatalf("CreateTemplate calls = %d, want %d", repo.createTemplateCalls, tt.wantCreateRepoCalls)
			}

			if tt.wantError != "" {
				assertErrorResponse(t, recorder.Body, tt.wantError)
				return
			}

			var response appointments.ScheduleTemplate
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.ID != tt.wantTemplate.ID || response.ProfessionalID != tt.wantTemplate.ProfessionalID || len(response.Versions) != len(tt.wantTemplate.Versions) {
				t.Fatalf("response = %+v, want %+v", response, *tt.wantTemplate)
			}
		})
	}
}

func TestCreateSchedulePostReplacesSameEffectiveFromWithoutConflict(t *testing.T) {
	const validProfessionalID = "550e8400-e29b-41d4-a716-446655440002"

	createdAt := time.Date(2026, time.April, 10, 9, 0, 0, 0, time.UTC)
	baseVersion := appointments.ScheduleTemplateVersion{
		ID:            "550e8400-e29b-41d4-a716-446655440011",
		TemplateID:    "550e8400-e29b-41d4-a716-446655440010",
		VersionNumber: 1,
		EffectiveFrom: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:     createdAt,
	}

	recurrences := []json.RawMessage{
		json.RawMessage(`{"days":[1,3],"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}`),
		json.RawMessage(`{"days":[1,3,5],"start_time":"08:00","end_time":"13:00","slot_duration_minutes":30}`),
	}
	var repo *stubAppointmentsRepository
	repo = &stubAppointmentsRepository{
		createTemplateFn: func(_ context.Context, params appointments.CreateTemplateParams) (appointments.ScheduleTemplate, error) {
			if params.ProfessionalID != validProfessionalID {
				t.Fatalf("professional_id = %q, want %q", params.ProfessionalID, validProfessionalID)
			}
			if params.EffectiveFrom != "2026-05-01" {
				t.Fatalf("effective_from = %q, want %q", params.EffectiveFrom, "2026-05-01")
			}

			callIndex := repo.createTemplateCalls - 1
			if callIndex < 0 || callIndex >= len(recurrences) {
				t.Fatalf("CreateTemplate call index = %d, want within recurrence fixtures", callIndex)
			}
			if string(params.Recurrence) != string(recurrences[callIndex]) {
				t.Fatalf("recurrence = %s, want %s", params.Recurrence, recurrences[callIndex])
			}

			version := baseVersion
			version.Recurrence = recurrences[callIndex]
			return appointments.ScheduleTemplate{
				ID:             baseVersion.TemplateID,
				ProfessionalID: validProfessionalID,
				CreatedAt:      createdAt,
				UpdatedAt:      createdAt.Add(time.Duration(callIndex) * time.Minute),
				Versions:       []appointments.ScheduleTemplateVersion{version},
			}, nil
		},
	}
	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{professionalExists: true})

	bodies := []string{
		`{"professional_id":"` + validProfessionalID + `","effective_from":"2026-05-01","recurrence":{"days":[1,3],"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}}`,
		`{"professional_id":"` + validProfessionalID + `","effective_from":"2026-05-01","recurrence":{"days":[1,3,5],"start_time":"08:00","end_time":"13:00","slot_duration_minutes":30}}`,
	}

	for index, body := range bodies {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodPost, "/schedules", bytes.NewBufferString(body)))

		if recorder.Code != http.StatusCreated {
			t.Fatalf("request %d status = %d, want %d; body=%s", index+1, recorder.Code, http.StatusCreated, recorder.Body.String())
		}

		var response appointments.ScheduleTemplate
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("request %d decode response: %v", index+1, err)
		}
		if len(response.Versions) != 1 {
			t.Fatalf("request %d versions len = %d, want 1", index+1, len(response.Versions))
		}
		if response.Versions[0].ID != baseVersion.ID || response.Versions[0].VersionNumber != baseVersion.VersionNumber {
			t.Fatalf("request %d version identity = %s/%d, want %s/%d", index+1, response.Versions[0].ID, response.Versions[0].VersionNumber, baseVersion.ID, baseVersion.VersionNumber)
		}
		if string(response.Versions[0].Recurrence) != string(recurrences[index]) {
			t.Fatalf("request %d recurrence = %s, want %s", index+1, response.Versions[0].Recurrence, recurrences[index])
		}
	}

	if repo.createTemplateCalls != 2 {
		t.Fatalf("CreateTemplate calls = %d, want 2", repo.createTemplateCalls)
	}
}

func TestGetScheduleScenarios(t *testing.T) {
	const validProfessionalID = "550e8400-e29b-41d4-a716-446655440002"

	schedule := appointments.ScheduleTemplate{
		ID:             "550e8400-e29b-41d4-a716-446655440010",
		ProfessionalID: validProfessionalID,
		CreatedAt:      time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.April, 20, 12, 0, 0, 0, time.UTC),
		Versions: []appointments.ScheduleTemplateVersion{{
			ID:            "550e8400-e29b-41d4-a716-446655440011",
			TemplateID:    "550e8400-e29b-41d4-a716-446655440010",
			VersionNumber: 2,
			EffectiveFrom: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
			Recurrence:    json.RawMessage(`{"days":[1,3],"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}`),
		}},
	}

	doctorProfessionalID := validProfessionalID

	tests := []struct {
		name                 string
		target               string
		directory            stubDirectoryLookup
		repoErr              error
		repoSchedule         appointments.ScheduleTemplate
		wantStatus           int
		wantError            string
		wantSchedule         *appointments.ScheduleTemplate
		wantGetActiveCalls   int
		wantGetTemplateCalls int
	}{
		{
			name:               "invalid filters return bad request",
			target:             "/schedules?professional_id=not-a-uuid&effective_date=2026-05-10",
			wantStatus:         http.StatusBadRequest,
			wantError:          "invalid schedule filters",
			wantGetActiveCalls: 1,
		},
		{
			name:   "doctor scope is enforced for get",
			target: "/schedules?effective_date=2026-05-10",
			directory: stubDirectoryLookup{
				currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true},
			},
			repoSchedule:         schedule,
			wantStatus:           http.StatusOK,
			wantSchedule:         &schedule,
			wantGetActiveCalls:   1,
			wantGetTemplateCalls: 1,
		},
		{
			name:               "missing effective date returns bad request",
			target:             "/schedules?professional_id=" + validProfessionalID,
			wantStatus:         http.StatusBadRequest,
			wantError:          "invalid schedule filters",
			wantGetActiveCalls: 1,
		},
		{
			name:               "not found returns not found",
			target:             "/schedules?professional_id=" + validProfessionalID + "&effective_date=2026-05-10",
			repoErr:            appointments.ErrNotFound,
			wantStatus:         http.StatusNotFound,
			wantError:          "schedule not found",
			wantGetActiveCalls: 1,
		},
		{
			name:                 "success returns schedule",
			target:               "/schedules?professional_id=" + validProfessionalID + "&effective_date=2026-05-10",
			repoSchedule:         schedule,
			wantStatus:           http.StatusOK,
			wantSchedule:         &schedule,
			wantGetActiveCalls:   1,
			wantGetTemplateCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubAppointmentsRepository{
				getActiveTemplateFn: func(_ context.Context, professionalID string, effectiveDate string) (appointments.ScheduleTemplateVersion, error) {
					if effectiveDate != "2026-05-10" {
						return appointments.ScheduleTemplateVersion{}, appointments.ErrValidation
					}
					if professionalID != validProfessionalID {
						return appointments.ScheduleTemplateVersion{}, appointments.ErrValidation
					}
					if tt.repoErr != nil {
						return appointments.ScheduleTemplateVersion{}, tt.repoErr
					}
					return appointments.ScheduleTemplateVersion{TemplateID: tt.repoSchedule.ID}, nil
				},
				getTemplateFn: func(_ context.Context, templateID string) (appointments.ScheduleTemplate, error) {
					if tt.repoErr != nil {
						return appointments.ScheduleTemplate{}, tt.repoErr
					}
					if templateID != tt.repoSchedule.ID {
						t.Fatalf("templateID = %q, want %q", templateID, tt.repoSchedule.ID)
					}
					return tt.repoSchedule, nil
				},
			}

			server := NewServer(testAppointmentsConfig(), repo, &tt.directory)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, tt.target, nil))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if repo.getActiveTemplateCalls != tt.wantGetActiveCalls {
				t.Fatalf("GetActiveTemplate calls = %d, want %d", repo.getActiveTemplateCalls, tt.wantGetActiveCalls)
			}
			if repo.getTemplateCalls != tt.wantGetTemplateCalls {
				t.Fatalf("GetTemplate calls = %d, want %d", repo.getTemplateCalls, tt.wantGetTemplateCalls)
			}

			if tt.wantError != "" {
				assertErrorResponse(t, recorder.Body, tt.wantError)
				return
			}

			var response appointments.ScheduleTemplate
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.ID != tt.wantSchedule.ID || response.ProfessionalID != tt.wantSchedule.ProfessionalID || len(response.Versions) != 1 {
				t.Fatalf("response = %+v, want %+v", response, *tt.wantSchedule)
			}
		})
	}
}

func TestGetScheduleVersionsScenarios(t *testing.T) {
	const validTemplateID = "550e8400-e29b-41d4-a716-446655440010"
	const validProfessionalID = "550e8400-e29b-41d4-a716-446655440002"

	versions := []appointments.ScheduleTemplateVersion{{
		ID:            "550e8400-e29b-41d4-a716-446655440021",
		TemplateID:    validTemplateID,
		VersionNumber: 2,
		EffectiveFrom: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
		Recurrence:    json.RawMessage(`{"days":[1,3],"start_time":"08:00","end_time":"12:00","slot_duration_minutes":30}`),
	}, {
		ID:            "550e8400-e29b-41d4-a716-446655440020",
		TemplateID:    validTemplateID,
		VersionNumber: 1,
		EffectiveFrom: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
		Recurrence:    json.RawMessage(`{"days":[1,3],"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}`),
	}}
	template := appointments.ScheduleTemplate{ID: validTemplateID, ProfessionalID: validProfessionalID, Versions: versions}
	doctorProfessionalID := validProfessionalID
	otherProfessionalID := "550e8400-e29b-41d4-a716-446655440099"

	tests := []struct {
		name                 string
		target               string
		directory            stubDirectoryLookup
		repoTemplate         appointments.ScheduleTemplate
		getTemplateErr       error
		listVersionsErr      error
		wantStatus           int
		wantError            string
		wantVersionCount     int
		wantGetTemplateCalls int
		wantListVersionCalls int
	}{
		{
			name:                 "invalid template id returns bad request",
			target:               "/schedules/versions?template_id=bad-id",
			getTemplateErr:       appointments.ErrValidation,
			wantStatus:           http.StatusBadRequest,
			wantError:            "invalid schedule filters",
			wantGetTemplateCalls: 1,
		},
		{
			name:   "doctor scope is enforced for versions",
			target: "/schedules/versions?template_id=" + validTemplateID,
			directory: stubDirectoryLookup{
				currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true},
			},
			repoTemplate:         template,
			wantStatus:           http.StatusOK,
			wantVersionCount:     len(versions),
			wantGetTemplateCalls: 1,
			wantListVersionCalls: 1,
		},
		{
			name:   "doctor cannot access another professionals history",
			target: "/schedules/versions?template_id=" + validTemplateID,
			directory: stubDirectoryLookup{
				currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true},
			},
			repoTemplate:         appointments.ScheduleTemplate{ID: validTemplateID, ProfessionalID: otherProfessionalID, Versions: versions},
			wantStatus:           http.StatusForbidden,
			wantError:            "forbidden professional scope",
			wantGetTemplateCalls: 1,
		},
		{
			name:                 "not found returns not found",
			target:               "/schedules/versions?template_id=" + validTemplateID,
			getTemplateErr:       appointments.ErrNotFound,
			wantStatus:           http.StatusNotFound,
			wantError:            "schedule not found",
			wantGetTemplateCalls: 1,
		},
		{
			name:                 "success returns versions",
			target:               "/schedules/versions?template_id=" + validTemplateID,
			repoTemplate:         template,
			wantStatus:           http.StatusOK,
			wantVersionCount:     len(versions),
			wantGetTemplateCalls: 1,
			wantListVersionCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubAppointmentsRepository{
				getTemplateFn: func(_ context.Context, templateID string) (appointments.ScheduleTemplate, error) {
					if templateID != validTemplateID && tt.getTemplateErr == nil {
						t.Fatalf("templateID = %q, want %q", templateID, validTemplateID)
					}
					if tt.getTemplateErr != nil {
						return appointments.ScheduleTemplate{}, tt.getTemplateErr
					}
					return appointments.ScheduleTemplate{ID: tt.repoTemplate.ID, ProfessionalID: tt.repoTemplate.ProfessionalID}, nil
				},
				listTemplateVersionsFn: func(_ context.Context, templateID string) ([]appointments.ScheduleTemplateVersion, error) {
					if templateID != validTemplateID {
						t.Fatalf("templateID = %q, want %q", templateID, validTemplateID)
					}
					if tt.listVersionsErr != nil {
						return nil, tt.listVersionsErr
					}
					return tt.repoTemplate.Versions, nil
				},
			}

			server := NewServer(testAppointmentsConfig(), repo, &tt.directory)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, tt.target, nil))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if repo.getTemplateCalls != tt.wantGetTemplateCalls {
				t.Fatalf("GetTemplate calls = %d, want %d", repo.getTemplateCalls, tt.wantGetTemplateCalls)
			}
			if repo.listTemplateVersionsCalls != tt.wantListVersionCalls {
				t.Fatalf("ListTemplateVersions calls = %d, want %d", repo.listTemplateVersionsCalls, tt.wantListVersionCalls)
			}

			if tt.wantError != "" {
				assertErrorResponse(t, recorder.Body, tt.wantError)
				return
			}

			var response struct {
				Items []appointments.ScheduleTemplateVersion `json:"items"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(response.Items) != tt.wantVersionCount {
				t.Fatalf("items len = %d, want %d", len(response.Items), tt.wantVersionCount)
			}
			if response.Items[0].VersionNumber != tt.repoTemplate.Versions[0].VersionNumber {
				t.Fatalf("first version = %d, want %d", response.Items[0].VersionNumber, tt.repoTemplate.Versions[0].VersionNumber)
			}
		})
	}
}

func TestCreateBlockPostScenarios(t *testing.T) {
	const validProfessionalID = "550e8400-e29b-41d4-a716-446655440502"
	validTemplateID := "550e8400-e29b-41d4-a716-446655440503"
	doctorProfessionalID := validProfessionalID
	createdAt := time.Date(2026, time.April, 16, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)
	blockDate := time.Date(2026, time.May, 12, 0, 0, 0, 0, time.UTC)
	createdBlock := appointments.ScheduleBlock{
		ID:             "550e8400-e29b-41d4-a716-446655440501",
		ProfessionalID: validProfessionalID,
		Scope:          "single",
		BlockDate:      &blockDate,
		StartTime:      "09:00",
		EndTime:        "12:00",
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
	templateBlock := appointments.ScheduleBlock{
		ID:             "550e8400-e29b-41d4-a716-446655440504",
		ProfessionalID: validProfessionalID,
		Scope:          "template",
		DayOfWeek:      intPtr(1),
		StartTime:      "13:00",
		EndTime:        "15:00",
		TemplateID:     &validTemplateID,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}

	tests := []struct {
		name                  string
		body                  string
		directory             stubDirectoryLookup
		repoErr               error
		repoBlock             appointments.ScheduleBlock
		wantStatus            int
		wantError             string
		wantProfessionalCalls int
		wantCreateRepoCalls   int
	}{
		{
			name:                "invalid json body returns bad request",
			body:                `{"professional_id":`,
			wantStatus:          http.StatusBadRequest,
			wantError:           "invalid json body",
			wantCreateRepoCalls: 0,
		},
		{
			name:                "invalid request returns bad request",
			body:                `{"professional_id":"not-a-uuid","scope":"single","block_date":"2026-05-12","start_time":"09:00","end_time":"12:00"}`,
			wantStatus:          http.StatusBadRequest,
			wantError:           "invalid block request",
			wantCreateRepoCalls: 0,
		},
		{
			name: "doctor scope is enforced before create",
			body: `{"scope":"single","block_date":"2026-05-12","start_time":"09:00","end_time":"12:00"}`,
			directory: stubDirectoryLookup{
				currentUser:        directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true},
				professionalExists: true,
			},
			repoBlock:             createdBlock,
			wantStatus:            http.StatusCreated,
			wantProfessionalCalls: 1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "missing professional returns bad request",
			body: `{"professional_id":"` + validProfessionalID + `","scope":"single","block_date":"2026-05-12","start_time":"09:00","end_time":"12:00"}`,
			directory: stubDirectoryLookup{
				professionalExists: false,
			},
			wantStatus:            http.StatusBadRequest,
			wantError:             "professional not found",
			wantProfessionalCalls: 1,
			wantCreateRepoCalls:   0,
		},
		{
			name: "template scope success returns created block",
			body: `{"professional_id":"` + validProfessionalID + `","scope":"template","day_of_week":1,"start_time":"13:00","end_time":"15:00","template_id":"` + validTemplateID + `"}`,
			directory: stubDirectoryLookup{
				professionalExists: true,
			},
			repoBlock:             templateBlock,
			wantStatus:            http.StatusCreated,
			wantProfessionalCalls: 1,
			wantCreateRepoCalls:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.directory
			repo := &stubAppointmentsRepository{
				createScheduleBlockFn: func(_ context.Context, params appointments.CreateScheduleBlockParams) (appointments.ScheduleBlock, error) {
					if params.ProfessionalID != validProfessionalID {
						t.Fatalf("professional_id = %q, want %q", params.ProfessionalID, validProfessionalID)
					}
					return tt.repoBlock, tt.repoErr
				},
			}

			server := NewServer(testAppointmentsConfig(), repo, &dir)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodPost, "/blocks", bytes.NewBufferString(tt.body)))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if dir.professionalCalls != tt.wantProfessionalCalls {
				t.Fatalf("ProfessionalExists calls = %d, want %d", dir.professionalCalls, tt.wantProfessionalCalls)
			}
			if repo.createScheduleBlockCalls != tt.wantCreateRepoCalls {
				t.Fatalf("CreateScheduleBlock calls = %d, want %d", repo.createScheduleBlockCalls, tt.wantCreateRepoCalls)
			}

			if tt.wantError != "" {
				assertErrorResponse(t, recorder.Body, tt.wantError)
				return
			}

			var response appointments.ScheduleBlock
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.ID != tt.repoBlock.ID || response.Scope != tt.repoBlock.Scope || response.ProfessionalID != tt.repoBlock.ProfessionalID {
				t.Fatalf("response = %+v, want %+v", response, tt.repoBlock)
			}
		})
	}
}

func TestListBlocksScenarios(t *testing.T) {
	const validProfessionalID = "550e8400-e29b-41d4-a716-446655440602"
	doctorProfessionalID := validProfessionalID
	blockDate := time.Date(2026, time.May, 12, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name               string
		target             string
		directory          stubDirectoryLookup
		repoErr            error
		wantStatus         int
		wantError          string
		wantListRepoCalls  int
		wantReturnedItems  int
		wantProfessionalID string
	}{
		{
			name:               "invalid filters return bad request",
			target:             "/blocks?professional_id=bad-id",
			repoErr:            appointments.ErrValidation,
			wantStatus:         http.StatusBadRequest,
			wantError:          "invalid block filters",
			wantListRepoCalls:  1,
			wantProfessionalID: "bad-id",
		},
		{
			name:   "doctor scope is enforced for list",
			target: "/blocks?scope=single",
			directory: stubDirectoryLookup{
				currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true},
			},
			wantStatus:         http.StatusOK,
			wantListRepoCalls:  1,
			wantReturnedItems:  1,
			wantProfessionalID: validProfessionalID,
		},
		{
			name:               "success returns listed blocks",
			target:             "/blocks?professional_id=" + validProfessionalID + "&scope=single",
			wantStatus:         http.StatusOK,
			wantListRepoCalls:  1,
			wantReturnedItems:  1,
			wantProfessionalID: validProfessionalID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubAppointmentsRepository{
				listScheduleBlocksFn: func(_ context.Context, filters appointments.ScheduleBlockFilters) ([]appointments.ScheduleBlock, error) {
					if filters.ProfessionalID != tt.wantProfessionalID {
						t.Fatalf("professional_id = %q, want %q", filters.ProfessionalID, tt.wantProfessionalID)
					}
					if tt.repoErr != nil {
						return nil, tt.repoErr
					}
					return []appointments.ScheduleBlock{{
						ID:             "550e8400-e29b-41d4-a716-446655440601",
						ProfessionalID: validProfessionalID,
						Scope:          "single",
						BlockDate:      &blockDate,
						StartTime:      "09:00",
						EndTime:        "12:00",
					}}, nil
				},
			}

			server := NewServer(testAppointmentsConfig(), repo, &tt.directory)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, tt.target, nil))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if repo.listScheduleBlocksCalls != tt.wantListRepoCalls {
				t.Fatalf("ListScheduleBlocks calls = %d, want %d", repo.listScheduleBlocksCalls, tt.wantListRepoCalls)
			}
			if tt.wantError != "" {
				assertErrorResponse(t, recorder.Body, tt.wantError)
				return
			}

			var response struct {
				Items []appointments.ScheduleBlock `json:"items"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(response.Items) != tt.wantReturnedItems {
				t.Fatalf("items len = %d, want %d", len(response.Items), tt.wantReturnedItems)
			}
		})
	}
}

func TestBlockByIDScenarios(t *testing.T) {
	const validBlockID = "550e8400-e29b-41d4-a716-446655440701"
	const validProfessionalID = "550e8400-e29b-41d4-a716-446655440702"
	const otherProfessionalID = "550e8400-e29b-41d4-a716-446655440799"
	doctorProfessionalID := validProfessionalID
	blockDate := time.Date(2026, time.May, 12, 0, 0, 0, 0, time.UTC)
	storedBlock := appointments.ScheduleBlock{ID: validBlockID, ProfessionalID: validProfessionalID, Scope: "single", BlockDate: &blockDate, StartTime: "09:00", EndTime: "12:00"}
	otherBlock := appointments.ScheduleBlock{ID: validBlockID, ProfessionalID: otherProfessionalID, Scope: "single", BlockDate: &blockDate, StartTime: "09:00", EndTime: "12:00"}

	t.Run("get enforces doctor ownership", func(t *testing.T) {
		repo := &stubAppointmentsRepository{
			getScheduleBlockFn: func(context.Context, string) (appointments.ScheduleBlock, error) {
				return otherBlock, nil
			},
		}
		dir := &stubDirectoryLookup{currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true}}
		server := NewServer(testAppointmentsConfig(), repo, dir)
		recorder := httptest.NewRecorder()

		server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, "/blocks/"+validBlockID, nil))

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}
		assertErrorResponse(t, recorder.Body, "forbidden professional scope")
	})

	t.Run("patch updates block for allowed scope", func(t *testing.T) {
		repo := &stubAppointmentsRepository{
			getScheduleBlockFn: func(context.Context, string) (appointments.ScheduleBlock, error) {
				return storedBlock, nil
			},
			updateScheduleBlockFn: func(_ context.Context, blockID string, params appointments.UpdateScheduleBlockParams) (appointments.ScheduleBlock, error) {
				if blockID != validBlockID {
					t.Fatalf("blockID = %q, want %q", blockID, validBlockID)
				}
				if params.ProfessionalID != validProfessionalID {
					t.Fatalf("professional_id = %q, want %q", params.ProfessionalID, validProfessionalID)
				}
				return storedBlock, nil
			},
		}
		dir := &stubDirectoryLookup{currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true}, professionalExists: true}
		server := NewServer(testAppointmentsConfig(), repo, dir)
		recorder := httptest.NewRecorder()

		server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodPatch, "/blocks/"+validBlockID, bytes.NewBufferString(`{"scope":"single","block_date":"2026-05-12","start_time":"10:00","end_time":"12:00"}`)))

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if repo.getScheduleBlockCalls != 1 {
			t.Fatalf("GetScheduleBlock calls = %d, want 1", repo.getScheduleBlockCalls)
		}
		if repo.updateScheduleBlockCalls != 1 {
			t.Fatalf("UpdateScheduleBlock calls = %d, want 1", repo.updateScheduleBlockCalls)
		}
	})

	t.Run("delete checks ownership before mutation", func(t *testing.T) {
		repo := &stubAppointmentsRepository{
			getScheduleBlockFn: func(context.Context, string) (appointments.ScheduleBlock, error) {
				return otherBlock, nil
			},
			deleteScheduleBlockFn: func(context.Context, string) error {
				return errors.New("delete should not be called")
			},
		}
		dir := &stubDirectoryLookup{currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true}}
		server := NewServer(testAppointmentsConfig(), repo, dir)
		recorder := httptest.NewRecorder()

		server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodDelete, "/blocks/"+validBlockID, nil))

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}
		if repo.deleteScheduleBlockCalls != 0 {
			t.Fatalf("DeleteScheduleBlock calls = %d, want 0", repo.deleteScheduleBlockCalls)
		}
		assertErrorResponse(t, recorder.Body, "forbidden professional scope")
	})
}

func TestCreateConsultationPostScenarios(t *testing.T) {
	const (
		validConsultationID = "550e8400-e29b-41d4-a716-446655440401"
		validSlotID         = "550e8400-e29b-41d4-a716-446655440402"
		validPatientID      = "550e8400-e29b-41d4-a716-446655440403"
		validProfessionalID = "550e8400-e29b-41d4-a716-446655440404"
	)

	validBody := `{"slot_id":"` + validSlotID + `","patient_id":"` + validPatientID + `","professional_id":"` + validProfessionalID + `","source":"secretary","notes":"Paciente con estudios"}`
	standaloneStart := time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC)
	standaloneEnd := time.Date(2026, time.April, 16, 11, 20, 0, 0, time.UTC)
	standaloneBody := `{"patient_id":"` + validPatientID + `","professional_id":"` + validProfessionalID + `","source":"doctor","scheduled_start":"2026-04-16T11:00:00Z","scheduled_end":"2026-04-16T11:20:00Z"}`
	createdAt := time.Date(2026, time.April, 16, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)
	scheduledStart := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, time.April, 16, 10, 30, 0, 0, time.UTC)
	notes := "Paciente con estudios"
	createdConsultation := appointments.Consultation{
		ID:             validConsultationID,
		SlotID:         ptrToString(validSlotID),
		ProfessionalID: validProfessionalID,
		PatientID:      validPatientID,
		Status:         appointments.ConsultationStatusScheduled,
		Source:         appointments.ConsultationSourceSecretary,
		ScheduledStart: scheduledStart,
		ScheduledEnd:   scheduledEnd,
		Notes:          &notes,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
	standaloneConsultation := appointments.Consultation{
		ID:             validConsultationID,
		ProfessionalID: validProfessionalID,
		PatientID:      validPatientID,
		Status:         appointments.ConsultationStatusScheduled,
		Source:         appointments.ConsultationSourceDoctor,
		ScheduledStart: standaloneStart,
		ScheduledEnd:   standaloneEnd,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
	doctorProfessionalID := validProfessionalID

	tests := []struct {
		name                  string
		body                  string
		directory             stubDirectoryLookup
		repoErr               error
		repoConsultation      appointments.Consultation
		wantStatus            int
		wantError             string
		wantConsultation      *appointments.Consultation
		wantProfessionalCalls int
		wantPatientCalls      int
		wantCreateRepoCalls   int
	}{
		{
			name:                "invalid json body returns bad request",
			body:                `{"professional_id":`,
			wantStatus:          http.StatusBadRequest,
			wantError:           "invalid json body",
			wantCreateRepoCalls: 0,
		},
		{
			name:                "invalid request returns bad request",
			body:                `{"professional_id":"not-a-uuid","patient_id":"` + validPatientID + `","source":"secretary"}`,
			wantStatus:          http.StatusBadRequest,
			wantError:           "invalid consultation request",
			wantCreateRepoCalls: 0,
		},
		{
			name:                "standalone consultation requires explicit schedule range",
			body:                `{"professional_id":"` + validProfessionalID + `","patient_id":"` + validPatientID + `","source":"doctor"}`,
			wantStatus:          http.StatusBadRequest,
			wantError:           "invalid consultation request",
			wantCreateRepoCalls: 0,
		},
		{
			name: "doctor scope is enforced before create",
			body: validBody,
			directory: stubDirectoryLookup{
				currentUser:        directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true},
				professionalExists: true,
				patientExists:      true,
			},
			repoConsultation:      createdConsultation,
			wantStatus:            http.StatusCreated,
			wantConsultation:      &createdConsultation,
			wantProfessionalCalls: 1,
			wantPatientCalls:      1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "missing professional returns bad request",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: false,
			},
			wantStatus:            http.StatusBadRequest,
			wantError:             "professional not found",
			wantProfessionalCalls: 1,
			wantCreateRepoCalls:   0,
		},
		{
			name: "missing patient returns bad request",
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
			name: "repo conflict returns conflict",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
				patientExists:      true,
			},
			repoErr:               appointments.ErrConflict,
			wantStatus:            http.StatusConflict,
			wantError:             "consultation conflicts with existing schedule",
			wantProfessionalCalls: 1,
			wantPatientCalls:      1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "standalone success returns created consultation",
			body: standaloneBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
				patientExists:      true,
			},
			repoConsultation:      standaloneConsultation,
			wantStatus:            http.StatusCreated,
			wantConsultation:      &standaloneConsultation,
			wantProfessionalCalls: 1,
			wantPatientCalls:      1,
			wantCreateRepoCalls:   1,
		},
		{
			name: "success returns created consultation",
			body: validBody,
			directory: stubDirectoryLookup{
				professionalExists: true,
				patientExists:      true,
			},
			repoConsultation:      createdConsultation,
			wantStatus:            http.StatusCreated,
			wantConsultation:      &createdConsultation,
			wantProfessionalCalls: 1,
			wantPatientCalls:      1,
			wantCreateRepoCalls:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.directory
			repo := &stubAppointmentsRepository{
				createConsultationFn: func(_ context.Context, params appointments.CreateConsultationParams) (appointments.Consultation, error) {
					if params.ProfessionalID != validProfessionalID {
						t.Fatalf("professional_id = %q, want %q", params.ProfessionalID, validProfessionalID)
					}
					if params.PatientID != validPatientID {
						t.Fatalf("patient_id = %q, want %q", params.PatientID, validPatientID)
					}
					if strings.Contains(tt.body, `"source":"doctor"`) {
						if params.Source != appointments.ConsultationSourceDoctor {
							t.Fatalf("source = %q, want %q", params.Source, appointments.ConsultationSourceDoctor)
						}
						if params.SlotID != nil {
							t.Fatalf("slot_id = %v, want nil", params.SlotID)
						}
						if !timesMatch(params.ScheduledStart, &standaloneStart) {
							t.Fatalf("scheduled_start = %v, want %v", params.ScheduledStart, standaloneStart)
						}
						if !timesMatch(params.ScheduledEnd, &standaloneEnd) {
							t.Fatalf("scheduled_end = %v, want %v", params.ScheduledEnd, standaloneEnd)
						}
					} else {
						if params.Source != appointments.ConsultationSourceSecretary {
							t.Fatalf("source = %q, want %q", params.Source, appointments.ConsultationSourceSecretary)
						}
						if params.SlotID == nil || *params.SlotID != validSlotID {
							t.Fatalf("slot_id = %v, want %q", params.SlotID, validSlotID)
						}
						if params.ScheduledStart != nil || params.ScheduledEnd != nil {
							t.Fatalf("scheduled range = [%v %v], want nil", params.ScheduledStart, params.ScheduledEnd)
						}
						if params.Notes == nil || *params.Notes != notes {
							t.Fatalf("notes = %v, want %q", params.Notes, notes)
						}
					}
					return tt.repoConsultation, tt.repoErr
				},
			}

			server := NewServer(testAppointmentsConfig(), repo, &dir)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodPost, "/consultations", bytes.NewBufferString(tt.body)))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if dir.professionalCalls != tt.wantProfessionalCalls {
				t.Fatalf("ProfessionalExists calls = %d, want %d", dir.professionalCalls, tt.wantProfessionalCalls)
			}
			if dir.patientCalls != tt.wantPatientCalls {
				t.Fatalf("PatientExists calls = %d, want %d", dir.patientCalls, tt.wantPatientCalls)
			}
			if repo.createConsultationCalls != tt.wantCreateRepoCalls {
				t.Fatalf("CreateConsultation calls = %d, want %d", repo.createConsultationCalls, tt.wantCreateRepoCalls)
			}

			if tt.wantError != "" {
				assertErrorResponse(t, recorder.Body, tt.wantError)
				return
			}

			var response appointments.Consultation
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.ID != tt.wantConsultation.ID || response.ProfessionalID != tt.wantConsultation.ProfessionalID || response.PatientID != tt.wantConsultation.PatientID || response.Status != tt.wantConsultation.Status {
				t.Fatalf("response = %+v, want %+v", response, *tt.wantConsultation)
			}
		})
	}
}

func TestPatientRequestPostScenarios(t *testing.T) {
	const (
		validPatientID      = "550e8400-e29b-41d4-a716-446655440503"
		validProfessionalID = "550e8400-e29b-41d4-a716-446655440504"
	)

	createdAt := time.Date(2026, time.April, 16, 9, 0, 0, 0, time.UTC)
	requestedStart := createdAt
	requestedEnd := createdAt.Add(time.Minute)
	notes := "Prefiero turno por la tarde. Contacto: 11-5555"
	createdRequest := appointments.Consultation{
		ID:             "550e8400-e29b-41d4-a716-446655440501",
		ProfessionalID: validProfessionalID,
		PatientID:      validPatientID,
		Status:         appointments.ConsultationStatusRequested,
		Source:         appointments.ConsultationSourcePatient,
		ScheduledStart: requestedStart,
		ScheduledEnd:   requestedEnd,
		Notes:          &notes,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}

	tests := []struct {
		name                 string
		body                 string
		directory            stubDirectoryLookup
		repoErr              error
		wantStatus           int
		wantError            string
		wantDocumentLookups  int
		wantProfessionalHits int
		wantRepoCalls        int
	}{
		{
			name:                 "invalid document returns bad request",
			body:                 `{"document":" ","professional_id":"` + validProfessionalID + `"}`,
			wantStatus:           http.StatusBadRequest,
			wantError:            "invalid patient request",
			wantDocumentLookups:  0,
			wantProfessionalHits: 0,
			wantRepoCalls:        0,
		},
		{
			name: "patient document not found returns not found without leaking profile data",
			body: `{"document":"12345678","professional_id":"` + validProfessionalID + `"}`,
			directory: stubDirectoryLookup{
				patientDocumentErr: directory.ErrNotFound,
			},
			wantStatus:           http.StatusNotFound,
			wantError:            "patient not found",
			wantDocumentLookups:  1,
			wantProfessionalHits: 0,
			wantRepoCalls:        0,
		},
		{
			name: "success creates public requested patient consultation",
			body: `{"document":" 12345678 ","professional_id":"` + validProfessionalID + `","notes":"Prefiero turno por la tarde","contact":"11-5555"}`,
			directory: stubDirectoryLookup{
				professionalExists: true,
				patientByDocument:  directory.Patient{ID: validPatientID, Document: "12345678", Active: true},
			},
			wantStatus:           http.StatusCreated,
			wantDocumentLookups:  1,
			wantProfessionalHits: 1,
			wantRepoCalls:        1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.directory
			repo := &stubAppointmentsRepository{
				createConsultationFn: func(_ context.Context, params appointments.CreateConsultationParams) (appointments.Consultation, error) {
					if params.PatientID != validPatientID {
						t.Fatalf("patient_id = %q, want %q", params.PatientID, validPatientID)
					}
					if params.ProfessionalID != validProfessionalID {
						t.Fatalf("professional_id = %q, want %q", params.ProfessionalID, validProfessionalID)
					}
					if params.Source != appointments.ConsultationSourcePatient {
						t.Fatalf("source = %q, want %q", params.Source, appointments.ConsultationSourcePatient)
					}
					if params.Status != appointments.ConsultationStatusRequested {
						t.Fatalf("status = %q, want %q", params.Status, appointments.ConsultationStatusRequested)
					}
					if params.SlotID != nil {
						t.Fatalf("slot_id = %v, want nil", params.SlotID)
					}
					if params.Notes == nil || *params.Notes != notes {
						t.Fatalf("notes = %v, want %q", params.Notes, notes)
					}
					return createdRequest, tt.repoErr
				},
			}
			server := NewServer(testAppointmentsConfig(), repo, &dir)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/patient-requests", bytes.NewBufferString(tt.body)))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if dir.patientDocumentCalls != tt.wantDocumentLookups {
				t.Fatalf("PatientByDocument calls = %d, want %d", dir.patientDocumentCalls, tt.wantDocumentLookups)
			}
			if dir.professionalCalls != tt.wantProfessionalHits {
				t.Fatalf("ProfessionalExists calls = %d, want %d", dir.professionalCalls, tt.wantProfessionalHits)
			}
			if repo.createConsultationCalls != tt.wantRepoCalls {
				t.Fatalf("CreateConsultation calls = %d, want %d", repo.createConsultationCalls, tt.wantRepoCalls)
			}

			if tt.wantError != "" {
				assertErrorResponse(t, recorder.Body, tt.wantError)
				return
			}

			var response appointments.Consultation
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.PatientID != validPatientID || response.Status != appointments.ConsultationStatusRequested || response.Source != appointments.ConsultationSourcePatient {
				t.Fatalf("response = %+v, want requested patient consultation", response)
			}
			if strings.Contains(recorder.Body.String(), "12345678") {
				t.Fatalf("response leaked patient document: %s", recorder.Body.String())
			}
		})
	}
}

func TestGetConsultationScenarios(t *testing.T) {
	const (
		validConsultationID = "550e8400-e29b-41d4-a716-446655440411"
		validProfessionalID = "550e8400-e29b-41d4-a716-446655440412"
		validPatientID      = "550e8400-e29b-41d4-a716-446655440413"
	)

	otherProfessionalID := "550e8400-e29b-41d4-a716-446655440414"
	doctorProfessionalID := validProfessionalID
	consultation := appointments.Consultation{
		ID:             validConsultationID,
		ProfessionalID: validProfessionalID,
		PatientID:      validPatientID,
		Status:         appointments.ConsultationStatusCheckedIn,
		Source:         appointments.ConsultationSourceOnline,
	}

	tests := []struct {
		name             string
		target           string
		directory        stubDirectoryLookup
		repoErr          error
		repoConsultation appointments.Consultation
		wantStatus       int
		wantError        string
		wantGetRepoCalls int
	}{
		{
			name:             "invalid consultation id returns bad request",
			target:           "/consultations?id=bad-id",
			repoErr:          appointments.ErrValidation,
			wantStatus:       http.StatusBadRequest,
			wantError:        "invalid consultation filters",
			wantGetRepoCalls: 1,
		},
		{
			name:   "doctor cannot read another professional consultation",
			target: "/consultations?id=" + validConsultationID,
			directory: stubDirectoryLookup{
				currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true},
			},
			repoConsultation: appointments.Consultation{ID: validConsultationID, ProfessionalID: otherProfessionalID, PatientID: validPatientID},
			wantStatus:       http.StatusForbidden,
			wantError:        "forbidden professional scope",
			wantGetRepoCalls: 1,
		},
		{
			name:             "not found returns not found",
			target:           "/consultations?id=" + validConsultationID,
			repoErr:          appointments.ErrNotFound,
			wantStatus:       http.StatusNotFound,
			wantError:        "consultation not found",
			wantGetRepoCalls: 1,
		},
		{
			name:             "success returns consultation",
			target:           "/consultations?id=" + validConsultationID,
			repoConsultation: consultation,
			wantStatus:       http.StatusOK,
			wantGetRepoCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubAppointmentsRepository{
				getConsultationFn: func(_ context.Context, consultationID string) (appointments.Consultation, error) {
					if consultationID != validConsultationID && tt.repoErr == nil {
						t.Fatalf("consultationID = %q, want %q", consultationID, validConsultationID)
					}
					if tt.repoErr != nil {
						return appointments.Consultation{}, tt.repoErr
					}
					return tt.repoConsultation, nil
				},
			}

			server := NewServer(testAppointmentsConfig(), repo, &tt.directory)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, tt.target, nil))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if repo.getConsultationCalls != tt.wantGetRepoCalls {
				t.Fatalf("GetConsultation calls = %d, want %d", repo.getConsultationCalls, tt.wantGetRepoCalls)
			}

			if tt.wantError != "" {
				assertErrorResponse(t, recorder.Body, tt.wantError)
				return
			}

			var response appointments.Consultation
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.ID != tt.repoConsultation.ID || response.ProfessionalID != tt.repoConsultation.ProfessionalID || response.Status != tt.repoConsultation.Status {
				t.Fatalf("response = %+v, want %+v", response, tt.repoConsultation)
			}
		})
	}
}

func TestPatchConsultationStatusScenarios(t *testing.T) {
	const (
		validConsultationID = "550e8400-e29b-41d4-a716-446655440421"
		validProfessionalID = "550e8400-e29b-41d4-a716-446655440422"
		validPatientID      = "550e8400-e29b-41d4-a716-446655440423"
	)

	doctorProfessionalID := validProfessionalID
	otherProfessionalID := "550e8400-e29b-41d4-a716-446655440424"
	checkInTime := time.Date(2026, time.April, 16, 10, 25, 0, 0, time.UTC)
	receptionNotes := "Paciente ya en recepción"
	updatedConsultation := appointments.Consultation{
		ID:             validConsultationID,
		ProfessionalID: validProfessionalID,
		PatientID:      validPatientID,
		Status:         appointments.ConsultationStatusCheckedIn,
		Source:         appointments.ConsultationSourceSecretary,
		CheckInTime:    &checkInTime,
		ReceptionNotes: &receptionNotes,
	}

	tests := []struct {
		name            string
		body            string
		directory       stubDirectoryLookup
		lookupErr       error
		lookupResult    appointments.Consultation
		updateErr       error
		updatedResult   appointments.Consultation
		wantStatus      int
		wantError       string
		wantLookupCalls int
		wantUpdateCalls int
	}{
		{
			name:            "invalid json body returns bad request",
			body:            `{"status":`,
			wantStatus:      http.StatusBadRequest,
			wantError:       "invalid json body",
			wantLookupCalls: 0,
			wantUpdateCalls: 0,
		},
		{
			name:            "invalid consultation id returns bad request",
			body:            `{"id":"bad-id","status":"checked_in"}`,
			lookupErr:       appointments.ErrValidation,
			wantStatus:      http.StatusBadRequest,
			wantError:       "invalid consultation status update",
			wantLookupCalls: 1,
			wantUpdateCalls: 0,
		},
		{
			name: "doctor cannot update another professional consultation",
			body: `{"id":"` + validConsultationID + `","status":"checked_in"}`,
			directory: stubDirectoryLookup{
				currentUser: directory.User{ID: "user-1", Role: "doctor", ProfessionalID: &doctorProfessionalID, Active: true},
			},
			lookupResult:    appointments.Consultation{ID: validConsultationID, ProfessionalID: otherProfessionalID, PatientID: validPatientID, Status: appointments.ConsultationStatusScheduled, Source: appointments.ConsultationSourceSecretary},
			wantStatus:      http.StatusForbidden,
			wantError:       "forbidden professional scope",
			wantLookupCalls: 1,
			wantUpdateCalls: 0,
		},
		{
			name: "secretary cannot mark consultation completed",
			body: `{"id":"` + validConsultationID + `","status":"completed"}`,
			directory: stubDirectoryLookup{
				currentUser: directory.User{ID: "user-2", Role: "secretary", Active: true},
			},
			lookupResult:    appointments.Consultation{ID: validConsultationID, ProfessionalID: validProfessionalID, PatientID: validPatientID, Status: appointments.ConsultationStatusCheckedIn, Source: appointments.ConsultationSourceSecretary},
			wantStatus:      http.StatusForbidden,
			wantError:       "insufficient role",
			wantLookupCalls: 1,
			wantUpdateCalls: 0,
		},
		{
			name:            "success returns updated consultation",
			body:            `{"id":"` + validConsultationID + `","status":"checked_in","check_in_time":"2026-04-16T10:25:00Z","reception_notes":"Paciente ya en recepción"}`,
			lookupResult:    appointments.Consultation{ID: validConsultationID, ProfessionalID: validProfessionalID, PatientID: validPatientID, Status: appointments.ConsultationStatusScheduled, Source: appointments.ConsultationSourceSecretary},
			updatedResult:   updatedConsultation,
			wantStatus:      http.StatusOK,
			wantLookupCalls: 2,
			wantUpdateCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubAppointmentsRepository{
				getConsultationFn: func(_ context.Context, consultationID string) (appointments.Consultation, error) {
					if consultationID != validConsultationID && tt.lookupErr == nil {
						t.Fatalf("consultationID = %q, want %q", consultationID, validConsultationID)
					}
					if tt.lookupErr != nil {
						return appointments.Consultation{}, tt.lookupErr
					}
					return tt.lookupResult, nil
				},
				updateConsultationStatusFn: func(_ context.Context, consultationID string, params appointments.UpdateConsultationStatusParams) (appointments.Consultation, error) {
					if consultationID != validConsultationID {
						t.Fatalf("consultationID = %q, want %q", consultationID, validConsultationID)
					}
					if params.Status != appointments.ConsultationStatusCheckedIn {
						t.Fatalf("status = %q, want %q", params.Status, appointments.ConsultationStatusCheckedIn)
					}
					if !timesMatch(params.CheckInTime, &checkInTime) {
						t.Fatalf("check_in_time = %v, want %v", params.CheckInTime, checkInTime)
					}
					if params.ReceptionNotes == nil || *params.ReceptionNotes != receptionNotes {
						t.Fatalf("reception_notes = %v, want %q", params.ReceptionNotes, receptionNotes)
					}
					return tt.updatedResult, tt.updateErr
				},
			}

			server := NewServer(testAppointmentsConfig(), repo, &tt.directory)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodPatch, "/consultations", bytes.NewBufferString(tt.body)))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if repo.getConsultationCalls != tt.wantLookupCalls {
				t.Fatalf("GetConsultation calls = %d, want %d", repo.getConsultationCalls, tt.wantLookupCalls)
			}
			if repo.updateConsultationStatusCalls != tt.wantUpdateCalls {
				t.Fatalf("UpdateConsultationStatus calls = %d, want %d", repo.updateConsultationStatusCalls, tt.wantUpdateCalls)
			}

			if tt.wantError != "" {
				assertErrorResponse(t, recorder.Body, tt.wantError)
				return
			}

			var response appointments.Consultation
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.ID != tt.updatedResult.ID || response.Status != tt.updatedResult.Status {
				t.Fatalf("response = %+v, want %+v", response, tt.updatedResult)
			}
		})
	}
}

func TestGetAgendaWeekReturnsMergedAgenda(t *testing.T) {
	t.Parallel()

	const professionalID = "550e8400-e29b-41d4-a716-446655440010"
	templateID := "550e8400-e29b-41d4-a716-446655440011"
	weekStart := time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC)
	versionRecurrence := json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}}`)
	version := appointments.ScheduleTemplateVersion{
		ID:            "550e8400-e29b-41d4-a716-446655440012",
		TemplateID:    templateID,
		VersionNumber: 1,
		EffectiveFrom: weekStart,
		Recurrence:    versionRecurrence,
		CreatedAt:     weekStart.Add(-24 * time.Hour),
	}
	blockDate := weekStart
	consultationStart := weekStart.Add(9 * time.Hour).Add(15 * time.Minute)
	consultationEnd := consultationStart.Add(15 * time.Minute)

	repo := &stubAppointmentsRepository{
		getActiveTemplateFn: func(_ context.Context, gotProfessionalID, effectiveDate string) (appointments.ScheduleTemplateVersion, error) {
			if gotProfessionalID != professionalID {
				t.Fatalf("professionalID = %q, want %q", gotProfessionalID, professionalID)
			}
			if effectiveDate != "2026-04-12" {
				t.Fatalf("effectiveDate = %q, want %q", effectiveDate, "2026-04-12")
			}
			return version, nil
		},
		getTemplateFn: func(_ context.Context, gotTemplateID string) (appointments.ScheduleTemplate, error) {
			if gotTemplateID != templateID {
				t.Fatalf("templateID = %q, want %q", gotTemplateID, templateID)
			}
			return appointments.ScheduleTemplate{
				ID:             templateID,
				ProfessionalID: professionalID,
				CreatedAt:      weekStart.Add(-48 * time.Hour),
				UpdatedAt:      weekStart.Add(-24 * time.Hour),
			}, nil
		},
		listTemplateVersionsFn: func(_ context.Context, gotTemplateID string) ([]appointments.ScheduleTemplateVersion, error) {
			if gotTemplateID != templateID {
				t.Fatalf("templateID = %q, want %q", gotTemplateID, templateID)
			}
			return []appointments.ScheduleTemplateVersion{version}, nil
		},
		listScheduleBlocksFn: func(_ context.Context, filters appointments.ScheduleBlockFilters) ([]appointments.ScheduleBlock, error) {
			if filters.ProfessionalID != professionalID {
				t.Fatalf("filters.ProfessionalID = %q, want %q", filters.ProfessionalID, professionalID)
			}
			return []appointments.ScheduleBlock{{
				ID:             "550e8400-e29b-41d4-a716-446655440013",
				ProfessionalID: professionalID,
				Scope:          "single",
				BlockDate:      &blockDate,
				StartTime:      "09:30",
				EndTime:        "10:00",
				CreatedAt:      weekStart.Add(-12 * time.Hour),
				UpdatedAt:      weekStart.Add(-12 * time.Hour),
			}}, nil
		},
		listConsultationsFn: func(_ context.Context, filters appointments.ConsultationFilters) ([]appointments.Consultation, error) {
			if filters.ProfessionalID != professionalID {
				t.Fatalf("filters.ProfessionalID = %q, want %q", filters.ProfessionalID, professionalID)
			}
			if filters.WeekStart != "2026-04-06" {
				t.Fatalf("filters.WeekStart = %q, want %q", filters.WeekStart, "2026-04-06")
			}
			return []appointments.Consultation{{
				ID:             "550e8400-e29b-41d4-a716-446655440014",
				ProfessionalID: professionalID,
				PatientID:      "550e8400-e29b-41d4-a716-446655440015",
				Status:         appointments.ConsultationStatusScheduled,
				Source:         appointments.ConsultationSourceSecretary,
				ScheduledStart: consultationStart,
				ScheduledEnd:   consultationEnd,
				CreatedAt:      weekStart.Add(-6 * time.Hour),
				UpdatedAt:      weekStart.Add(-6 * time.Hour),
			}}, nil
		},
	}

	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{})
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, "/agenda/week?professional_id="+professionalID+"&week_start=2026-04-06", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if repo.getActiveTemplateCalls != 1 {
		t.Fatalf("GetActiveTemplate calls = %d, want 1", repo.getActiveTemplateCalls)
	}
	if repo.getTemplateCalls != 1 {
		t.Fatalf("GetTemplate calls = %d, want 1", repo.getTemplateCalls)
	}
	if repo.listTemplateVersionsCalls != 1 {
		t.Fatalf("ListTemplateVersions calls = %d, want 1", repo.listTemplateVersionsCalls)
	}
	if repo.listScheduleBlocksCalls != 1 {
		t.Fatalf("ListScheduleBlocks calls = %d, want 1", repo.listScheduleBlocksCalls)
	}
	if repo.listConsultationsCalls != 1 {
		t.Fatalf("ListConsultations calls = %d, want 1", repo.listConsultationsCalls)
	}

	var response appointments.WeekAgenda
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ProfessionalID != professionalID {
		t.Fatalf("professional_id = %q, want %q", response.ProfessionalID, professionalID)
	}
	if response.WeekStart != "2026-04-06" {
		t.Fatalf("week_start = %q, want %q", response.WeekStart, "2026-04-06")
	}
	if len(response.Templates) != 1 || len(response.Templates[0].Versions) != 1 {
		t.Fatalf("templates = %+v, want one template with one version", response.Templates)
	}
	if len(response.Blocks) != 1 {
		t.Fatalf("blocks len = %d, want 1", len(response.Blocks))
	}
	if len(response.Consultations) != 1 {
		t.Fatalf("consultations len = %d, want 1", len(response.Consultations))
	}
	if len(response.Slots) != 1 {
		t.Fatalf("slots len = %d, want 1", len(response.Slots))
	}
}

func TestGetAgendaWeekRejectsInvalidFilters(t *testing.T) {
	t.Parallel()

	server := NewServer(testAppointmentsConfig(), &stubAppointmentsRepository{}, &stubDirectoryLookup{})
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, "/agenda/week?professional_id=bad-id&week_start=not-a-date", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	assertErrorResponse(t, recorder.Body, "invalid agenda week filters")
}

func TestGetAgendaWeekReturnsEmptyScheduleWhenNoTemplateExists(t *testing.T) {
	t.Parallel()

	const professionalID = "550e8400-e29b-41d4-a716-446655440020"
	weekStart := time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC)
	repo := &stubAppointmentsRepository{
		getActiveTemplateFn: func(context.Context, string, string) (appointments.ScheduleTemplateVersion, error) {
			return appointments.ScheduleTemplateVersion{}, appointments.ErrNotFound
		},
		listScheduleBlocksFn: func(context.Context, appointments.ScheduleBlockFilters) ([]appointments.ScheduleBlock, error) {
			return nil, nil
		},
		listConsultationsFn: func(context.Context, appointments.ConsultationFilters) ([]appointments.Consultation, error) {
			return []appointments.Consultation{{
				ID:             "550e8400-e29b-41d4-a716-446655440021",
				ProfessionalID: professionalID,
				PatientID:      "550e8400-e29b-41d4-a716-446655440022",
				Status:         appointments.ConsultationStatusScheduled,
				Source:         appointments.ConsultationSourceDoctor,
				ScheduledStart: weekStart.Add(11 * time.Hour),
				ScheduledEnd:   weekStart.Add(11*time.Hour + 20*time.Minute),
			}}, nil
		},
	}

	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{})
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, newAuthenticatedRequest(http.MethodGet, "/agenda/week?professional_id="+professionalID+"&week_start=2026-04-06", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if repo.getTemplateCalls != 0 {
		t.Fatalf("GetTemplate calls = %d, want 0", repo.getTemplateCalls)
	}
	if repo.listTemplateVersionsCalls != 0 {
		t.Fatalf("ListTemplateVersions calls = %d, want 0", repo.listTemplateVersionsCalls)
	}

	var response appointments.WeekAgenda
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Templates) != 0 {
		t.Fatalf("templates len = %d, want 0", len(response.Templates))
	}
	if len(response.Slots) != 0 {
		t.Fatalf("slots len = %d, want 0", len(response.Slots))
	}
	if len(response.Consultations) != 1 {
		t.Fatalf("consultations len = %d, want 1", len(response.Consultations))
	}
}

func TestGetPublicAvailabilityReturnsOnlySafeAvailableSlots(t *testing.T) {
	t.Parallel()

	const professionalID = "550e8400-e29b-41d4-a716-446655440030"
	templateID := "550e8400-e29b-41d4-a716-446655440031"
	weekStart := time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC)
	version := appointments.ScheduleTemplateVersion{
		ID:            "550e8400-e29b-41d4-a716-446655440032",
		TemplateID:    templateID,
		VersionNumber: 1,
		EffectiveFrom: weekStart,
		Recurrence:    json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}}`),
		CreatedAt:     weekStart.Add(-24 * time.Hour),
	}
	occupiedStart := weekStart.Add(9 * time.Hour)
	occupiedEnd := occupiedStart.Add(30 * time.Minute)
	privateNote := "private note"

	repo := &stubAppointmentsRepository{
		getActiveTemplateFn: func(_ context.Context, gotProfessionalID, effectiveDate string) (appointments.ScheduleTemplateVersion, error) {
			if gotProfessionalID != professionalID {
				t.Fatalf("professionalID = %q, want %q", gotProfessionalID, professionalID)
			}
			if effectiveDate != "2026-04-12" {
				t.Fatalf("effectiveDate = %q, want %q", effectiveDate, "2026-04-12")
			}
			return version, nil
		},
		getTemplateFn: func(_ context.Context, gotTemplateID string) (appointments.ScheduleTemplate, error) {
			if gotTemplateID != templateID {
				t.Fatalf("templateID = %q, want %q", gotTemplateID, templateID)
			}
			return appointments.ScheduleTemplate{ID: templateID, ProfessionalID: professionalID, CreatedAt: weekStart.Add(-48 * time.Hour), UpdatedAt: weekStart.Add(-24 * time.Hour)}, nil
		},
		listTemplateVersionsFn: func(_ context.Context, gotTemplateID string) ([]appointments.ScheduleTemplateVersion, error) {
			if gotTemplateID != templateID {
				t.Fatalf("templateID = %q, want %q", gotTemplateID, templateID)
			}
			return []appointments.ScheduleTemplateVersion{version}, nil
		},
		listScheduleBlocksFn: func(_ context.Context, filters appointments.ScheduleBlockFilters) ([]appointments.ScheduleBlock, error) {
			if filters.ProfessionalID != professionalID {
				t.Fatalf("filters.ProfessionalID = %q, want %q", filters.ProfessionalID, professionalID)
			}
			return nil, nil
		},
		listConsultationsFn: func(_ context.Context, filters appointments.ConsultationFilters) ([]appointments.Consultation, error) {
			if filters.ProfessionalID != professionalID || filters.WeekStart != "2026-04-06" {
				t.Fatalf("filters = %+v, want professional %q week 2026-04-06", filters, professionalID)
			}
			return []appointments.Consultation{
				{
					ID:             "550e8400-e29b-41d4-a716-446655440033",
					ProfessionalID: professionalID,
					PatientID:      "550e8400-e29b-41d4-a716-446655440034",
					Status:         appointments.ConsultationStatusScheduled,
					Source:         appointments.ConsultationSourceSecretary,
					ScheduledStart: occupiedStart,
					ScheduledEnd:   occupiedEnd,
					Notes:          &privateNote,
				},
				{
					ID:             "550e8400-e29b-41d4-a716-446655440036",
					ProfessionalID: professionalID,
					PatientID:      "550e8400-e29b-41d4-a716-446655440037",
					Status:         appointments.ConsultationStatusRequested,
					Source:         appointments.ConsultationSourcePatient,
					ScheduledStart: weekStart.Add(9*time.Hour + 30*time.Minute),
					ScheduledEnd:   weekStart.Add(9*time.Hour + 31*time.Minute),
				},
			}, nil
		},
		listSlotsFn: func(_ context.Context, filters appointments.SlotFilters) ([]appointments.AvailabilitySlot, error) {
			if filters.ProfessionalID != professionalID || filters.Status != "available" {
				t.Fatalf("slot filters = %+v, want professional %q available", filters, professionalID)
			}
			if filters.Date == "2026-04-06" {
				return []appointments.AvailabilitySlot{{
					ID:             "550e8400-e29b-41d4-a716-446655440035",
					ProfessionalID: professionalID,
					StartTime:      weekStart.Add(9*time.Hour + 30*time.Minute),
					EndTime:        weekStart.Add(10 * time.Hour),
					Status:         "available",
				}}, nil
			}
			return nil, nil
		},
	}

	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{})
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/public/availability?professional_id="+professionalID+"&week_start=2026-04-06", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Items []struct {
			ID             string `json:"id,omitempty"`
			ProfessionalID string `json:"professional_id"`
			StartTime      string `json:"start_time"`
			EndTime        string `json:"end_time"`
		} `json:"items"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items len = %d, want 1: %+v", len(response.Items), response.Items)
	}
	if response.Items[0].ID != "550e8400-e29b-41d4-a716-446655440035" || response.Items[0].ProfessionalID != professionalID || response.Items[0].StartTime != "2026-04-06T09:30:00Z" || response.Items[0].EndTime != "2026-04-06T10:00:00Z" {
		t.Fatalf("item = %+v, want only the free 09:30 slot", response.Items[0])
	}
	body := recorder.Body.String()
	for _, forbidden := range []string{"consultations", "patient_id", "notes", "private note", "blocks", "templates", "status"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public availability leaked %q in body: %s", forbidden, body)
		}
	}
}

func TestPatientRequestDirectBookingCreatesScheduledConsultation(t *testing.T) {
	const (
		validPatientID      = "550e8400-e29b-41d4-a716-446655440513"
		validProfessionalID = "550e8400-e29b-41d4-a716-446655440514"
	)
	scheduledStart := time.Date(2026, time.April, 20, 14, 0, 0, 0, time.UTC)
	scheduledEnd := scheduledStart.Add(30 * time.Minute)
	created := appointments.Consultation{
		ID:             "550e8400-e29b-41d4-a716-446655440515",
		ProfessionalID: validProfessionalID,
		PatientID:      validPatientID,
		Status:         appointments.ConsultationStatusScheduled,
		Source:         appointments.ConsultationSourcePatient,
		ScheduledStart: scheduledStart,
		ScheduledEnd:   scheduledEnd,
	}
	dir := stubDirectoryLookup{
		professionalExists: true,
		patientByDocument:  directory.Patient{ID: validPatientID, Document: "12345678", Active: true},
	}
	repo := &stubAppointmentsRepository{
		createConsultationFn: func(_ context.Context, params appointments.CreateConsultationParams) (appointments.Consultation, error) {
			if params.PatientID != validPatientID || params.ProfessionalID != validProfessionalID {
				t.Fatalf("params patient/professional = %q/%q", params.PatientID, params.ProfessionalID)
			}
			if params.Status != appointments.ConsultationStatusScheduled {
				t.Fatalf("status = %q, want %q", params.Status, appointments.ConsultationStatusScheduled)
			}
			if params.Source != appointments.ConsultationSourcePatient {
				t.Fatalf("source = %q, want %q", params.Source, appointments.ConsultationSourcePatient)
			}
			if params.ScheduledStart == nil || !params.ScheduledStart.Equal(scheduledStart) {
				t.Fatalf("scheduled_start = %v, want %v", params.ScheduledStart, scheduledStart)
			}
			if params.ScheduledEnd == nil || !params.ScheduledEnd.Equal(scheduledEnd) {
				t.Fatalf("scheduled_end = %v, want %v", params.ScheduledEnd, scheduledEnd)
			}
			return created, nil
		},
	}

	server := NewServer(testAppointmentsConfig(), repo, &dir)
	recorder := httptest.NewRecorder()
	body := `{"document":"12345678","professional_id":"` + validProfessionalID + `","scheduled_start":"2026-04-20T14:00:00Z","scheduled_end":"2026-04-20T14:30:00Z"}`

	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/patient-requests", bytes.NewBufferString(body)))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if repo.createConsultationCalls != 1 {
		t.Fatalf("CreateConsultation calls = %d, want 1", repo.createConsultationCalls)
	}
	var response appointments.Consultation
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != appointments.ConsultationStatusScheduled || response.Source != appointments.ConsultationSourcePatient {
		t.Fatalf("response = %+v, want scheduled patient consultation", response)
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
	createSlotsBulkFn             func(context.Context, appointments.BulkCreateSlotsParams) ([]appointments.AvailabilitySlot, error)
	listSlotsFn                   func(context.Context, appointments.SlotFilters) ([]appointments.AvailabilitySlot, error)
	createTemplateFn              func(context.Context, appointments.CreateTemplateParams) (appointments.ScheduleTemplate, error)
	getActiveTemplateFn           func(context.Context, string, string) (appointments.ScheduleTemplateVersion, error)
	getTemplateFn                 func(context.Context, string) (appointments.ScheduleTemplate, error)
	listTemplateVersionsFn        func(context.Context, string) ([]appointments.ScheduleTemplateVersion, error)
	createScheduleBlockFn         func(context.Context, appointments.CreateScheduleBlockParams) (appointments.ScheduleBlock, error)
	getScheduleBlockFn            func(context.Context, string) (appointments.ScheduleBlock, error)
	listScheduleBlocksFn          func(context.Context, appointments.ScheduleBlockFilters) ([]appointments.ScheduleBlock, error)
	updateScheduleBlockFn         func(context.Context, string, appointments.UpdateScheduleBlockParams) (appointments.ScheduleBlock, error)
	deleteScheduleBlockFn         func(context.Context, string) error
	createConsultationFn          func(context.Context, appointments.CreateConsultationParams) (appointments.Consultation, error)
	getConsultationFn             func(context.Context, string) (appointments.Consultation, error)
	listConsultationsFn           func(context.Context, appointments.ConsultationFilters) ([]appointments.Consultation, error)
	updateConsultationStatusFn    func(context.Context, string, appointments.UpdateConsultationStatusParams) (appointments.Consultation, error)
	createAppointmentFn           func(context.Context, appointments.CreateAppointmentParams) (appointments.Appointment, error)
	listAppointmentsFn            func(context.Context, appointments.AppointmentFilters) ([]appointments.Appointment, error)
	getAppointmentByIDFn          func(context.Context, string) (appointments.Appointment, error)
	cancelAppointmentFn           func(context.Context, string) (appointments.Appointment, error)
	createTemplateCalls           int
	getActiveTemplateCalls        int
	getTemplateCalls              int
	listTemplateVersionsCalls     int
	createScheduleBlockCalls      int
	getScheduleBlockCalls         int
	listScheduleBlocksCalls       int
	updateScheduleBlockCalls      int
	deleteScheduleBlockCalls      int
	createConsultationCalls       int
	getConsultationCalls          int
	listConsultationsCalls        int
	updateConsultationStatusCalls int
	createAppointmentCalls        int
	cancelAppointmentCalls        int
}

type stubDirectoryLookup struct {
	currentUser          directory.User
	currentUserErr       error
	professionalExists   bool
	professionalErr      error
	patientExists        bool
	patientErr           error
	patientByDocument    directory.Patient
	patientDocumentErr   error
	currentUserCalls     int
	professionalCalls    int
	patientCalls         int
	patientDocumentCalls int
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

func (s *stubDirectoryLookup) PatientByDocument(context.Context, string) (directory.Patient, error) {
	s.patientDocumentCalls++
	if s.patientDocumentErr != nil {
		return directory.Patient{}, s.patientDocumentErr
	}
	return s.patientByDocument, nil
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

func (s *stubAppointmentsRepository) CreateTemplate(ctx context.Context, params appointments.CreateTemplateParams) (appointments.ScheduleTemplate, error) {
	s.createTemplateCalls++
	if s.createTemplateFn == nil {
		return appointments.ScheduleTemplate{}, errors.New("unexpected CreateTemplate call")
	}
	return s.createTemplateFn(ctx, params)
}

func (s *stubAppointmentsRepository) GetActiveTemplate(ctx context.Context, professionalID string, effectiveDate string) (appointments.ScheduleTemplateVersion, error) {
	s.getActiveTemplateCalls++
	if s.getActiveTemplateFn == nil {
		return appointments.ScheduleTemplateVersion{}, errors.New("unexpected GetActiveTemplate call")
	}
	return s.getActiveTemplateFn(ctx, professionalID, effectiveDate)
}

func (s *stubAppointmentsRepository) GetTemplate(ctx context.Context, templateID string) (appointments.ScheduleTemplate, error) {
	s.getTemplateCalls++
	if s.getTemplateFn == nil {
		return appointments.ScheduleTemplate{}, errors.New("unexpected GetTemplate call")
	}
	return s.getTemplateFn(ctx, templateID)
}

func (s *stubAppointmentsRepository) ListTemplateVersions(ctx context.Context, templateID string) ([]appointments.ScheduleTemplateVersion, error) {
	s.listTemplateVersionsCalls++
	if s.listTemplateVersionsFn == nil {
		return nil, errors.New("unexpected ListTemplateVersions call")
	}
	return s.listTemplateVersionsFn(ctx, templateID)
}

func (s *stubAppointmentsRepository) CreateScheduleBlock(ctx context.Context, params appointments.CreateScheduleBlockParams) (appointments.ScheduleBlock, error) {
	s.createScheduleBlockCalls++
	if s.createScheduleBlockFn == nil {
		return appointments.ScheduleBlock{}, errors.New("unexpected CreateScheduleBlock call")
	}
	return s.createScheduleBlockFn(ctx, params)
}

func (s *stubAppointmentsRepository) GetScheduleBlock(ctx context.Context, blockID string) (appointments.ScheduleBlock, error) {
	s.getScheduleBlockCalls++
	if s.getScheduleBlockFn == nil {
		return appointments.ScheduleBlock{}, errors.New("unexpected GetScheduleBlock call")
	}
	return s.getScheduleBlockFn(ctx, blockID)
}

func (s *stubAppointmentsRepository) ListScheduleBlocks(ctx context.Context, filters appointments.ScheduleBlockFilters) ([]appointments.ScheduleBlock, error) {
	s.listScheduleBlocksCalls++
	if s.listScheduleBlocksFn == nil {
		return nil, errors.New("unexpected ListScheduleBlocks call")
	}
	return s.listScheduleBlocksFn(ctx, filters)
}

func (s *stubAppointmentsRepository) UpdateScheduleBlock(ctx context.Context, blockID string, params appointments.UpdateScheduleBlockParams) (appointments.ScheduleBlock, error) {
	s.updateScheduleBlockCalls++
	if s.updateScheduleBlockFn == nil {
		return appointments.ScheduleBlock{}, errors.New("unexpected UpdateScheduleBlock call")
	}
	return s.updateScheduleBlockFn(ctx, blockID, params)
}

func (s *stubAppointmentsRepository) DeleteScheduleBlock(ctx context.Context, blockID string) error {
	s.deleteScheduleBlockCalls++
	if s.deleteScheduleBlockFn == nil {
		return errors.New("unexpected DeleteScheduleBlock call")
	}
	return s.deleteScheduleBlockFn(ctx, blockID)
}

func (s *stubAppointmentsRepository) CreateConsultation(ctx context.Context, params appointments.CreateConsultationParams) (appointments.Consultation, error) {
	s.createConsultationCalls++
	if s.createConsultationFn == nil {
		return appointments.Consultation{}, errors.New("unexpected CreateConsultation call")
	}
	return s.createConsultationFn(ctx, params)
}

func (s *stubAppointmentsRepository) GetConsultation(ctx context.Context, consultationID string) (appointments.Consultation, error) {
	s.getConsultationCalls++
	if s.getConsultationFn == nil {
		return appointments.Consultation{}, errors.New("unexpected GetConsultation call")
	}
	return s.getConsultationFn(ctx, consultationID)
}

func (s *stubAppointmentsRepository) ListConsultations(ctx context.Context, filters appointments.ConsultationFilters) ([]appointments.Consultation, error) {
	s.listConsultationsCalls++
	if s.listConsultationsFn == nil {
		return nil, errors.New("unexpected ListConsultations call")
	}
	return s.listConsultationsFn(ctx, filters)
}

func (s *stubAppointmentsRepository) UpdateConsultationStatus(ctx context.Context, consultationID string, params appointments.UpdateConsultationStatusParams) (appointments.Consultation, error) {
	s.updateConsultationStatusCalls++
	if s.updateConsultationStatusFn == nil {
		return appointments.Consultation{}, errors.New("unexpected UpdateConsultationStatus call")
	}
	return s.updateConsultationStatusFn(ctx, consultationID, params)
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

func ptrToString(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func timesMatch(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return left.Equal(*right)
}
