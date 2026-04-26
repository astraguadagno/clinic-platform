package appointments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestRepositoryIntegrationLegacyAppointmentsRemainFunctionalAfterConsultationMigration(t *testing.T) {
	repo, db := newPostgresConsultationIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440401"
	patientID := "550e8400-e29b-41d4-a716-446655440402"

	slots, err := repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-20",
		StartTime:           "09:00",
		EndTime:             "09:30",
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
		t.Fatalf("create appointment through legacy flow: %v", err)
	}
	if created.Status != "booked" {
		t.Fatalf("created status = %q, want booked", created.Status)
	}

	persisted := fetchConsultationRecord(t, db, created.ID)
	if persisted.Status != "scheduled" {
		t.Fatalf("persisted consultation status = %q, want scheduled", persisted.Status)
	}
	if persisted.Source != "secretary" {
		t.Fatalf("persisted consultation source = %q, want secretary", persisted.Source)
	}

	fetched, err := repo.GetAppointmentByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get appointment by id through legacy flow: %v", err)
	}
	if fetched.Status != "booked" {
		t.Fatalf("fetched status = %q, want booked", fetched.Status)
	}

	listed, err := repo.ListAppointments(ctx, AppointmentFilters{ProfessionalID: professionalID})
	if err != nil {
		t.Fatalf("list appointments through legacy flow: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed len = %d, want 1", len(listed))
	}
	if listed[0].Status != "booked" {
		t.Fatalf("listed status = %q, want booked", listed[0].Status)
	}

	cancelled, err := repo.CancelAppointment(ctx, created.ID)
	if err != nil {
		t.Fatalf("cancel appointment through legacy flow: %v", err)
	}
	if cancelled.Status != "cancelled" {
		t.Fatalf("cancelled status = %q, want cancelled", cancelled.Status)
	}
	if cancelled.CancelledAt == nil {
		t.Fatal("cancelled appointment missing cancelled_at")
	}

	reloaded := fetchConsultationRecord(t, db, created.ID)
	if reloaded.Status != "cancelled" {
		t.Fatalf("reloaded consultation status = %q, want cancelled", reloaded.Status)
	}
	if !reloaded.CancelledAt.Valid {
		t.Fatal("reloaded consultation cancelled_at = NULL, want timestamp")
	}

	slot := fetchSlotByID(t, db, slots[0].ID)
	if slot.Status != "available" {
		t.Fatalf("slot status after cancellation = %q, want available", slot.Status)
	}
}

func TestRepositoryIntegrationLegacyAppointmentsListScheduledConsultationsAsBooked(t *testing.T) {
	repo, db := newPostgresConsultationIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440411"
	patientID := "550e8400-e29b-41d4-a716-446655440412"
	slotID := "550e8400-e29b-41d4-a716-446655440413"
	start := time.Date(2026, time.April, 21, 10, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)

	seedAvailabilitySlot(t, db, slotID, professionalID, start, end, "booked")

	consultation, err := repo.CreateConsultation(ctx, CreateConsultationParams{
		SlotID:         &slotID,
		ProfessionalID: professionalID,
		PatientID:      patientID,
		Source:         ConsultationSourceDoctor,
	})
	if err != nil {
		t.Fatalf("create consultation: %v", err)
	}

	fetched, err := repo.GetAppointmentByID(ctx, consultation.ID)
	if err != nil {
		t.Fatalf("get scheduled consultation through legacy flow: %v", err)
	}
	if fetched.Status != "booked" {
		t.Fatalf("legacy fetched status = %q, want booked", fetched.Status)
	}

	listed, err := repo.ListAppointments(ctx, AppointmentFilters{ProfessionalID: professionalID, Status: "booked"})
	if err != nil {
		t.Fatalf("list scheduled consultation through legacy flow: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed len = %d, want 1", len(listed))
	}
	if listed[0].ID != consultation.ID {
		t.Fatalf("listed consultation id = %q, want %q", listed[0].ID, consultation.ID)
	}
	if listed[0].Status != "booked" {
		t.Fatalf("listed status = %q, want booked", listed[0].Status)
	}
}

func TestRepositoryIntegrationLegacyAppointmentsRejectBlockedSlotsAfterConsultationMigration(t *testing.T) {
	repo, db := newPostgresConsultationIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440421"
	patientID := "550e8400-e29b-41d4-a716-446655440422"

	template, err := repo.CreateTemplate(ctx, CreateTemplateParams{
		ProfessionalID: professionalID,
		EffectiveFrom:  "2026-04-01",
		Recurrence:     json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}}`),
	})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	_, err = repo.CreateScheduleBlock(ctx, CreateScheduleBlockParams{
		ProfessionalID: professionalID,
		Scope:          "template",
		DayOfWeek:      intPtr(1),
		StartTime:      "09:00",
		EndTime:        "09:30",
		TemplateID:     &template.ID,
	})
	if err != nil {
		t.Fatalf("create schedule block: %v", err)
	}

	slots, err := repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-20",
		StartTime:           "09:00",
		EndTime:             "09:30",
		SlotDurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create slots: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("slots len = %d, want 1", len(slots))
	}

	_, err = repo.CreateAppointment(ctx, CreateAppointmentParams{
		SlotID:         slots[0].ID,
		PatientID:      patientID,
		ProfessionalID: professionalID,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("create appointment err = %v, want %v", err, ErrConflict)
	}

	if got := countRows(t, db, "consultations"); got != 0 {
		t.Fatalf("consultations persisted = %d, want 0", got)
	}

	slot := fetchSlotByID(t, db, slots[0].ID)
	if slot.Status != "available" {
		t.Fatalf("slot status after blocked booking attempt = %q, want available", slot.Status)
	}
}
