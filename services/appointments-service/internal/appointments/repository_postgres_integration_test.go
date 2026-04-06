package appointments

import (
	"context"
	"errors"
	"testing"
)

func TestRepositoryIntegrationCreateSlotsBulkRejectsOverlapInPostgres(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440000"

	initialSlots, err := repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-10",
		StartTime:           "09:00",
		EndTime:             "10:00",
		SlotDurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("seed slots: %v", err)
	}
	if len(initialSlots) != 2 {
		t.Fatalf("initial slots len = %d, want 2", len(initialSlots))
	}

	_, err = repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-10",
		StartTime:           "09:15",
		EndTime:             "09:45",
		SlotDurationMinutes: 30,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v, want %v", err, ErrConflict)
	}

	if got := countSlots(t, db); got != 2 {
		t.Fatalf("slots persisted = %d, want 2", got)
	}
}

func TestRepositoryIntegrationAppointmentLifecyclePersistsAppointmentAndSlotState(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440010"
	patientID := "550e8400-e29b-41d4-a716-446655440011"

	slots, err := repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-11",
		StartTime:           "11:00",
		EndTime:             "11:30",
		SlotDurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create slots: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("slots len = %d, want 1", len(slots))
	}

	created, err := repo.CreateAppointment(ctx, CreateAppointmentParams{
		SlotID:         slots[0].ID,
		PatientID:      patientID,
		ProfessionalID: professionalID,
	})
	if err != nil {
		t.Fatalf("create appointment: %v", err)
	}
	if created.Status != "booked" {
		t.Fatalf("created status = %q, want booked", created.Status)
	}

	persistedCreatedAppointment := fetchAppointmentByID(t, db, created.ID)
	if persistedCreatedAppointment.Status != "booked" {
		t.Fatalf("persisted appointment status = %q, want booked", persistedCreatedAppointment.Status)
	}
	if persistedCreatedAppointment.CancelledAt != nil {
		t.Fatalf("persisted appointment cancelled_at = %v, want nil", persistedCreatedAppointment.CancelledAt)
	}

	persistedBookedSlot := fetchSlotByID(t, db, slots[0].ID)
	if persistedBookedSlot.Status != "booked" {
		t.Fatalf("slot status after booking = %q, want booked", persistedBookedSlot.Status)
	}

	cancelled, err := repo.CancelAppointment(ctx, created.ID)
	if err != nil {
		t.Fatalf("cancel appointment: %v", err)
	}
	if cancelled.Status != "cancelled" {
		t.Fatalf("cancelled status = %q, want cancelled", cancelled.Status)
	}
	if cancelled.CancelledAt == nil {
		t.Fatal("cancelled appointment missing cancelled_at")
	}

	persistedCancelledAppointment := fetchAppointmentByID(t, db, created.ID)
	if persistedCancelledAppointment.Status != "cancelled" {
		t.Fatalf("persisted cancelled status = %q, want cancelled", persistedCancelledAppointment.Status)
	}
	if persistedCancelledAppointment.CancelledAt == nil {
		t.Fatal("persisted cancelled appointment missing cancelled_at")
	}

	persistedAvailableSlot := fetchSlotByID(t, db, slots[0].ID)
	if persistedAvailableSlot.Status != "available" {
		t.Fatalf("slot status after cancellation = %q, want available", persistedAvailableSlot.Status)
	}
	if !persistedAvailableSlot.UpdatedAt.Equal(persistedCancelledAppointment.UpdatedAt) {
		t.Fatalf("slot updated_at = %s, want %s", persistedAvailableSlot.UpdatedAt, persistedCancelledAppointment.UpdatedAt)
	}
}

func TestRepositoryIntegrationCreateAppointmentRejectsDoubleBookingForSameSlot(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440020"
	firstPatientID := "550e8400-e29b-41d4-a716-446655440021"
	secondPatientID := "550e8400-e29b-41d4-a716-446655440022"

	slots, err := repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-12",
		StartTime:           "14:00",
		EndTime:             "14:30",
		SlotDurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create slots: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("slots len = %d, want 1", len(slots))
	}

	created, err := repo.CreateAppointment(ctx, CreateAppointmentParams{
		SlotID:         slots[0].ID,
		PatientID:      firstPatientID,
		ProfessionalID: professionalID,
	})
	if err != nil {
		t.Fatalf("first booking: %v", err)
	}

	_, err = repo.CreateAppointment(ctx, CreateAppointmentParams{
		SlotID:         slots[0].ID,
		PatientID:      secondPatientID,
		ProfessionalID: professionalID,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("second booking err = %v, want %v", err, ErrConflict)
	}

	if got := countAppointments(t, db); got != 1 {
		t.Fatalf("appointments persisted = %d, want 1", got)
	}

	persistedAppointment := fetchAppointmentByID(t, db, created.ID)
	if persistedAppointment.PatientID != firstPatientID {
		t.Fatalf("persisted patient_id = %q, want %q", persistedAppointment.PatientID, firstPatientID)
	}
	if persistedAppointment.Status != "booked" {
		t.Fatalf("persisted appointment status = %q, want booked", persistedAppointment.Status)
	}

	persistedSlot := fetchSlotByID(t, db, slots[0].ID)
	if persistedSlot.Status != "booked" {
		t.Fatalf("slot status after double booking attempt = %q, want booked", persistedSlot.Status)
	}
	if !persistedSlot.UpdatedAt.Equal(persistedAppointment.UpdatedAt) {
		t.Fatalf("slot updated_at = %s, want %s", persistedSlot.UpdatedAt, persistedAppointment.UpdatedAt)
	}
}
