package appointments

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestRepositoryIntegrationConsultationScheduleRangeMigrationBackfillsAndAllowsStandaloneConsultations(t *testing.T) {
	_, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440170"
	slotID := "550e8400-e29b-41d4-a716-446655440171"
	consultationID := "550e8400-e29b-41d4-a716-446655440172"
	slotStart := time.Date(2026, time.May, 5, 9, 0, 0, 0, time.UTC)
	slotEnd := time.Date(2026, time.May, 5, 9, 30, 0, 0, time.UTC)

	seedAvailabilitySlot(t, db, slotID, professionalID, slotStart, slotEnd, "booked")

	if _, err := db.ExecContext(ctx, `
		INSERT INTO appointments (id, slot_id, professional_id, patient_id, status)
		VALUES ($1, $2, $3, $4, 'booked')
	`, consultationID, slotID, professionalID, "550e8400-e29b-41d4-a716-446655440173"); err != nil {
		t.Fatalf("seed appointment before migration: %v", err)
	}

	applySingleAppointmentsMigration(t, db, "006_consultation_entity.sql")
	applySingleAppointmentsMigration(t, db, "007_consultation_schedule_range.sql")

	var scheduledStart, scheduledEnd time.Time
	if err := db.QueryRowContext(ctx, `
		SELECT scheduled_start, scheduled_end
		FROM consultations
		WHERE id = $1
	`, consultationID).Scan(&scheduledStart, &scheduledEnd); err != nil {
		t.Fatalf("load migrated consultation schedule range: %v", err)
	}
	if !scheduledStart.Equal(slotStart) || !scheduledEnd.Equal(slotEnd) {
		t.Fatalf("migrated schedule = [%s %s], want [%s %s]", scheduledStart, scheduledEnd, slotStart, slotEnd)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO consultations (professional_id, patient_id, status, source, scheduled_start, scheduled_end)
		VALUES ($1, $2, 'scheduled', 'doctor', $3, $4)
	`, professionalID, "550e8400-e29b-41d4-a716-446655440174", time.Date(2026, time.May, 5, 11, 0, 0, 0, time.UTC), time.Date(2026, time.May, 5, 11, 20, 0, 0, time.UTC)); err != nil {
		t.Fatalf("insert standalone consultation after schedule range migration: %v", err)
	}

	assertConsultationScheduleRangeColumnsNotNull(t, db)
}

func assertConsultationScheduleRangeColumnsNotNull(t *testing.T, db *sql.DB) {
	t.Helper()

	if columnAllowsNulls(t, db, "consultations", "scheduled_start") {
		t.Fatal("consultations.scheduled_start allows NULL, want NOT NULL")
	}
	if columnAllowsNulls(t, db, "consultations", "scheduled_end") {
		t.Fatal("consultations.scheduled_end allows NULL, want NOT NULL")
	}
}
