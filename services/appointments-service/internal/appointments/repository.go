package appointments

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	ErrNotFound   = errors.New("appointments resource not found")
	ErrConflict   = errors.New("appointments conflict")
	ErrValidation = errors.New("appointments validation failed")
)

const availabilitySlotsNoOverlapConstraint = "availability_slots_no_overlap"

type Repository struct {
	db *sql.DB
}

type BulkCreateSlotsParams struct {
	ProfessionalID      string `json:"professional_id"`
	Date                string `json:"date"`
	StartTime           string `json:"start_time"`
	EndTime             string `json:"end_time"`
	SlotDurationMinutes int    `json:"slot_duration_minutes"`
}

type SlotFilters struct {
	ProfessionalID string
	Status         string
	Date           string
}

type AvailabilitySlot struct {
	ID             string    `json:"id"`
	ProfessionalID string    `json:"professional_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateAppointmentParams struct {
	SlotID         string `json:"slot_id"`
	PatientID      string `json:"patient_id"`
	ProfessionalID string `json:"professional_id"`
}

type AppointmentFilters struct {
	ProfessionalID string
	PatientID      string
	Status         string
	Date           string
}

type Appointment struct {
	ID             string     `json:"id"`
	SlotID         string     `json:"slot_id"`
	ProfessionalID string     `json:"professional_id"`
	PatientID      string     `json:"patient_id"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
}

type CreateTemplateParams struct {
	ProfessionalID string          `json:"professional_id"`
	EffectiveFrom  string          `json:"effective_from"`
	Recurrence     json.RawMessage `json:"recurrence"`
	CreatedBy      *string         `json:"created_by,omitempty"`
	Reason         *string         `json:"reason,omitempty"`
}

type CreateScheduleBlockParams struct {
	ProfessionalID string  `json:"professional_id"`
	Scope          string  `json:"scope"`
	BlockDate      *string `json:"block_date,omitempty"`
	StartDate      *string `json:"start_date,omitempty"`
	EndDate        *string `json:"end_date,omitempty"`
	DayOfWeek      *int    `json:"day_of_week,omitempty"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
	TemplateID     *string `json:"template_id,omitempty"`
}

type UpdateScheduleBlockParams struct {
	ProfessionalID string  `json:"professional_id"`
	Scope          string  `json:"scope"`
	BlockDate      *string `json:"block_date,omitempty"`
	StartDate      *string `json:"start_date,omitempty"`
	EndDate        *string `json:"end_date,omitempty"`
	DayOfWeek      *int    `json:"day_of_week,omitempty"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
	TemplateID     *string `json:"template_id,omitempty"`
}

type ScheduleBlockFilters struct {
	ProfessionalID string
	TemplateID     string
	Scope          string
}

func ValidateBulkCreateSlotsParams(params BulkCreateSlotsParams) error {
	_, _, _, _, err := parseBulkSlotInputs(params)
	return err
}

