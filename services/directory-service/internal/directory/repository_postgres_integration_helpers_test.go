package directory

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

	if testing.Short() {
		t.Skip("skipping PostgreSQL integration test in short mode")
	}

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

	resetDirectorySchema(t, db)
	applyDirectoryMigrations(t, db)

	return NewRepository(db), db
}

func postgresIntegrationTestDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("DIRECTORY_TEST_DATABASE_DSN")); dsn != "" {
		return dsn
	}

	return ""
}

func postgresIntegrationResetSkipReason(dsn string) string {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Sprintf("skipping destructive PostgreSQL integration reset: invalid %s: %v", "DIRECTORY_TEST_DATABASE_DSN", err)
	}

	databaseName := strings.TrimSpace(config.Database)
	if databaseName == "" {
		return fmt.Sprintf("skipping destructive PostgreSQL integration reset: %s must include a database name ending in _test", "DIRECTORY_TEST_DATABASE_DSN")
	}

	if !strings.HasSuffix(databaseName, "_test") {
		return fmt.Sprintf("skipping destructive PostgreSQL integration reset: database %q from %s must end with _test", databaseName, "DIRECTORY_TEST_DATABASE_DSN")
	}

	if strings.TrimSpace(os.Getenv("DIRECTORY_TEST_DATABASE_RESET_ALLOWED")) != "true" {
		return "skipping destructive PostgreSQL integration reset: set DIRECTORY_TEST_DATABASE_RESET_ALLOWED=true to allow schema reset"
	}

	return ""
}

func resetDirectorySchema(t *testing.T, db *sql.DB) {
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

func applyDirectoryMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	for _, migration := range []string{"001_init.sql", "002_access_auth.sql", "003_clinical_core.sql", "004_clinical_history_foundation.sql"} {
		contents := readDirectoryMigrationFile(t, migration)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := db.ExecContext(ctx, contents)
		cancel()
		if err != nil {
			t.Fatalf("apply migration %s: %v", migration, err)
		}
	}
}

func readDirectoryMigrationFile(t *testing.T, name string) string {
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

func fetchClinicalChartByOwner(t *testing.T, db *sql.DB, patientID, professionalID string) ClinicalChart {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := db.QueryRowContext(ctx, `
		SELECT id, patient_id, professional_id, created_at, updated_at
		FROM clinical_charts
		WHERE patient_id = $1 AND professional_id = $2
	`, patientID, professionalID)

	chart, err := scanClinicalChart(row)
	if err != nil {
		t.Fatalf("fetch clinical chart for patient %s and professional %s: %v", patientID, professionalID, err)
	}

	return chart
}

func fetchEncounterByID(t *testing.T, db *sql.DB, encounterID string) Encounter {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := db.QueryRowContext(ctx, `
		SELECT id, chart_id, patient_id, professional_id, occurred_at, created_at, updated_at
		FROM clinical_encounters
		WHERE id = $1
	`, encounterID)

	encounter, err := scanEncounterRow(row)
	if err != nil {
		t.Fatalf("fetch encounter %s: %v", encounterID, err)
	}

	return encounter
}

func fetchClinicalNoteByEncounterID(t *testing.T, db *sql.DB, encounterID string) ClinicalNote {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := db.QueryRowContext(ctx, `
		SELECT id, encounter_id, chart_id, patient_id, professional_id, kind, content, created_at, updated_at
		FROM clinical_notes
		WHERE encounter_id = $1 AND kind = 'initial'
	`, encounterID)

	note, err := scanClinicalNote(row)
	if err != nil {
		t.Fatalf("fetch clinical note for encounter %s: %v", encounterID, err)
	}

	return note
}

func countClinicalCharts(t *testing.T, db *sql.DB) int {
	t.Helper()

	return countTableRows(t, db, "clinical_charts")
}

func countClinicalEncounters(t *testing.T, db *sql.DB) int {
	t.Helper()

	return countTableRows(t, db, "clinical_encounters")
}

func countClinicalNotes(t *testing.T, db *sql.DB) int {
	t.Helper()

	return countTableRows(t, db, "clinical_notes")
}

func countTableRows(t *testing.T, db *sql.DB, table string) int {
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

func seedClinicalPatient(t *testing.T, repo *Repository, suffix string) Patient {
	t.Helper()

	patient, err := repo.CreatePatient(context.Background(), CreatePatientParams{
		FirstName: "Ada",
		LastName:  "Lovelace",
		Document:  fmt.Sprintf("DOC-%s", suffix),
		BirthDate: "1990-10-10",
		Phone:     fmt.Sprintf("555-%s", suffix),
		Email:     fmt.Sprintf("ada.%s@example.com", suffix),
	})
	if err != nil {
		t.Fatalf("seed patient %s: %v", suffix, err)
	}

	return patient
}

func seedClinicalProfessional(t *testing.T, repo *Repository, suffix string) Professional {
	t.Helper()

	professional, err := repo.CreateProfessional(context.Background(), CreateProfessionalParams{
		FirstName: "Ana",
		LastName:  fmt.Sprintf("Medina-%s", suffix),
		Specialty: "cardiology",
	})
	if err != nil {
		t.Fatalf("seed professional %s: %v", suffix, err)
	}

	return professional
}

func describeIntegrationDSNRequirement() string {
	return "run with DIRECTORY_TEST_DATABASE_DSN set to a dedicated PostgreSQL database ending in _test and DIRECTORY_TEST_DATABASE_RESET_ALLOWED=true"
}
