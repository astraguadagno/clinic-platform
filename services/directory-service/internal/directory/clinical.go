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

const maxClinicalHistoryTextLength = 4000
const maxWeightKG = 500
const maxHeightCM = 300

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

type ClinicalHistory struct {
	ID                  string    `json:"id"`
	PatientID           string    `json:"patient_id"`
	WeightKG            *float64  `json:"weight_kg"`
	HeightCM            *float64  `json:"height_cm"`
	Antecedentes        *string   `json:"antecedentes"`
	Allergies           *string   `json:"allergies"`
	HabitualMedication  *string   `json:"habitual_medication"`
	ChronicConditions   *string   `json:"chronic_conditions"`
	Habits              *string   `json:"habits"`
	GeneralObservations *string   `json:"general_observations"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type UpdateClinicalHistoryParams struct {
	PatientID           string   `json:"-"`
	WeightKG            *float64 `json:"weight_kg,omitempty"`
	HeightCM            *float64 `json:"height_cm,omitempty"`
	Antecedentes        *string  `json:"antecedentes,omitempty"`
	Allergies           *string  `json:"allergies,omitempty"`
	HabitualMedication  *string  `json:"habitual_medication,omitempty"`
	ChronicConditions   *string  `json:"chronic_conditions,omitempty"`
	Habits              *string  `json:"habits,omitempty"`
	GeneralObservations *string  `json:"general_observations,omitempty"`

	SetWeightKG            bool `json:"-"`
	SetHeightCM            bool `json:"-"`
	SetAntecedentes        bool `json:"-"`
	SetAllergies           bool `json:"-"`
	SetHabitualMedication  bool `json:"-"`
	SetChronicConditions   bool `json:"-"`
	SetHabits              bool `json:"-"`
	SetGeneralObservations bool `json:"-"`
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

func (r *Repository) GetClinicalHistory(ctx context.Context, patientID string) (ClinicalHistory, error) {
	normalizedPatientID, err := validateClinicalHistoryPatientID(patientID)
	if err != nil {
		return ClinicalHistory{}, err
	}

	var patientExists bool
	if err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM patients WHERE id = $1)
	`, normalizedPatientID).Scan(&patientExists); err != nil {
		return ClinicalHistory{}, err
	}
	if !patientExists {
		return ClinicalHistory{}, ErrNotFound
	}

	return scanClinicalHistory(r.db.QueryRowContext(ctx, `
		INSERT INTO clinical_history (patient_id)
		VALUES ($1)
		ON CONFLICT (patient_id) DO UPDATE SET updated_at = clinical_history.updated_at
		RETURNING id, patient_id, weight_kg, height_cm, antecedentes, allergies, habitual_medication,
			chronic_conditions, habits, general_observations, created_at, updated_at
	`, normalizedPatientID))
}