func ValidateCreateAppointmentParams(params CreateAppointmentParams) error {
	return validateAppointmentParams(params)
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func (r *Repository) CreateSlotsBulk(ctx context.Context, params BulkCreateSlotsParams) ([]AvailabilitySlot, error) {
	professionalID, startAt, endAt, duration, err := parseBulkSlotInputs(params)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO availability_slots (professional_id, start_time, end_time)
		VALUES ($1, $2, $3)
		RETURNING id, professional_id, start_time, end_time, status, created_at, updated_at
	`

	slots := make([]AvailabilitySlot, 0)
	for current := startAt; current.Before(endAt); current = current.Add(duration) {
		next := current.Add(duration)
		row := tx.QueryRowContext(ctx, query, professionalID, current, next)
		slot, scanErr := scanSlot(row)
		if scanErr != nil {
			if isConflictViolation(scanErr) {
				return nil, ErrConflict
			}
			return nil, scanErr
		}
		slots = append(slots, slot)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return slots, nil
}

func (r *Repository) ListSlots(ctx context.Context, filters SlotFilters) ([]AvailabilitySlot, error) {
	baseQuery := `
		SELECT id, professional_id, start_time, end_time, status, created_at, updated_at
		FROM availability_slots
	`

	where := make([]string, 0)
	args := make([]any, 0)

	if filters.ProfessionalID != "" {
		where = append(where, fmt.Sprintf("professional_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filters.ProfessionalID))
	}
	if filters.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filters.Status))
	}
	if filters.Date != "" {
		date, err := time.Parse("2006-01-02", strings.TrimSpace(filters.Date))
		if err != nil {
			return nil, ErrValidation
		}
		where = append(where, fmt.Sprintf("DATE(start_time AT TIME ZONE 'UTC') = $%d", len(args)+1))
		args = append(args, date.Format("2006-01-02"))
	}

	query := baseQuery
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY start_time"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	slots := make([]AvailabilitySlot, 0)
	for rows.Next() {
		slot, scanErr := scanSlot(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		slots = append(slots, slot)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return slots, nil
}

func (r *Repository) CreateAppointment(ctx context.Context, params CreateAppointmentParams) (Appointment, error) {
	if err := validateAppointmentParams(params); err != nil {
		return Appointment{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Appointment{}, err
	}
	defer tx.Rollback()

	var slot AvailabilitySlot
	row := tx.QueryRowContext(ctx, `
		SELECT id, professional_id, start_time, end_time, status, created_at, updated_at
		FROM availability_slots
		WHERE id = $1
		FOR UPDATE
	`, strings.TrimSpace(params.SlotID))
	slot, err = scanSlot(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Appointment{}, ErrNotFound
	}
	if err != nil {
		return Appointment{}, err
	}

	if slot.Status != "available" {
		return Appointment{}, ErrConflict
	}
	if slot.ProfessionalID != strings.TrimSpace(params.ProfessionalID) {
		return Appointment{}, ErrValidation
	}

	appointmentRow := tx.QueryRowContext(ctx, `
		INSERT INTO appointments (slot_id, professional_id, patient_id)
		VALUES ($1, $2, $3)
		RETURNING id, slot_id, professional_id, patient_id, status, created_at, updated_at, cancelled_at
	`, strings.TrimSpace(params.SlotID), strings.TrimSpace(params.ProfessionalID), strings.TrimSpace(params.PatientID))

	appointment, err := scanAppointment(appointmentRow)
	if err != nil {
		if isConflictViolation(err) {
			return Appointment{}, ErrConflict
		}
		return Appointment{}, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE availability_slots
		SET status = 'booked', updated_at = NOW()
		WHERE id = $1
	`, strings.TrimSpace(params.SlotID))
	if err != nil {
		return Appointment{}, err
	}

	if err := tx.Commit(); err != nil {
		return Appointment{}, err
	}

	return appointment, nil
}

