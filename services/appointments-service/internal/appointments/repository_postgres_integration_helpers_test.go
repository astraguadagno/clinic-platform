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
	applyAppointmentsMigrations(t, db)

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

	for _, migration := range []string{"001_init.sql", "002_prevent_availability_slot_overlaps.sql"} {
		contents := readAppointmentsMigrationFile(t, migration)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := db.ExecContext(ctx, contents)
		cancel()
		if err != nil {
			t.Fatalf("apply migration %s: %v", migration, err)
		}
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

	row := db.QueryRowContext(ctx, `
		SELECT id, slot_id, professional_id, patient_id, status, created_at, updated_at, cancelled_at
		FROM appointments
		WHERE id = $1
	`, appointmentID)

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

	var total int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM appointments`).Scan(&total); err != nil {
		t.Fatalf("count appointments: %v", err)
	}

	return total
}

func describeIntegrationDSNRequirement() string {
	return fmt.Sprintf("run with %s set to a dedicated PostgreSQL database ending in _test and APPOINTMENTS_TEST_DATABASE_RESET_ALLOWED=true", "APPOINTMENTS_TEST_DATABASE_DSN")
}
