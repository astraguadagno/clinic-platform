package appointments

import (
	"context"
	"database/sql"
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