func (r *Repository) ListAppointments(ctx context.Context, filters AppointmentFilters) ([]Appointment, error) {
	baseQuery := `
		SELECT id, slot_id, professional_id, patient_id, status, created_at, updated_at, cancelled_at
		FROM appointments
	`

	where := make([]string, 0)
	args := make([]any, 0)

	if filters.ProfessionalID != "" {
		where = append(where, fmt.Sprintf("professional_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filters.ProfessionalID))
	}
	if filters.PatientID != "" {
		where = append(where, fmt.Sprintf("patient_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filters.PatientID))
	}
	if filters.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filters.Status))
	}
	if filters.Date != "" {
		date, err := time.Parse("2006-01-02", strings.TrimSpace(filters.Date))
		if err != nil {
			return nil, ErrValidation
		}
		where = append(where, fmt.Sprintf(`slot_id IN (
			SELECT id FROM availability_slots WHERE DATE(start_time AT TIME ZONE 'UTC') = $%d
		)`, len(args)+1))
		args = append(args, date.Format("2006-01-02"))
	}

	query := baseQuery
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appointments := make([]Appointment, 0)
	for rows.Next() {
		appointment, scanErr := scanAppointment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		appointments = append(appointments, appointment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return appointments, nil
}

func (r *Repository) GetAppointmentByID(ctx context.Context, appointmentID string) (Appointment, error) {
	if _, err := uuid.Parse(strings.TrimSpace(appointmentID)); err != nil {
		return Appointment{}, ErrValidation
	}

	appointment, err := scanAppointment(r.db.QueryRowContext(ctx, `
		SELECT id, slot_id, professional_id, patient_id, status, created_at, updated_at, cancelled_at
		FROM appointments
		WHERE id = $1
	`, strings.TrimSpace(appointmentID)))
	if errors.Is(err, sql.ErrNoRows) {
		return Appointment{}, ErrNotFound
	}
	if err != nil {
		return Appointment{}, err
	}

	return appointment, nil
}

func (r *Repository) CancelAppointment(ctx context.Context, appointmentID string) (Appointment, error) {
	if _, err := uuid.Parse(strings.TrimSpace(appointmentID)); err != nil {
		return Appointment{}, ErrValidation
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Appointment{}, err
	}
	defer tx.Rollback()

	appointment, err := scanAppointment(tx.QueryRowContext(ctx, `
		SELECT id, slot_id, professional_id, patient_id, status, created_at, updated_at, cancelled_at
		FROM appointments
		WHERE id = $1
		FOR UPDATE
	`, strings.TrimSpace(appointmentID)))
	if errors.Is(err, sql.ErrNoRows) {
		return Appointment{}, ErrNotFound
	}
	if err != nil {
		return Appointment{}, err
	}
	if appointment.Status == "cancelled" {
		return Appointment{}, ErrConflict
	}

	now := time.Now().UTC()
	updated, err := scanAppointment(tx.QueryRowContext(ctx, `
		UPDATE appointments
		SET status = 'cancelled', cancelled_at = $2, updated_at = $2
		WHERE id = $1
		RETURNING id, slot_id, professional_id, patient_id, status, created_at, updated_at, cancelled_at
	`, strings.TrimSpace(appointmentID), now))
	if err != nil {
		return Appointment{}, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE availability_slots
		SET status = 'available', updated_at = $2
		WHERE id = $1
	`, appointment.SlotID, now)
	if err != nil {
		return Appointment{}, err
	}

	if err := tx.Commit(); err != nil {
		return Appointment{}, err
	}

	return updated, nil
}

func (r *Repository) CreateTemplate(ctx context.Context, params CreateTemplateParams) (ScheduleTemplate, error) {
	professionalID, effectiveFrom, recurrence, createdBy, reason, err := validateCreateTemplateParams(params)
	if err != nil {
		return ScheduleTemplate{}, err
	}

	row := r.db.QueryRowContext(ctx, `
		WITH upserted_template AS (
			INSERT INTO schedule_templates (professional_id)
			VALUES ($1)
			ON CONFLICT (professional_id)
			DO UPDATE SET updated_at = NOW()
			RETURNING id, professional_id, created_at, updated_at
		), inserted_version AS (
			INSERT INTO schedule_template_versions (template_id, version_number, effective_from, recurrence, created_by, reason)
			SELECT
				t.id,
				COALESCE((
					SELECT MAX(version_number)
					FROM schedule_template_versions
					WHERE template_id = t.id
				), 0) + 1,
				$2,
				$3,
				$4,
				$5
			FROM upserted_template t
			RETURNING id, template_id, version_number, effective_from, recurrence, created_at, created_by, reason
		)
		SELECT
			t.id,
			t.professional_id,
			t.created_at,
			t.updated_at,
			v.id,
			v.version_number,
			v.effective_from,
			v.recurrence,
			v.created_at,
			v.created_by,
			v.reason
		FROM upserted_template t
		JOIN inserted_version v ON v.template_id = t.id
	`, professionalID, effectiveFrom, recurrence, createdBy, reason)

	template, err := scanTemplateWithVersion(row)
	if err != nil {
		if isConflictViolation(err) {
			return ScheduleTemplate{}, ErrConflict
		}
		return ScheduleTemplate{}, err
	}

	return template, nil
}

func (r *Repository) GetTemplate(ctx context.Context, templateID string) (ScheduleTemplate, error) {
	if _, err := uuid.Parse(strings.TrimSpace(templateID)); err != nil {
		return ScheduleTemplate{}, ErrValidation
	}

	template, err := scanTemplate(r.db.QueryRowContext(ctx, `
		SELECT id, professional_id, created_at, updated_at
		FROM schedule_templates
		WHERE id = $1
	`, strings.TrimSpace(templateID)))
	if errors.Is(err, sql.ErrNoRows) {
		return ScheduleTemplate{}, ErrNotFound
	}
	if err != nil {
		return ScheduleTemplate{}, err
	}

	return template, nil
}

func (r *Repository) ListTemplateVersions(ctx context.Context, templateID string) ([]ScheduleTemplateVersion, error) {
	if _, err := uuid.Parse(strings.TrimSpace(templateID)); err != nil {
		return nil, ErrValidation
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, template_id, version_number, effective_from, recurrence, created_at, created_by, reason
		FROM schedule_template_versions
		WHERE template_id = $1
		ORDER BY effective_from DESC, version_number DESC
	`, strings.TrimSpace(templateID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make([]ScheduleTemplateVersion, 0)
	for rows.Next() {
		version, scanErr := scanTemplateVersion(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		versions = append(versions, version)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return versions, nil
}

func (r *Repository) GetActiveTemplate(ctx context.Context, professionalID string, effectiveDate string) (ScheduleTemplateVersion, error) {
	validatedProfessionalID, validatedEffectiveDate, err := validateActiveTemplateLookupParams(professionalID, effectiveDate)
	if err != nil {
		return ScheduleTemplateVersion{}, err
	}

	version, err := scanTemplateVersion(r.db.QueryRowContext(ctx, `
		SELECT v.id, v.template_id, v.version_number, v.effective_from, v.recurrence, v.created_at, v.created_by, v.reason
		FROM schedule_template_versions v
		JOIN schedule_templates t ON t.id = v.template_id
		WHERE t.professional_id = $1
		  AND v.effective_from <= $2
		ORDER BY v.effective_from DESC, v.version_number DESC
		LIMIT 1
	`, validatedProfessionalID, validatedEffectiveDate))
	if errors.Is(err, sql.ErrNoRows) {
		return ScheduleTemplateVersion{}, ErrNotFound
	}
	if err != nil {
		return ScheduleTemplateVersion{}, err
	}

	return version, nil
}

func (r *Repository) CreateScheduleBlock(ctx context.Context, params CreateScheduleBlockParams) (ScheduleBlock, error) {
	validated, err := validateScheduleBlockParams(params.ProfessionalID, params.Scope, params.BlockDate, params.StartDate, params.EndDate, params.DayOfWeek, params.StartTime, params.EndTime, params.TemplateID)
	if err != nil {
		return ScheduleBlock{}, err
	}

	block, err := scanScheduleBlock(r.db.QueryRowContext(ctx, `
		INSERT INTO schedule_blocks (professional_id, scope, block_date, start_date, end_date, day_of_week, start_time, end_time, template_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, professional_id, scope, block_date, start_date, end_date, day_of_week, TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'), template_id, created_at, updated_at
	`, validated.professionalID, validated.scope, validated.blockDate, validated.startDate, validated.endDate, validated.dayOfWeek, validated.startTime, validated.endTime, validated.templateID))
	if err != nil {
		if isConflictViolation(err) {
			return ScheduleBlock{}, ErrConflict
		}
		return ScheduleBlock{}, err
	}

	return block, nil
}

func (r *Repository) GetScheduleBlock(ctx context.Context, blockID string) (ScheduleBlock, error) {
	if _, err := uuid.Parse(strings.TrimSpace(blockID)); err != nil {
		return ScheduleBlock{}, ErrValidation
	}

	block, err := scanScheduleBlock(r.db.QueryRowContext(ctx, `
		SELECT id, professional_id, scope, block_date, start_date, end_date, day_of_week, TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'), template_id, created_at, updated_at
		FROM schedule_blocks
		WHERE id = $1
	`, strings.TrimSpace(blockID)))
	if errors.Is(err, sql.ErrNoRows) {
		return ScheduleBlock{}, ErrNotFound
	}
	if err != nil {
		return ScheduleBlock{}, err
	}

	return block, nil
}

func (r *Repository) ListScheduleBlocks(ctx context.Context, filters ScheduleBlockFilters) ([]ScheduleBlock, error) {
	if err := validateScheduleBlockFilters(filters); err != nil {
		return nil, err
	}

	baseQuery := `
		SELECT id, professional_id, scope, block_date, start_date, end_date, day_of_week, TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'), template_id, created_at, updated_at
		FROM schedule_blocks
	`

	where := make([]string, 0)
	args := make([]any, 0)

	if filters.ProfessionalID != "" {
		where = append(where, fmt.Sprintf("professional_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filters.ProfessionalID))
	}
	if filters.TemplateID != "" {
		where = append(where, fmt.Sprintf("template_id = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filters.TemplateID))
	}
	if filters.Scope != "" {
		where = append(where, fmt.Sprintf("scope = $%d", len(args)+1))
		args = append(args, strings.TrimSpace(filters.Scope))
	}

	query := baseQuery
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY COALESCE(block_date, start_date) NULLS LAST, day_of_week NULLS LAST, start_time, created_at"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocks := make([]ScheduleBlock, 0)
	for rows.Next() {
		block, scanErr := scanScheduleBlock(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}

func (r *Repository) UpdateScheduleBlock(ctx context.Context, blockID string, params UpdateScheduleBlockParams) (ScheduleBlock, error) {
	trimmedBlockID := strings.TrimSpace(blockID)
	if _, err := uuid.Parse(trimmedBlockID); err != nil {
		return ScheduleBlock{}, ErrValidation
	}

	validated, err := validateScheduleBlockParams(params.ProfessionalID, params.Scope, params.BlockDate, params.StartDate, params.EndDate, params.DayOfWeek, params.StartTime, params.EndTime, params.TemplateID)
	if err != nil {
		return ScheduleBlock{}, err
	}

	block, err := scanScheduleBlock(r.db.QueryRowContext(ctx, `
		UPDATE schedule_blocks
		SET professional_id = $2,
			scope = $3,
			block_date = $4,
			start_date = $5,
			end_date = $6,
			day_of_week = $7,
			start_time = $8,
			end_time = $9,
			template_id = $10,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, professional_id, scope, block_date, start_date, end_date, day_of_week, TO_CHAR(start_time, 'HH24:MI'), TO_CHAR(end_time, 'HH24:MI'), template_id, created_at, updated_at
	`, trimmedBlockID, validated.professionalID, validated.scope, validated.blockDate, validated.startDate, validated.endDate, validated.dayOfWeek, validated.startTime, validated.endTime, validated.templateID))
	if errors.Is(err, sql.ErrNoRows) {
		return ScheduleBlock{}, ErrNotFound
	}
	if err != nil {
		if isConflictViolation(err) {
			return ScheduleBlock{}, ErrConflict
		}
		return ScheduleBlock{}, err
	}

	return block, nil
}

func (r *Repository) DeleteScheduleBlock(ctx context.Context, blockID string) error {
	trimmedBlockID := strings.TrimSpace(blockID)
	if _, err := uuid.Parse(trimmedBlockID); err != nil {
		return ErrValidation
	}

	var deletedID string
	err := r.db.QueryRowContext(ctx, `
		DELETE FROM schedule_blocks
		WHERE id = $1
		RETURNING id
	`, trimmedBlockID).Scan(&deletedID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	return nil
}

func parseBulkSlotInputs(params BulkCreateSlotsParams) (string, time.Time, time.Time, time.Duration, error) {
	professionalID := strings.TrimSpace(params.ProfessionalID)
	if _, err := uuid.Parse(professionalID); err != nil {
		return "", time.Time{}, time.Time{}, 0, ErrValidation
	}
	if params.SlotDurationMinutes <= 0 {
		return "", time.Time{}, time.Time{}, 0, ErrValidation
	}

	date, err := time.Parse("2006-01-02", strings.TrimSpace(params.Date))
	if err != nil {
		return "", time.Time{}, time.Time{}, 0, ErrValidation
	}

	startClock, err := time.Parse("15:04", strings.TrimSpace(params.StartTime))
	if err != nil {
		return "", time.Time{}, time.Time{}, 0, ErrValidation
	}
	endClock, err := time.Parse("15:04", strings.TrimSpace(params.EndTime))
	if err != nil {
		return "", time.Time{}, time.Time{}, 0, ErrValidation
	}

	startAt := time.Date(date.Year(), date.Month(), date.Day(), startClock.Hour(), startClock.Minute(), 0, 0, time.UTC)
	endAt := time.Date(date.Year(), date.Month(), date.Day(), endClock.Hour(), endClock.Minute(), 0, 0, time.UTC)
	duration := time.Duration(params.SlotDurationMinutes) * time.Minute

	if !startAt.Before(endAt) {
		return "", time.Time{}, time.Time{}, 0, ErrValidation
	}
	if startAt.Add(duration).After(endAt) {
		return "", time.Time{}, time.Time{}, 0, ErrValidation
	}
	if endAt.Sub(startAt)%duration != 0 {
		return "", time.Time{}, time.Time{}, 0, ErrValidation
	}

	return professionalID, startAt, endAt, duration, nil
}

func validateAppointmentParams(params CreateAppointmentParams) error {
	if _, err := uuid.Parse(strings.TrimSpace(params.SlotID)); err != nil {
		return ErrValidation
	}
	if _, err := uuid.Parse(strings.TrimSpace(params.PatientID)); err != nil {
		return ErrValidation
	}
	if _, err := uuid.Parse(strings.TrimSpace(params.ProfessionalID)); err != nil {
		return ErrValidation
	}

	return nil
}

func validateCreateTemplateParams(params CreateTemplateParams) (string, time.Time, json.RawMessage, *string, *string, error) {
	professionalID := strings.TrimSpace(params.ProfessionalID)
	if _, err := uuid.Parse(professionalID); err != nil {
		return "", time.Time{}, nil, nil, nil, ErrValidation
	}

	effectiveFrom, err := time.Parse("2006-01-02", strings.TrimSpace(params.EffectiveFrom))
	if err != nil {
		return "", time.Time{}, nil, nil, nil, ErrValidation
	}

	recurrence := make(json.RawMessage, len(params.Recurrence))
	copy(recurrence, params.Recurrence)
	if !json.Valid(recurrence) {
		return "", time.Time{}, nil, nil, nil, ErrValidation
	}

	var recurrencePayload map[string]any
	if err := json.Unmarshal(recurrence, &recurrencePayload); err != nil || recurrencePayload == nil {
		return "", time.Time{}, nil, nil, nil, ErrValidation
	}

	createdBy, err := normalizeOptionalUUID(params.CreatedBy)
	if err != nil {
		return "", time.Time{}, nil, nil, nil, err
	}

	reason := normalizeOptionalString(params.Reason)

	return professionalID, effectiveFrom, recurrence, createdBy, reason, nil
}

func validateActiveTemplateLookupParams(professionalIDValue, effectiveDateValue string) (string, time.Time, error) {
	professionalID := strings.TrimSpace(professionalIDValue)
	if _, err := uuid.Parse(professionalID); err != nil {
		return "", time.Time{}, ErrValidation
	}

	effectiveDate, err := time.Parse("2006-01-02", strings.TrimSpace(effectiveDateValue))
	if err != nil {
		return "", time.Time{}, ErrValidation
	}

	return professionalID, effectiveDate, nil
}

type validatedScheduleBlockParams struct {
	professionalID string
	scope          string
	blockDate      *time.Time
	startDate      *time.Time
	endDate        *time.Time
	dayOfWeek      *int
	startTime      string
	endTime        string
	templateID     *string
}

func validateScheduleBlockParams(professionalIDValue, scopeValue string, blockDateValue, startDateValue, endDateValue *string, dayOfWeekValue *int, startTimeValue, endTimeValue string, templateIDValue *string) (validatedScheduleBlockParams, error) {
	professionalID := strings.TrimSpace(professionalIDValue)
	if _, err := uuid.Parse(professionalID); err != nil {
		return validatedScheduleBlockParams{}, ErrValidation
	}

	scope := strings.TrimSpace(scopeValue)
	if scope != "single" && scope != "range" && scope != "template" {
		return validatedScheduleBlockParams{}, ErrValidation
	}

	blockDate, err := normalizeOptionalDate(blockDateValue)
	if err != nil {
		return validatedScheduleBlockParams{}, err
	}
	startDate, err := normalizeOptionalDate(startDateValue)
	if err != nil {
		return validatedScheduleBlockParams{}, err
	}
	endDate, err := normalizeOptionalDate(endDateValue)
	if err != nil {
		return validatedScheduleBlockParams{}, err
	}

	startTime, err := normalizeClockString(startTimeValue)
	if err != nil {
		return validatedScheduleBlockParams{}, err
	}
	endTime, err := normalizeClockString(endTimeValue)
	if err != nil {
		return validatedScheduleBlockParams{}, err
	}
	if startTime >= endTime {
		return validatedScheduleBlockParams{}, ErrValidation
	}

	templateID, err := normalizeOptionalUUID(templateIDValue)
	if err != nil {
		return validatedScheduleBlockParams{}, err
	}

	var dayOfWeek *int
	if dayOfWeekValue != nil {
		day := *dayOfWeekValue
		if day < 1 || day > 7 {
			return validatedScheduleBlockParams{}, ErrValidation
		}
		dayOfWeek = &day
	}

	switch scope {
	case "single":
		if blockDate == nil || startDate != nil || endDate != nil || dayOfWeek != nil || templateID != nil {
			return validatedScheduleBlockParams{}, ErrValidation
		}
	case "range":
		if blockDate != nil || startDate == nil || endDate == nil || dayOfWeek != nil || templateID != nil {
			return validatedScheduleBlockParams{}, ErrValidation
		}
		if startDate.After(*endDate) {
			return validatedScheduleBlockParams{}, ErrValidation
		}
	case "template":
		if blockDate != nil || startDate != nil || endDate != nil || dayOfWeek == nil || templateID == nil {
			return validatedScheduleBlockParams{}, ErrValidation
		}
	}

	return validatedScheduleBlockParams{
		professionalID: professionalID,
		scope:          scope,
		blockDate:      blockDate,
		startDate:      startDate,
		endDate:        endDate,
		dayOfWeek:      dayOfWeek,
		startTime:      startTime,
		endTime:        endTime,
		templateID:     templateID,
	}, nil
}

func validateScheduleBlockFilters(filters ScheduleBlockFilters) error {
	if filters.ProfessionalID != "" {
		if _, err := uuid.Parse(strings.TrimSpace(filters.ProfessionalID)); err != nil {
			return ErrValidation
		}
	}
	if filters.TemplateID != "" {
		if _, err := uuid.Parse(strings.TrimSpace(filters.TemplateID)); err != nil {
			return ErrValidation
		}
	}
	if filters.Scope != "" {
		scope := strings.TrimSpace(filters.Scope)
		if scope != "single" && scope != "range" && scope != "template" {
			return ErrValidation
		}
	}

	return nil
}

func normalizeOptionalDate(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, ErrValidation
	}

	return &parsed, nil
}

func normalizeClockString(value string) (string, error) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return "", ErrValidation
	}

	return parsed.Format("15:04"), nil
}

func normalizeOptionalUUID(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(trimmed); err != nil {
		return nil, ErrValidation
	}

	return &trimmed, nil
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func isConflictViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "23505" || (pgErr.Code == "23P01" && pgErr.ConstraintName == availabilitySlotsNoOverlapConstraint)
}

type slotScanner interface {
	Scan(dest ...any) error
}

func scanSlot(scanner slotScanner) (AvailabilitySlot, error) {
	var slot AvailabilitySlot
	err := scanner.Scan(
		&slot.ID,
		&slot.ProfessionalID,
		&slot.StartTime,
		&slot.EndTime,
		&slot.Status,
		&slot.CreatedAt,
		&slot.UpdatedAt,
	)
	if err != nil {
		return AvailabilitySlot{}, err
	}
	return slot, nil
}

type appointmentScanner interface {
	Scan(dest ...any) error
}

type templateScanner interface {
	Scan(dest ...any) error
}

type scheduleBlockScanner interface {
	Scan(dest ...any) error
}

func scanAppointment(scanner appointmentScanner) (Appointment, error) {
	var appointment Appointment
	var cancelledAt sql.NullTime
	err := scanner.Scan(
		&appointment.ID,
		&appointment.SlotID,
		&appointment.ProfessionalID,
		&appointment.PatientID,
		&appointment.Status,
		&appointment.CreatedAt,
		&appointment.UpdatedAt,
		&cancelledAt,
	)
	if err != nil {
		return Appointment{}, err
	}
	if cancelledAt.Valid {
		appointment.CancelledAt = &cancelledAt.Time
	}
	return appointment, nil
}

func scanTemplate(scanner templateScanner) (ScheduleTemplate, error) {
	var template ScheduleTemplate
	err := scanner.Scan(
		&template.ID,
		&template.ProfessionalID,
		&template.CreatedAt,
		&template.UpdatedAt,
	)
	if err != nil {
		return ScheduleTemplate{}, err
	}

	return template, nil
}

func scanTemplateVersion(scanner templateScanner) (ScheduleTemplateVersion, error) {
	var (
		version           ScheduleTemplateVersion
		recurrence        []byte
		createdBy, reason sql.NullString
	)

	err := scanner.Scan(
		&version.ID,
		&version.TemplateID,
		&version.VersionNumber,
		&version.EffectiveFrom,
		&recurrence,
		&version.CreatedAt,
		&createdBy,
		&reason,
	)
	if err != nil {
		return ScheduleTemplateVersion{}, err
	}

	version.Recurrence = make(json.RawMessage, len(recurrence))
	copy(version.Recurrence, recurrence)
	if createdBy.Valid {
		version.CreatedBy = &createdBy.String
	}
	if reason.Valid {
		version.Reason = &reason.String
	}

	return version, nil
}

func scanTemplateWithVersion(scanner templateScanner) (ScheduleTemplate, error) {
	template, err := scanTemplateVersionJoin(scanner)
	if err != nil {
		return ScheduleTemplate{}, err
	}

	return template, nil
}

func scanScheduleBlock(scanner scheduleBlockScanner) (ScheduleBlock, error) {
	var (
		block                         ScheduleBlock
		blockDate, startDate, endDate sql.NullTime
		dayOfWeek                     sql.NullInt64
		templateID                    sql.NullString
	)

	err := scanner.Scan(
		&block.ID,
		&block.ProfessionalID,
		&block.Scope,
		&blockDate,
		&startDate,
		&endDate,
		&dayOfWeek,
		&block.StartTime,
		&block.EndTime,
		&templateID,
		&block.CreatedAt,
		&block.UpdatedAt,
	)
	if err != nil {
		return ScheduleBlock{}, err
	}

	if blockDate.Valid {
		date := blockDate.Time
		block.BlockDate = &date
	}
	if startDate.Valid {
		date := startDate.Time
		block.StartDate = &date
	}
	if endDate.Valid {
		date := endDate.Time
		block.EndDate = &date
	}
	if dayOfWeek.Valid {
		day := int(dayOfWeek.Int64)
		block.DayOfWeek = &day
	}
	if templateID.Valid {
		block.TemplateID = &templateID.String
	}

	return block, nil
}

func scanTemplateVersionJoin(scanner templateScanner) (ScheduleTemplate, error) {
	var (
		template          ScheduleTemplate
		version           ScheduleTemplateVersion
		recurrence        []byte
		createdBy, reason sql.NullString
	)

	err := scanner.Scan(
		&template.ID,
		&template.ProfessionalID,
		&template.CreatedAt,
		&template.UpdatedAt,
		&version.ID,
		&version.VersionNumber,
		&version.EffectiveFrom,
		&recurrence,
		&version.CreatedAt,
		&createdBy,
		&reason,
	)
	if err != nil {
		return ScheduleTemplate{}, err
	}

	version.TemplateID = template.ID
	version.Recurrence = make(json.RawMessage, len(recurrence))
	copy(version.Recurrence, recurrence)
	if createdBy.Valid {
		version.CreatedBy = &createdBy.String
	}
	if reason.Valid {
		version.Reason = &reason.String
	}
	template.Versions = []ScheduleTemplateVersion{version}

	return template, nil
}
