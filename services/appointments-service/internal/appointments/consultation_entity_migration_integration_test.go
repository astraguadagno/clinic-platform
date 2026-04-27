package appointments

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestRepositoryIntegrationConsultationEntityMigrationRenamesAppointmentsAndPreservesBookings(t *testing.T) {
	_, db := newPostgresLegacyAppointmentsIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440150"
	bookedSlotID := "550e8400-e29b-41d4-a716-446655440151"
	cancelledSlotID := "550e8400-e29b-41d4-a716-446655440152"
	bookedAppointmentID := "550e8400-e29b-41d4-a716-446655440153"
	cancelledAppointmentID := "550e8400-e29b-41d4-a716-446655440154"
	bookedPatientID := "550e8400-e29b-41d4-a716-446655440155"
	cancelledPatientID := "550e8400-e29b-41d4-a716-446655440156"

	seedAvailabilitySlot(t, db, bookedSlotID, professionalID, time.Date(2026, 5, 5, 9, 0, 0, 0, time.UTC), time.Date(2026, 5, 5, 9, 30, 0, 0, time.UTC), "booked")
	seedAvailabilitySlot(t, db, cancelledSlotID, professionalID, time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC), time.Date(2026, 5, 5, 10, 30, 0, 0, time.UTC), "cancelled")

	if _, err := db.ExecContext(ctx, `
		INSERT INTO appointments (id, slot_id, professional_id, patient_id, status)
		VALUES ($1, $2, $3, $4, 'booked')
	`, bookedAppointmentID, bookedSlotID, professionalID, bookedPatientID); err != nil {
		t.Fatalf("seed booked appointment: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO appointments (id, slot_id, professional_id, patient_id, status, cancelled_at)
		VALUES ($1, $2, $3, $4, 'cancelled', NOW())
	`, cancelledAppointmentID, cancelledSlotID, professionalID, cancelledPatientID); err != nil {
		t.Fatalf("seed cancelled appointment: %v", err)
	}

	applySingleAppointmentsMigration(t, db, "006_consultation_entity.sql")

	assertRelationExists(t, db, "consultations")
	assertRelationMissing(t, db, "appointments")

	booked := fetchConsultationRecord(t, db, bookedAppointmentID)
	if booked.Status != "scheduled" {
		t.Fatalf("booked migration status = %q, want scheduled", booked.Status)
	}
	if booked.Source != "secretary" {
		t.Fatalf("booked migration source = %q, want secretary", booked.Source)
	}
	if booked.Notes.Valid {
		t.Fatalf("booked migration notes = %q, want NULL", booked.Notes.String)
	}

	cancelled := fetchConsultationRecord(t, db, cancelledAppointmentID)
	if cancelled.Status != "cancelled" {
		t.Fatalf("cancelled migration status = %q, want cancelled", cancelled.Status)
	}
	if !cancelled.CancelledAt.Valid {
		t.Fatal("cancelled migration cancelled_at = NULL, want timestamp")
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO consultations (slot_id, professional_id, patient_id, status, source)
		VALUES ($1, $2, $3, 'checked_in', 'doctor')
	`, cancelledSlotID, professionalID, "550e8400-e29b-41d4-a716-446655440157"); err != nil {
		t.Fatalf("insert checked-in consultation: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO consultations (slot_id, professional_id, patient_id, status, source)
		VALUES ($1, $2, $3, 'rescheduled', 'doctor')
	`, cancelledSlotID, professionalID, "550e8400-e29b-41d4-a716-446655440158"); err == nil {
		t.Fatal("expected invalid consultation status insert to fail")
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO consultations (slot_id, professional_id, patient_id, status, source)
		VALUES ($1, $2, $3, 'scheduled', 'referral')
	`, cancelledSlotID, professionalID, "550e8400-e29b-41d4-a716-446655440159"); err == nil {
		t.Fatal("expected invalid consultation source insert to fail")
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO consultations (slot_id, professional_id, patient_id, status, source, cancelled_at)
		VALUES ($1, $2, $3, 'cancelled', 'doctor', NOW())
	`, bookedSlotID, professionalID, "550e8400-e29b-41d4-a716-446655440160"); err != nil {
		t.Fatalf("insert cancelled consultation on scheduled slot: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO consultations (slot_id, professional_id, patient_id, status, source)
		VALUES ($1, $2, $3, 'scheduled', 'doctor')
	`, bookedSlotID, professionalID, "550e8400-e29b-41d4-a716-446655440161"); err == nil {
		t.Fatal("expected duplicate scheduled consultation insert to fail")
	}

	if got := countRows(t, db, "consultations"); got != 4 {
		t.Fatalf("consultations persisted = %d, want 4", got)
	}
}

type consultationMigrationRecord struct {
	Status      string
	Source      string
	Notes       sql.NullString
	CancelledAt sql.NullTime
}

func applySingleAppointmentsMigration(t *testing.T, db *sql.DB, name string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, readAppointmentsMigrationFile(t, name)); err != nil {
		t.Fatalf("apply migration %s: %v", name, err)
	}
}

func assertRelationExists(t *testing.T, db *sql.DB, relation string) {
	t.Helper()

	var got sql.NullString
	if err := db.QueryRow(`SELECT to_regclass('public.' || $1)`, relation).Scan(&got); err != nil {
		t.Fatalf("lookup relation %s: %v", relation, err)
	}
	if !got.Valid || got.String == "" {
		t.Fatalf("relation %s missing", relation)
	}
}

func assertRelationMissing(t *testing.T, db *sql.DB, relation string) {
	t.Helper()

	var got sql.NullString
	if err := db.QueryRow(`SELECT to_regclass('public.' || $1)`, relation).Scan(&got); err != nil {
		t.Fatalf("lookup relation %s: %v", relation, err)
	}
	if got.Valid && got.String != "" {
		t.Fatalf("relation %s exists as %s, want missing", relation, got.String)
	}
}

func fetchConsultationRecord(t *testing.T, db *sql.DB, consultationID string) consultationMigrationRecord {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var record consultationMigrationRecord
	if err := db.QueryRowContext(ctx, `
		SELECT status, source, notes, cancelled_at
		FROM consultations
		WHERE id = $1
	`, consultationID).Scan(&record.Status, &record.Source, &record.Notes, &record.CancelledAt); err != nil {
		t.Fatalf("fetch consultation %s: %v", consultationID, err)
	}

	return record
}

func seedAvailabilitySlot(t *testing.T, db *sql.DB, slotID string, professionalID string, start time.Time, end time.Time, status string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO availability_slots (id, professional_id, start_time, end_time, status)
		VALUES ($1, $2, $3, $4, $5)
	`, slotID, professionalID, start, end, status); err != nil {
		t.Fatalf("seed slot %s: %v", slotID, err)
	}
}
