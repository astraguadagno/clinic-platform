package appointments

import (
	"context"
	"testing"
	"time"
)

func TestRepositoryIntegrationConsultationLifecyclePersistsOptionalMetadata(t *testing.T) {
	repo, _ := newPostgresConsultationIntegrationRepository(t)

	ctx := context.Background()
	slotID := "550e8400-e29b-41d4-a716-446655440301"
	professionalID := "550e8400-e29b-41d4-a716-446655440302"
	patientID := "550e8400-e29b-41d4-a716-446655440303"
	notes := "Trae resultados de laboratorio"
	receptionNotes := "Paciente presente en recepción"
	checkInTime := time.Date(2026, time.April, 16, 9, 55, 0, 0, time.UTC)
	scheduledStart := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, time.April, 16, 10, 30, 0, 0, time.UTC)

	seedAvailabilitySlot(t, repo.db, slotID, professionalID, scheduledStart, scheduledEnd, "available")

	created, err := repo.CreateConsultation(ctx, CreateConsultationParams{
		SlotID:         &slotID,
		ProfessionalID: professionalID,
		PatientID:      patientID,
		Source:         ConsultationSourceSecretary,
		Notes:          &notes,
	})
	if err != nil {
		t.Fatalf("create consultation: %v", err)
	}
	if created.SlotID == nil || *created.SlotID != slotID {
		t.Fatalf("created slot_id = %v, want %q", created.SlotID, slotID)
	}
	if created.Notes == nil || *created.Notes != notes {
		t.Fatalf("created notes = %v, want %q", created.Notes, notes)
	}
	if !created.ScheduledStart.Equal(scheduledStart) {
		t.Fatalf("created scheduled_start = %s, want %s", created.ScheduledStart, scheduledStart)
	}
	if !created.ScheduledEnd.Equal(scheduledEnd) {
		t.Fatalf("created scheduled_end = %s, want %s", created.ScheduledEnd, scheduledEnd)
	}

	persisted, err := repo.GetConsultation(ctx, created.ID)
	if err != nil {
		t.Fatalf("get consultation: %v", err)
	}
	if persisted.Status != ConsultationStatusScheduled {
		t.Fatalf("persisted status = %q, want %q", persisted.Status, ConsultationStatusScheduled)
	}
	if !persisted.ScheduledStart.Equal(scheduledStart) || !persisted.ScheduledEnd.Equal(scheduledEnd) {
		t.Fatalf("persisted schedule = [%s %s], want [%s %s]", persisted.ScheduledStart, persisted.ScheduledEnd, scheduledStart, scheduledEnd)
	}

	updated, err := repo.UpdateConsultationStatus(ctx, created.ID, UpdateConsultationStatusParams{
		Status:         ConsultationStatusCheckedIn,
		CheckInTime:    &checkInTime,
		ReceptionNotes: &receptionNotes,
	})
	if err != nil {
		t.Fatalf("update consultation status: %v", err)
	}
	if updated.CheckInTime == nil || !updated.CheckInTime.Equal(checkInTime) {
		t.Fatalf("updated check_in_time = %v, want %s", updated.CheckInTime, checkInTime)
	}
	if updated.ReceptionNotes == nil || *updated.ReceptionNotes != receptionNotes {
		t.Fatalf("updated reception_notes = %v, want %q", updated.ReceptionNotes, receptionNotes)
	}

	reloaded, err := repo.GetConsultation(ctx, created.ID)
	if err != nil {
		t.Fatalf("reload consultation: %v", err)
	}
	if reloaded.CheckInTime == nil || !reloaded.CheckInTime.Equal(checkInTime) {
		t.Fatalf("reloaded check_in_time = %v, want %s", reloaded.CheckInTime, checkInTime)
	}
	if reloaded.ReceptionNotes == nil || *reloaded.ReceptionNotes != receptionNotes {
		t.Fatalf("reloaded reception_notes = %v, want %q", reloaded.ReceptionNotes, receptionNotes)
	}
}

func TestRepositoryIntegrationStandaloneConsultationPersistsScheduleRange(t *testing.T) {
	repo, _ := newPostgresConsultationIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440311"
	patientID := "550e8400-e29b-41d4-a716-446655440312"
	scheduledStart := time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, time.April, 16, 11, 20, 0, 0, time.UTC)

	created, err := repo.CreateConsultation(ctx, CreateConsultationParams{
		ProfessionalID: professionalID,
		PatientID:      patientID,
		Source:         ConsultationSourceDoctor,
		ScheduledStart: &scheduledStart,
		ScheduledEnd:   &scheduledEnd,
	})
	if err != nil {
		t.Fatalf("create standalone consultation: %v", err)
	}
	if created.SlotID != nil {
		t.Fatalf("created slot_id = %v, want nil", created.SlotID)
	}
	if !created.ScheduledStart.Equal(scheduledStart) || !created.ScheduledEnd.Equal(scheduledEnd) {
		t.Fatalf("created schedule = [%s %s], want [%s %s]", created.ScheduledStart, created.ScheduledEnd, scheduledStart, scheduledEnd)
	}

	persisted, err := repo.GetConsultation(ctx, created.ID)
	if err != nil {
		t.Fatalf("get standalone consultation: %v", err)
	}
	if !persisted.ScheduledStart.Equal(scheduledStart) || !persisted.ScheduledEnd.Equal(scheduledEnd) {
		t.Fatalf("persisted schedule = [%s %s], want [%s %s]", persisted.ScheduledStart, persisted.ScheduledEnd, scheduledStart, scheduledEnd)
	}
}
