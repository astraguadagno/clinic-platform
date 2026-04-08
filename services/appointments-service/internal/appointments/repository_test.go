package appointments

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

var testSQLDriverCounter atomic.Int64

func TestParseBulkSlotInputs(t *testing.T) {
	tests := []struct {
		name    string
		params  BulkCreateSlotsParams
		wantErr bool
	}{
		{
			name: "valid range",
			params: BulkCreateSlotsParams{
				ProfessionalID:      "550e8400-e29b-41d4-a716-446655440000",
				Date:                "2026-04-10",
				StartTime:           "09:00",
				EndTime:             "10:00",
				SlotDurationMinutes: 30,
			},
		},
		{
			name: "invalid professional id",
			params: BulkCreateSlotsParams{
				ProfessionalID:      "bad-id",
				Date:                "2026-04-10",
				StartTime:           "09:00",
				EndTime:             "10:00",
				SlotDurationMinutes: 30,
			},
			wantErr: true,
		},
		{
			name: "range not divisible by duration",
			params: BulkCreateSlotsParams{
				ProfessionalID:      "550e8400-e29b-41d4-a716-446655440000",
				Date:                "2026-04-10",
				StartTime:           "09:00",
				EndTime:             "10:10",
				SlotDurationMinutes: 30,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, startAt, endAt, duration, err := parseBulkSlotInputs(tt.params)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !errors.Is(err, ErrValidation) {
					t.Fatalf("err = %v, want %v", err, ErrValidation)
				}
				return
			}
			if endAt.Sub(startAt) != time.Hour {
				t.Fatalf("range = %v, want 1h", endAt.Sub(startAt))
			}
			if duration != 30*time.Minute {
				t.Fatalf("duration = %v, want 30m", duration)
			}
		})
	}
}

func TestValidateAppointmentParams(t *testing.T) {
	valid := CreateAppointmentParams{
		SlotID:         "550e8400-e29b-41d4-a716-446655440000",
		PatientID:      "550e8400-e29b-41d4-a716-446655440001",
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440002",
	}

	if err := validateAppointmentParams(valid); err != nil {
		t.Fatalf("valid params error = %v", err)
	}

	invalid := valid
	invalid.PatientID = "bad-id"

	err := validateAppointmentParams(invalid)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want %v", err, ErrValidation)
	}
}

func TestCreateSlotsBulkReturnsConflictOnRealOverlap(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{
		{err: &pgconn.PgError{Code: "23P01", ConstraintName: availabilitySlotsNoOverlapConstraint}},
	}))

	_, err := repo.CreateSlotsBulk(context.Background(), BulkCreateSlotsParams{
		ProfessionalID:      "550e8400-e29b-41d4-a716-446655440000",
		Date:                "2026-04-10",
		StartTime:           "09:15",
		EndTime:             "09:45",
		SlotDurationMinutes: 30,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v, want %v", err, ErrConflict)
	}
}

func TestCreateSlotsBulkAllowsBackToBackSlots(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.April, 10, 9, 30, 0, 0, time.UTC)
	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{
		{row: newSlotRow("slot-1", "550e8400-e29b-41d4-a716-446655440000", start, start.Add(30*time.Minute))},
		{row: newSlotRow("slot-2", "550e8400-e29b-41d4-a716-446655440000", start.Add(30*time.Minute), start.Add(time.Hour))},
	}))

	slots, err := repo.CreateSlotsBulk(context.Background(), BulkCreateSlotsParams{
		ProfessionalID:      "550e8400-e29b-41d4-a716-446655440000",
		Date:                "2026-04-10",
		StartTime:           "09:30",
		EndTime:             "10:30",
		SlotDurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("CreateSlotsBulk error = %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("slots len = %d, want 2", len(slots))
	}
	if !slots[0].EndTime.Equal(slots[1].StartTime) {
		t.Fatalf("expected back-to-back slots, got %s and %s", slots[0].EndTime, slots[1].StartTime)
	}
}

func TestCreateSlotsBulkReturnsConflictWhenBulkHitsExistingSlots(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.April, 10, 9, 0, 0, 0, time.UTC)
	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{
		{row: newSlotRow("slot-1", "550e8400-e29b-41d4-a716-446655440000", start, start.Add(30*time.Minute))},
		{err: &pgconn.PgError{Code: "23P01", ConstraintName: availabilitySlotsNoOverlapConstraint}},
	}))

	_, err := repo.CreateSlotsBulk(context.Background(), BulkCreateSlotsParams{
		ProfessionalID:      "550e8400-e29b-41d4-a716-446655440000",
		Date:                "2026-04-10",
		StartTime:           "09:00",
		EndTime:             "10:00",
		SlotDurationMinutes: 30,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v, want %v", err, ErrConflict)
	}
}

