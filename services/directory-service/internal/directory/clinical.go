package directory

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrForbidden = errors.New("directory forbidden")

type CreateEncounterParams struct {
	PatientID      string `json:"-"`
	ProfessionalID string `json:"-"`
	OccurredAt     string `json:"occurred_at"`
	Note           string `json:"note"`
}

type ClinicalChart struct {
	ID             string    `json:"id"`
	PatientID      string    `json:"patient_id"`
	ProfessionalID string    `json:"professional_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ClinicalNote struct {
	ID             string    `json:"id"`
	EncounterID    string    `json:"encounter_id"`
	ChartID        string    `json:"chart_id"`
	PatientID      string    `json:"patient_id"`
	ProfessionalID string    `json:"professional_id"`
	Kind           string    `json:"kind"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Encounter struct {
	ID             string       `json:"id"`
	ChartID        string       `json:"chart_id"`
	PatientID      string       `json:"patient_id"`
	ProfessionalID string       `json:"professional_id"`
	OccurredAt     time.Time    `json:"occurred_at"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	InitialNote    ClinicalNote `json:"initial_note"`
}

func (r *Repository) CreateEncounter(ctx context.Context, params CreateEncounterParams) (Encounter, error) {
	normalized, occurredAt, err := validateCreateEncounterParams(params, time.Now().UTC())
	if err != nil {
		return Encounter{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Encounter{}, err
	}
	defer tx.Rollback()

	var patientExists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM patients WHERE id = $1)
	`, normalized.PatientID).Scan(&patientExists); err != nil {
		return Encounter{}, err
	}
	if !patientExists {
		return Encounter{}, ErrNotFound
	}

	chart, err := scanClinicalChart(tx.QueryRowContext(ctx, `
		INSERT INTO clinical_charts (patient_id, professional_id)
		VALUES ($1, $2)
		ON CONFLICT (patient_id, professional_id)
		DO UPDATE SET updated_at = clinical_charts.updated_at
		RETURNING id, patient_id, professional_id, created_at, updated_at
	`, normalized.PatientID, normalized.ProfessionalID))
	if err != nil {
		return Encounter{}, err
	}

	encounter, err := scanEncounterRow(tx.QueryRowContext(ctx, `
		INSERT INTO clinical_encounters (chart_id, patient_id, professional_id, occurred_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, chart_id, patient_id, professional_id, occurred_at, created_at, updated_at
	`, chart.ID, normalized.PatientID, normalized.ProfessionalID, occurredAt))
	if err != nil {
		return Encounter{}, err
	}

	note, err := scanClinicalNote(tx.QueryRowContext(ctx, `
		INSERT INTO clinical_notes (chart_id, encounter_id, patient_id, professional_id, kind, content)
		VALUES ($1, $2, $3, $4, 'initial', $5)
		RETURNING id, encounter_id, chart_id, patient_id, professional_id, kind, content, created_at, updated_at
	`, chart.ID, encounter.ID, normalized.PatientID, normalized.ProfessionalID, normalized.Note))
	if err != nil {
		return Encounter{}, err
	}

	encounter.InitialNote = note

	if err := tx.Commit(); err != nil {
		return Encounter{}, err
	}

	return encounter, nil
}

func (r *Repository) ListPatientEncounters(ctx context.Context, patientID, professionalID string) ([]Encounter, error) {
	normalizedPatientID, normalizedProfessionalID, err := validateEncounterOwnership(patientID, professionalID)
	if err != nil {
		return nil, err
	}

	var patientExists bool
	if err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM patients WHERE id = $1)
	`, normalizedPatientID).Scan(&patientExists); err != nil {
		return nil, err
	}
	if !patientExists {
		return nil, ErrNotFound
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			e.id,
			e.chart_id,
			e.patient_id,
			e.professional_id,
			e.occurred_at,
			e.created_at,
			e.updated_at,
			n.id,
			n.encounter_id,
			n.chart_id,
			n.patient_id,
			n.professional_id,
			n.kind,
			n.content,
			n.created_at,
			n.updated_at
		FROM clinical_encounters e
		JOIN clinical_notes n ON n.encounter_id = e.id AND n.kind = 'initial'
		WHERE e.patient_id = $1
		  AND e.professional_id = $2
		ORDER BY e.occurred_at DESC, e.created_at DESC
	`, normalizedPatientID, normalizedProfessionalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	encounters := make([]Encounter, 0)
	for rows.Next() {
		encounter, err := scanEncounterWithNote(rows)
		if err != nil {
			return nil, err
		}
		encounters = append(encounters, encounter)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return encounters, nil
}