func (r *Repository) UpdateClinicalHistory(ctx context.Context, params UpdateClinicalHistoryParams) (ClinicalHistory, error) {
	normalized, err := validateUpdateClinicalHistoryParams(params)
	if err != nil {
		return ClinicalHistory{}, err
	}

	var patientExists bool
	if err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM patients WHERE id = $1)
	`, normalized.PatientID).Scan(&patientExists); err != nil {
		return ClinicalHistory{}, err
	}
	if !patientExists {
		return ClinicalHistory{}, ErrNotFound
	}

	return scanClinicalHistory(r.db.QueryRowContext(ctx, `
		INSERT INTO clinical_history (
			patient_id, weight_kg, height_cm, antecedentes, allergies, habitual_medication,
			chronic_conditions, habits, general_observations
		)
		VALUES (
			$1,
			CASE WHEN $2 THEN $3::numeric ELSE NULL END,
			CASE WHEN $4 THEN $5::numeric ELSE NULL END,
			CASE WHEN $6 THEN $7::text ELSE NULL END,
			CASE WHEN $8 THEN $9::text ELSE NULL END,
			CASE WHEN $10 THEN $11::text ELSE NULL END,
			CASE WHEN $12 THEN $13::text ELSE NULL END,
			CASE WHEN $14 THEN $15::text ELSE NULL END,
			CASE WHEN $16 THEN $17::text ELSE NULL END
		)
		ON CONFLICT (patient_id) DO UPDATE SET
			weight_kg = CASE WHEN $2 THEN EXCLUDED.weight_kg ELSE clinical_history.weight_kg END,
			height_cm = CASE WHEN $4 THEN EXCLUDED.height_cm ELSE clinical_history.height_cm END,
			antecedentes = CASE WHEN $6 THEN EXCLUDED.antecedentes ELSE clinical_history.antecedentes END,
			allergies = CASE WHEN $8 THEN EXCLUDED.allergies ELSE clinical_history.allergies END,
			habitual_medication = CASE WHEN $10 THEN EXCLUDED.habitual_medication ELSE clinical_history.habitual_medication END,
			chronic_conditions = CASE WHEN $12 THEN EXCLUDED.chronic_conditions ELSE clinical_history.chronic_conditions END,
			habits = CASE WHEN $14 THEN EXCLUDED.habits ELSE clinical_history.habits END,
			general_observations = CASE WHEN $16 THEN EXCLUDED.general_observations ELSE clinical_history.general_observations END,
			updated_at = NOW()
		RETURNING id, patient_id, weight_kg, height_cm, antecedentes, allergies, habitual_medication,
			chronic_conditions, habits, general_observations, created_at, updated_at
	`,
		normalized.PatientID,
		normalized.SetWeightKG, normalized.WeightKG,
		normalized.SetHeightCM, normalized.HeightCM,
		normalized.SetAntecedentes, normalized.Antecedentes,
		normalized.SetAllergies, normalized.Allergies,
		normalized.SetHabitualMedication, normalized.HabitualMedication,
		normalized.SetChronicConditions, normalized.ChronicConditions,
		normalized.SetHabits, normalized.Habits,
		normalized.SetGeneralObservations, normalized.GeneralObservations,
	))
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

func validateUpdateClinicalHistoryParams(params UpdateClinicalHistoryParams) (UpdateClinicalHistoryParams, error) {
	normalizedPatientID, err := validateClinicalHistoryPatientID(params.PatientID)
	if err != nil {
		return UpdateClinicalHistoryParams{}, err
	}

	normalized := UpdateClinicalHistoryParams{
		PatientID:              normalizedPatientID,
		WeightKG:               params.WeightKG,
		HeightCM:               params.HeightCM,
		SetWeightKG:            params.WeightKG != nil || params.SetWeightKG,
		SetHeightCM:            params.HeightCM != nil || params.SetHeightCM,
		SetAntecedentes:        params.Antecedentes != nil || params.SetAntecedentes,
		SetAllergies:           params.Allergies != nil || params.SetAllergies,
		SetHabitualMedication:  params.HabitualMedication != nil || params.SetHabitualMedication,
		SetChronicConditions:   params.ChronicConditions != nil || params.SetChronicConditions,
		SetHabits:              params.Habits != nil || params.SetHabits,
		SetGeneralObservations: params.GeneralObservations != nil || params.SetGeneralObservations,
	}

	if err := validateClinicalMeasurement(normalized.WeightKG, maxWeightKG); err != nil {
		return UpdateClinicalHistoryParams{}, err
	}
	if err := validateClinicalMeasurement(normalized.HeightCM, maxHeightCM); err != nil {
		return UpdateClinicalHistoryParams{}, err
	}

	if normalized.Antecedentes, err = normalizeClinicalHistoryText(params.Antecedentes); err != nil {
		return UpdateClinicalHistoryParams{}, err
	}
	if normalized.Allergies, err = normalizeClinicalHistoryText(params.Allergies); err != nil {
		return UpdateClinicalHistoryParams{}, err
	}
	if normalized.HabitualMedication, err = normalizeClinicalHistoryText(params.HabitualMedication); err != nil {
		return UpdateClinicalHistoryParams{}, err
	}
	if normalized.ChronicConditions, err = normalizeClinicalHistoryText(params.ChronicConditions); err != nil {
		return UpdateClinicalHistoryParams{}, err
	}
	if normalized.Habits, err = normalizeClinicalHistoryText(params.Habits); err != nil {
		return UpdateClinicalHistoryParams{}, err
	}
	if normalized.GeneralObservations, err = normalizeClinicalHistoryText(params.GeneralObservations); err != nil {
		return UpdateClinicalHistoryParams{}, err
	}

	return normalized, nil
}

func validateClinicalHistoryPatientID(patientID string) (string, error) {
	normalizedPatientID := strings.TrimSpace(patientID)
	if _, err := uuid.Parse(normalizedPatientID); err != nil {
		return "", ErrNotFound
	}

	return normalizedPatientID, nil
}

func validateClinicalMeasurement(value *float64, max float64) error {
	if value == nil {
		return nil
	}
	if *value <= 0 || *value > max {
		return ErrValidation
	}

	return nil
}

func normalizeClinicalHistoryText(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil, nil
	}
	if len([]rune(normalized)) > maxClinicalHistoryTextLength {
		return nil, ErrValidation
	}

	return &normalized, nil
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

type clinicalHistoryScanner interface {
	Scan(dest ...any) error
}

func scanClinicalHistory(scanner clinicalHistoryScanner) (ClinicalHistory, error) {
	var history ClinicalHistory
	var weightKG, heightCM sql.NullFloat64
	var antecedentes, allergies, habitualMedication, chronicConditions, habits, generalObservations sql.NullString

	err := scanner.Scan(
		&history.ID,
		&history.PatientID,
		&weightKG,
		&heightCM,
		&antecedentes,
		&allergies,
		&habitualMedication,
		&chronicConditions,
		&habits,
		&generalObservations,
		&history.CreatedAt,
		&history.UpdatedAt,
	)
	if err != nil {
		return ClinicalHistory{}, err
	}

	history.WeightKG = nullFloatPtr(weightKG)
	history.HeightCM = nullFloatPtr(heightCM)
	history.Antecedentes = nullStringPtr(antecedentes)
	history.Allergies = nullStringPtr(allergies)
	history.HabitualMedication = nullStringPtr(habitualMedication)
	history.ChronicConditions = nullStringPtr(chronicConditions)
	history.Habits = nullStringPtr(habits)
	history.GeneralObservations = nullStringPtr(generalObservations)

	return history, nil
}

func nullFloatPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}

	return &value.Float64
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
