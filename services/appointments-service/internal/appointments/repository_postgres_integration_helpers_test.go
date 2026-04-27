package appointments

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func newPostgresIntegrationRepository(t *testing.T) (*Repository, *sql.DB) {
	t.Helper()

	dsn := postgresIntegrationTestDSN()
	if dsn == "" {
		t.Skip(describeIntegrationDSNRequirement())
	}

	if reason := postgresIntegrationResetSkipReason(dsn); reason != "" {
		t.Skip(reason)
	}

	db, err := OpenDB(dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	resetAppointmentsSchema(t, db)
	applyCurrentAppointmentsSchema(t, db)

	return NewRepository(db), db
}

func newPostgresConsultationIntegrationRepository(t *testing.T) (*Repository, *sql.DB) {
	t.Helper()

	dsn := postgresIntegrationTestDSN()
	if dsn == "" {
		t.Skip(describeIntegrationDSNRequirement())
	}

	if reason := postgresIntegrationResetSkipReason(dsn); reason != "" {
		t.Skip(reason)
	}

	db, err := OpenDB(dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	resetAppointmentsSchema(t, db)
	applyCurrentAppointmentsSchema(t, db)

	if reason := consultationRepositorySchemaSupportSkipReason(t, db); reason != "" {
		t.Fatal(reason)
	}

	return NewRepository(db), db
}

func newPostgresLegacyAppointmentsIntegrationRepository(t *testing.T) (*Repository, *sql.DB) {
	t.Helper()

	dsn := postgresIntegrationTestDSN()
	if dsn == "" {
		t.Skip(describeIntegrationDSNRequirement())
	}

	if reason := postgresIntegrationResetSkipReason(dsn); reason != "" {
		t.Skip(reason)
	}

	db, err := OpenDB(dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	resetAppointmentsSchema(t, db)
	applyLegacyAppointmentsSchema(t, db)
	applySingleAppointmentsMigration(t, db, "004_schedule_templates.sql")
	applySingleAppointmentsMigration(t, db, "005_schedule_blocks.sql")

	return NewRepository(db), db
}

func postgresIntegrationTestDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("APPOINTMENTS_TEST_DATABASE_DSN")); dsn != "" {
		return dsn
	}

	return ""
}

func postgresIntegrationResetSkipReason(dsn string) string {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Sprintf("skipping destructive PostgreSQL integration reset: invalid %s: %v", "APPOINTMENTS_TEST_DATABASE_DSN", err)
	}

	databaseName := strings.TrimSpace(config.Database)
	if databaseName == "" {
		return fmt.Sprintf("skipping destructive PostgreSQL integration reset: %s must include a database name ending in _test", "APPOINTMENTS_TEST_DATABASE_DSN")
	}

	if !strings.HasSuffix(databaseName, "_test") {
		return fmt.Sprintf("skipping destructive PostgreSQL integration reset: database %q from %s must end with _test", databaseName, "APPOINTMENTS_TEST_DATABASE_DSN")
	}

	if strings.TrimSpace(os.Getenv("APPOINTMENTS_TEST_DATABASE_RESET_ALLOWED")) != "true" {
		return "skipping destructive PostgreSQL integration reset: set APPOINTMENTS_TEST_DATABASE_RESET_ALLOWED=true to allow schema reset"
	}

	return ""
}

func resetAppointmentsSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const resetSQL = `
		DROP SCHEMA IF EXISTS public CASCADE;
		CREATE SCHEMA public;
		GRANT ALL ON SCHEMA public TO CURRENT_USER;
		GRANT ALL ON SCHEMA public TO public;
	`

	if _, err := db.ExecContext(ctx, resetSQL); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
}

func applyAppointmentsMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	applyCurrentAppointmentsSchema(t, db)
}

func applyCurrentAppointmentsSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	for _, migration := range []string{"001_init.sql"} {
		contents := readAppointmentsMigrationFile(t, migration)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := db.ExecContext(ctx, contents)
		cancel()
		if err != nil {
			t.Fatalf("apply migration %s: %v", migration, err)
		}
	}
}

func applyLegacyAppointmentsSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	const legacySchema = `
		CREATE EXTENSION IF NOT EXISTS pgcrypto;
		CREATE EXTENSION IF NOT EXISTS btree_gist;

		CREATE TABLE availability_slots (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			professional_id UUID NOT NULL,
			start_time TIMESTAMPTZ NOT NULL,
			end_time TIMESTAMPTZ NOT NULL,
			status TEXT NOT NULL DEFAULT 'available',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT availability_slots_time_range_valid CHECK (start_time < end_time),
			CONSTRAINT availability_slots_status_valid CHECK (status IN ('available', 'booked', 'cancelled')),
			CONSTRAINT availability_slots_professional_start_unique UNIQUE (professional_id, start_time),
			CONSTRAINT availability_slots_no_overlap EXCLUDE USING gist (
				professional_id WITH =,
				tstzrange(start_time, end_time, '[)') WITH &&
			)
		);

		CREATE INDEX availability_slots_professional_start_idx
			ON availability_slots (professional_id, start_time);
		CREATE INDEX availability_slots_status_idx
			ON availability_slots (status);

		CREATE TABLE appointments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			slot_id UUID NOT NULL,
			professional_id UUID NOT NULL,
			patient_id UUID NOT NULL,
			status TEXT NOT NULL DEFAULT 'booked',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			cancelled_at TIMESTAMPTZ,
			CONSTRAINT appointments_slot_fk FOREIGN KEY (slot_id) REFERENCES availability_slots(id),
			CONSTRAINT appointments_status_valid CHECK (status IN ('booked', 'cancelled')),
			CONSTRAINT appointments_cancelled_at_consistency CHECK (
				(status = 'cancelled' AND cancelled_at IS NOT NULL) OR
				(status = 'booked' AND cancelled_at IS NULL)
			)
		);

		CREATE UNIQUE INDEX appointments_slot_booked_unique_idx
			ON appointments (slot_id)
			WHERE status = 'booked';
		CREATE INDEX appointments_patient_id_idx
			ON appointments (patient_id);
		CREATE INDEX appointments_professional_id_idx
			ON appointments (professional_id);
		CREATE INDEX appointments_status_idx
			ON appointments (status);
	`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, legacySchema); err != nil {
		t.Fatalf("apply legacy appointments schema: %v", err)
	}
}

func readAppointmentsMigrationFile(t *testing.T, name string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}

	path := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", name)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}

	return string(contents)
}

func fetchSlotByID(t *testing.T, db *sql.DB, slotID string) AvailabilitySlot {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := db.QueryRowContext(ctx, `
		SELECT id, professional_id, start_time, end_time, status, created_at, updated_at
		FROM availability_slots
		WHERE id = $1
	`, slotID)

	slot, err := scanSlot(row)
	if err != nil {
		t.Fatalf("fetch slot %s: %v", slotID, err)
	}

	return slot
}

func fetchAppointmentByID(t *testing.T, db *sql.DB, appointmentID string) Appointment {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	useConsultations, err := usesConsultationAppointmentStore(ctx, db)
	if err != nil {
		t.Fatalf("resolve appointment store: %v", err)
	}

	query := `
		SELECT id, slot_id, professional_id, patient_id, status, created_at, updated_at, cancelled_at
		FROM appointments
		WHERE id = $1
	`
	if useConsultations {
		query = `
			SELECT id, slot_id, professional_id, patient_id, ` + legacyAppointmentStatusSelectSQL("status") + `, created_at, updated_at, cancelled_at
			FROM consultations
			WHERE id = $1
		`
	}

	row := db.QueryRowContext(ctx, query, appointmentID)

	appointment, err := scanAppointment(row)
	if err != nil {
		t.Fatalf("fetch appointment %s: %v", appointmentID, err)
	}

	return appointment
}

func countSlots(t *testing.T, db *sql.DB) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var total int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM availability_slots`).Scan(&total); err != nil {
		t.Fatalf("count slots: %v", err)
	}

	return total
}

func countAppointments(t *testing.T, db *sql.DB) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	useConsultations, err := usesConsultationAppointmentStore(ctx, db)
	if err != nil {
		t.Fatalf("resolve appointment store: %v", err)
	}
	tableName := "appointments"
	if useConsultations {
		tableName = "consultations"
	}

	var total int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if err := db.QueryRowContext(ctx, query).Scan(&total); err != nil {
		t.Fatalf("count appointments: %v", err)
	}

	return total
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var total int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if err := db.QueryRowContext(ctx, query).Scan(&total); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}

	return total
}

func describeIntegrationDSNRequirement() string {
	return fmt.Sprintf("run with %s set to a dedicated PostgreSQL database ending in _test and APPOINTMENTS_TEST_DATABASE_RESET_ALLOWED=true", "APPOINTMENTS_TEST_DATABASE_DSN")
}

func consultationRepositorySchemaSupportSkipReason(t *testing.T, db *sql.DB) string {
	t.Helper()

	if !columnExists(t, db, "consultations", "check_in_time") {
		return "consultation repository integration schema invalid: consultations.check_in_time column is not present"
	}
	if !columnExists(t, db, "consultations", "reception_notes") {
		return "consultation repository integration schema invalid: consultations.reception_notes column is not present"
	}
	if !columnAllowsNulls(t, db, "consultations", "slot_id") {
		return "consultation repository integration schema invalid: consultations.slot_id is still NOT NULL"
	}
	if !columnExists(t, db, "consultations", "scheduled_start") {
		return "consultation repository integration schema invalid: consultations.scheduled_start column is not present"
	}
	if !columnExists(t, db, "consultations", "scheduled_end") {
		return "consultation repository integration schema invalid: consultations.scheduled_end column is not present"
	}

	return ""
}

func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var exists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = $1
			  AND column_name = $2
		)
	`, table, column).Scan(&exists); err != nil {
		t.Fatalf("lookup column %s.%s: %v", table, column, err)
	}

	return exists
}

func columnAllowsNulls(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var isNullable string
	if err := db.QueryRowContext(ctx, `
		SELECT is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = $1
		  AND column_name = $2
	`, table, column).Scan(&isNullable); err != nil {
		t.Fatalf("lookup nullability %s.%s: %v", table, column, err)
	}

	return strings.EqualFold(strings.TrimSpace(isNullable), "YES")
}