func validateCreateEncounterParams(params CreateEncounterParams, now time.Time) (CreateEncounterParams, time.Time, error) {
	normalized := CreateEncounterParams{
		PatientID:      strings.TrimSpace(params.PatientID),
		ProfessionalID: strings.TrimSpace(params.ProfessionalID),
		OccurredAt:     strings.TrimSpace(params.OccurredAt),
		Note:           strings.TrimSpace(params.Note),
	}

	patientID, professionalID, err := validateEncounterOwnership(normalized.PatientID, normalized.ProfessionalID)
	if err != nil {
		return CreateEncounterParams{}, time.Time{}, err
	}
	if normalized.Note == "" {
		return CreateEncounterParams{}, time.Time{}, ErrValidation
	}

	normalized.PatientID = patientID
	normalized.ProfessionalID = professionalID

	if normalized.OccurredAt == "" {
		return normalized, now.UTC(), nil
	}

	occurredAt, err := time.Parse(time.RFC3339, normalized.OccurredAt)
	if err != nil {
		return CreateEncounterParams{}, time.Time{}, ErrValidation
	}

	return normalized, occurredAt.UTC(), nil
}

func validateEncounterOwnership(patientID, professionalID string) (string, string, error) {
	normalizedPatientID := strings.TrimSpace(patientID)
	normalizedProfessionalID := strings.TrimSpace(professionalID)

	if _, err := uuid.Parse(normalizedPatientID); err != nil {
		return "", "", ErrNotFound
	}
	if _, err := uuid.Parse(normalizedProfessionalID); err != nil {
		return "", "", ErrValidation
	}

	return normalizedPatientID, normalizedProfessionalID, nil
}

type clinicalChartScanner interface {
	Scan(dest ...any) error
}

func scanClinicalChart(scanner clinicalChartScanner) (ClinicalChart, error) {
	var chart ClinicalChart

	err := scanner.Scan(
		&chart.ID,
		&chart.PatientID,
		&chart.ProfessionalID,
		&chart.CreatedAt,
		&chart.UpdatedAt,
	)
	if err != nil {
		return ClinicalChart{}, err
	}

	return chart, nil
}

type encounterScanner interface {
	Scan(dest ...any) error
}

func scanEncounterRow(scanner encounterScanner) (Encounter, error) {
	var encounter Encounter

	err := scanner.Scan(
		&encounter.ID,
		&encounter.ChartID,
		&encounter.PatientID,
		&encounter.ProfessionalID,
		&encounter.OccurredAt,
		&encounter.CreatedAt,
		&encounter.UpdatedAt,
	)
	if err != nil {
		return Encounter{}, err
	}

	return encounter, nil
}

type clinicalNoteScanner interface {
	Scan(dest ...any) error
}

func scanClinicalNote(scanner clinicalNoteScanner) (ClinicalNote, error) {
	var note ClinicalNote

	err := scanner.Scan(
		&note.ID,
		&note.EncounterID,
		&note.ChartID,
		&note.PatientID,
		&note.ProfessionalID,
		&note.Kind,
		&note.Content,
		&note.CreatedAt,
		&note.UpdatedAt,
	)
	if err != nil {
		return ClinicalNote{}, err
	}

	return note, nil
}

func scanEncounterWithNote(scanner encounterScanner) (Encounter, error) {
	var encounter Encounter

	err := scanner.Scan(
		&encounter.ID,
		&encounter.ChartID,
		&encounter.PatientID,
		&encounter.ProfessionalID,
		&encounter.OccurredAt,
		&encounter.CreatedAt,
		&encounter.UpdatedAt,
		&encounter.InitialNote.ID,
		&encounter.InitialNote.EncounterID,
		&encounter.InitialNote.ChartID,
		&encounter.InitialNote.PatientID,
		&encounter.InitialNote.ProfessionalID,
		&encounter.InitialNote.Kind,
		&encounter.InitialNote.Content,
		&encounter.InitialNote.CreatedAt,
		&encounter.InitialNote.UpdatedAt,
	)
	if err != nil {
		return Encounter{}, err
	}

	return encounter, nil
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
