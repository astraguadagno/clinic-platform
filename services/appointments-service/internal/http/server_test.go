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

func TestCreateAppointmentReturnsConflict(t *testing.T) {
	repoCalled := false
	repo := &stubAppointmentsRepository{
		createAppointmentFn: func(context.Context, appointments.CreateAppointmentParams) (appointments.Appointment, error) {
			repoCalled = true
			return appointments.Appointment{}, appointments.ErrConflict
		},
	}

	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{professionalExists: true, patientExists: true})
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"slot_id":"550e8400-e29b-41d4-a716-446655440000","patient_id":"550e8400-e29b-41d4-a716-446655440001","professional_id":"550e8400-e29b-41d4-a716-446655440002"}`)
	request := httptest.NewRequest(http.MethodPost, "/appointments", body)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	if !repoCalled {
		t.Fatal("expected repo to be called")
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

func TestCreateAppointmentReturnsBadRequestWhenPatientMissing(t *testing.T) {
	repo := &stubAppointmentsRepository{}
	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{professionalExists: true, patientExists: false})
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"slot_id":"550e8400-e29b-41d4-a716-446655440000","patient_id":"550e8400-e29b-41d4-a716-446655440001","professional_id":"550e8400-e29b-41d4-a716-446655440002"}`)
	request := httptest.NewRequest(http.MethodPost, "/appointments", body)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestCreateAppointmentReturnsBadRequestWhenProfessionalMissing(t *testing.T) {
	repo := &stubAppointmentsRepository{}
	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{professionalExists: false})
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"slot_id":"550e8400-e29b-41d4-a716-446655440000","patient_id":"550e8400-e29b-41d4-a716-446655440001","professional_id":"550e8400-e29b-41d4-a716-446655440002"}`)
	request := httptest.NewRequest(http.MethodPost, "/appointments", body)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestCreateAppointmentReturnsServiceUnavailableWhenDirectoryFails(t *testing.T) {
	repo := &stubAppointmentsRepository{}
	server := NewServer(testAppointmentsConfig(), repo, &stubDirectoryLookup{professionalExists: true, patientErr: directory.ErrUnavailable})
	recorder := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"slot_id":"550e8400-e29b-41d4-a716-446655440000","patient_id":"550e8400-e29b-41d4-a716-446655440001","professional_id":"550e8400-e29b-41d4-a716-446655440002"}`)
	request := httptest.NewRequest(http.MethodPost, "/appointments", body)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

type stubAppointmentsRepository struct {
	createSlotsBulkFn   func(context.Context, appointments.BulkCreateSlotsParams) ([]appointments.AvailabilitySlot, error)
	listSlotsFn         func(context.Context, appointments.SlotFilters) ([]appointments.AvailabilitySlot, error)
	createAppointmentFn func(context.Context, appointments.CreateAppointmentParams) (appointments.Appointment, error)
	listAppointmentsFn  func(context.Context, appointments.AppointmentFilters) ([]appointments.Appointment, error)
	cancelAppointmentFn func(context.Context, string) (appointments.Appointment, error)
}

type stubDirectoryLookup struct {
	professionalExists bool
	professionalErr    error
	patientExists      bool
	patientErr         error
}

func (s *stubDirectoryLookup) ProfessionalExists(context.Context, string) (bool, error) {
	if s.professionalErr != nil {
		return false, s.professionalErr
	}
	return s.professionalExists, nil
}

func (s *stubDirectoryLookup) PatientExists(context.Context, string) (bool, error) {
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
	if s.createAppointmentFn == nil {
		return appointments.Appointment{}, errors.New("unexpected CreateAppointment call")
	}
	return s.createAppointmentFn(ctx, params)
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