func TestGetAppointmentByIDReturnsAppointment(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 10, 9, 0, 0, 0, time.UTC)
	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newAppointmentRow(
			"550e8400-e29b-41d4-a716-446655440099",
			"550e8400-e29b-41d4-a716-446655440010",
			"550e8400-e29b-41d4-a716-446655440011",
			"550e8400-e29b-41d4-a716-446655440012",
			"booked",
			now,
			now,
			nil,
		),
	}}))

	appointment, err := repo.GetAppointmentByID(context.Background(), "550e8400-e29b-41d4-a716-446655440099")
	if err != nil {
		t.Fatalf("GetAppointmentByID error = %v", err)
	}
	if appointment.ProfessionalID != "550e8400-e29b-41d4-a716-446655440011" {
		t.Fatalf("professional_id = %q, want own agenda id", appointment.ProfessionalID)
	}
}

func TestGetAppointmentByIDReturnsValidationOnBadID(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, nil))
	_, err := repo.GetAppointmentByID(context.Background(), "bad-id")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want %v", err, ErrValidation)
	}
}

func newScriptedDB(t *testing.T, results []scriptedQueryResult) *sql.DB {
	t.Helper()

	driverName := fmt.Sprintf("appointments-scripted-%d", testSQLDriverCounter.Add(1))
	sql.Register(driverName, scriptedDriver{results: results})

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func newSlotRow(id, professionalID string, startTime, endTime time.Time) []driver.Value {
	now := time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC)
	return []driver.Value{id, professionalID, startTime, endTime, "available", now, now}
}

func newAppointmentRow(id, slotID, professionalID, patientID, status string, createdAt, updatedAt time.Time, cancelledAt *time.Time) []driver.Value {
	var cancelled any = nil
	if cancelledAt != nil {
		cancelled = *cancelledAt
	}
	return []driver.Value{id, slotID, professionalID, patientID, status, createdAt, updatedAt, cancelled}
}

type scriptedDriver struct {
	results []scriptedQueryResult
}

func (d scriptedDriver) Open(string) (driver.Conn, error) {
	results := make([]scriptedQueryResult, len(d.results))
	copy(results, d.results)
	return &scriptedConn{results: results}, nil
}

type scriptedConn struct {
	results []scriptedQueryResult
	index   int
}

func (c *scriptedConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (c *scriptedConn) Close() error { return nil }
func (c *scriptedConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *scriptedConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &scriptedTx{}, nil
}

func (c *scriptedConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	if c.index >= len(c.results) {
		return nil, fmt.Errorf("unexpected query #%d", c.index+1)
	}

	result := c.results[c.index]
	c.index++
	if result.err != nil {
		return nil, result.err
	}

	return &scriptedRows{row: result.row}, nil
}

type scriptedTx struct{}

func (scriptedTx) Commit() error   { return nil }
func (scriptedTx) Rollback() error { return nil }

type scriptedQueryResult struct {
	row []driver.Value
	err error
}

type scriptedRows struct {
	row  []driver.Value
	read bool
}

func (r *scriptedRows) Columns() []string {
	switch len(r.row) {
	case 8:
		return []string{"id", "slot_id", "professional_id", "patient_id", "status", "created_at", "updated_at", "cancelled_at"}
	default:
		return []string{"id", "professional_id", "start_time", "end_time", "status", "created_at", "updated_at"}
	}
}

func (r *scriptedRows) Close() error { return nil }

func (r *scriptedRows) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	copy(dest, r.row)
	r.read = true
	return nil
}
