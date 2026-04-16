package appointments

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
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

func TestCreateTemplateCreatesInitialTemplateVersion(t *testing.T) {
	t.Parallel()

	recurrence := json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}}`)
	now := time.Date(2026, time.April, 15, 10, 0, 0, 0, time.UTC)
	createdBy := "550e8400-e29b-41d4-a716-446655440099"
	reason := "initial weekly template"

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newTemplateVersionJoinRow(
			"550e8400-e29b-41d4-a716-446655440010",
			"550e8400-e29b-41d4-a716-446655440000",
			now,
			now,
			"550e8400-e29b-41d4-a716-446655440020",
			1,
			time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
			recurrence,
			now,
			&createdBy,
			&reason,
		),
	}}))

	template, err := repo.CreateTemplate(context.Background(), CreateTemplateParams{
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440000",
		EffectiveFrom:  "2026-05-01",
		Recurrence:     recurrence,
		CreatedBy:      &createdBy,
		Reason:         &reason,
	})
	if err != nil {
		t.Fatalf("CreateTemplate error = %v", err)
	}
	if template.ID != "550e8400-e29b-41d4-a716-446655440010" {
		t.Fatalf("template id = %q, want template id", template.ID)
	}
	if template.ProfessionalID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("professional_id = %q, want own agenda id", template.ProfessionalID)
	}
	if len(template.Versions) != 1 {
		t.Fatalf("versions len = %d, want 1", len(template.Versions))
	}
	if template.Versions[0].VersionNumber != 1 {
		t.Fatalf("version number = %d, want 1", template.Versions[0].VersionNumber)
	}
	if string(template.Versions[0].Recurrence) != string(recurrence) {
		t.Fatalf("recurrence = %s, want %s", template.Versions[0].Recurrence, recurrence)
	}
	if template.Versions[0].CreatedBy == nil || *template.Versions[0].CreatedBy != createdBy {
		t.Fatalf("created_by = %v, want %q", template.Versions[0].CreatedBy, createdBy)
	}
}

func TestGetTemplateReturnsNotFoundWhenTemplateMissing(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{err: sql.ErrNoRows}}))

	_, err := repo.GetTemplate(context.Background(), "550e8400-e29b-41d4-a716-446655440010")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrNotFound)
	}
}

func TestCreateTemplateRejectsInvalidRecurrencePayload(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, nil))

	_, err := repo.CreateTemplate(context.Background(), CreateTemplateParams{
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440000",
		EffectiveFrom:  "2026-05-01",
		Recurrence:     json.RawMessage(`[]`),
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want %v", err, ErrValidation)
	}
}

func TestGetTemplateReturnsTemplate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 15, 10, 0, 0, 0, time.UTC)
	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: []driver.Value{"550e8400-e29b-41d4-a716-446655440010", "550e8400-e29b-41d4-a716-446655440000", now, now},
	}}))

	template, err := repo.GetTemplate(context.Background(), "550e8400-e29b-41d4-a716-446655440010")
	if err != nil {
		t.Fatalf("GetTemplate error = %v", err)
	}
	if template.ProfessionalID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("professional_id = %q, want own agenda id", template.ProfessionalID)
	}
	if !template.CreatedAt.Equal(now) {
		t.Fatalf("created_at = %s, want %s", template.CreatedAt, now)
	}
}

func TestListTemplateVersionsReturnsOrderedVersions(t *testing.T) {
	t.Parallel()

	firstCreatedBy := "550e8400-e29b-41d4-a716-446655440099"
	secondReason := "hours extended"
	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		rows: [][]driver.Value{
			newTemplateVersionRow(
				"550e8400-e29b-41d4-a716-446655440021",
				"550e8400-e29b-41d4-a716-446655440010",
				2,
				time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
				json.RawMessage(`{"monday":{"start_time":"08:00","end_time":"12:00","slot_duration_minutes":30}}`),
				time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC),
				nil,
				&secondReason,
			),
			newTemplateVersionRow(
				"550e8400-e29b-41d4-a716-446655440020",
				"550e8400-e29b-41d4-a716-446655440010",
				1,
				time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
				json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}}`),
				time.Date(2026, time.April, 15, 10, 0, 0, 0, time.UTC),
				&firstCreatedBy,
				nil,
			),
		},
	}}))

	versions, err := repo.ListTemplateVersions(context.Background(), "550e8400-e29b-41d4-a716-446655440010")
	if err != nil {
		t.Fatalf("ListTemplateVersions error = %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("versions len = %d, want 2", len(versions))
	}
	if versions[0].VersionNumber != 2 || versions[1].VersionNumber != 1 {
		t.Fatalf("version order = [%d %d], want [2 1]", versions[0].VersionNumber, versions[1].VersionNumber)
	}
	if versions[1].CreatedBy == nil || *versions[1].CreatedBy != firstCreatedBy {
		t.Fatalf("created_by = %v, want %q", versions[1].CreatedBy, firstCreatedBy)
	}
	if versions[0].Reason == nil || *versions[0].Reason != secondReason {
		t.Fatalf("reason = %v, want %q", versions[0].Reason, secondReason)
	}
}

func TestListTemplateVersionsReturnsValidationOnBadTemplateID(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, nil))

	_, err := repo.ListTemplateVersions(context.Background(), "bad-id")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want %v", err, ErrValidation)
	}
}

func TestGetActiveTemplateReturnsLatestVersionForEffectiveDate(t *testing.T) {
	t.Parallel()

	secondReason := "winter hours"
	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newTemplateVersionRow(
			"550e8400-e29b-41d4-a716-446655440021",
			"550e8400-e29b-41d4-a716-446655440010",
			2,
			time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
			json.RawMessage(`{"monday":{"start_time":"08:00","end_time":"12:00","slot_duration_minutes":30}}`),
			time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC),
			nil,
			&secondReason,
		),
	}}))

	version, err := repo.GetActiveTemplate(context.Background(), "550e8400-e29b-41d4-a716-446655440000", "2026-06-01")
	if err != nil {
		t.Fatalf("GetActiveTemplate error = %v", err)
	}
	if version.VersionNumber != 2 {
		t.Fatalf("version number = %d, want 2", version.VersionNumber)
	}
	if version.Reason == nil || *version.Reason != secondReason {
		t.Fatalf("reason = %v, want %q", version.Reason, secondReason)
	}
	if version.TemplateID != "550e8400-e29b-41d4-a716-446655440010" {
		t.Fatalf("template_id = %q, want template id", version.TemplateID)
	}
}

func TestGetActiveTemplateReturnsNotFoundWhenNoVersionApplies(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{err: sql.ErrNoRows}}))

	_, err := repo.GetActiveTemplate(context.Background(), "550e8400-e29b-41d4-a716-446655440000", "2026-04-30")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrNotFound)
	}
}

func TestGetActiveTemplateReturnsValidationOnBadInputs(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, nil))

	_, err := repo.GetActiveTemplate(context.Background(), "bad-id", "2026-06-01")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("bad professional id err = %v, want %v", err, ErrValidation)
	}

	_, err = repo.GetActiveTemplate(context.Background(), "550e8400-e29b-41d4-a716-446655440000", "06-01-2026")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("bad effective date err = %v, want %v", err, ErrValidation)
	}
}

func TestCreateScheduleBlockReturnsSingleScopeBlock(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC)
	blockDate := time.Date(2026, time.May, 5, 0, 0, 0, 0, time.UTC)
	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newScheduleBlockRow(
			"550e8400-e29b-41d4-a716-446655440301",
			"550e8400-e29b-41d4-a716-446655440300",
			"single",
			&blockDate,
			nil,
			nil,
			nil,
			"09:00",
			"12:00",
			nil,
			now,
			now,
		),
	}}))

	block, err := repo.CreateScheduleBlock(context.Background(), CreateScheduleBlockParams{
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440300",
		Scope:          "single",
		BlockDate:      stringPtr("2026-05-05"),
		StartTime:      "09:00",
		EndTime:        "12:00",
	})
	if err != nil {
		t.Fatalf("CreateScheduleBlock error = %v", err)
	}
	if block.Scope != "single" {
		t.Fatalf("scope = %q, want single", block.Scope)
	}
	if block.BlockDate == nil || !block.BlockDate.Equal(blockDate) {
		t.Fatalf("block_date = %v, want %s", block.BlockDate, blockDate)
	}
	if block.StartTime != "09:00" || block.EndTime != "12:00" {
		t.Fatalf("time range = %s-%s, want 09:00-12:00", block.StartTime, block.EndTime)
	}
	if block.TemplateID != nil {
		t.Fatalf("template_id = %v, want nil", block.TemplateID)
	}
}

func TestCreateScheduleBlockRejectsInvalidTemplateScope(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, nil))

	_, err := repo.CreateScheduleBlock(context.Background(), CreateScheduleBlockParams{
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440300",
		Scope:          "template",
		DayOfWeek:      intPtr(1),
		StartTime:      "10:00",
		EndTime:        "11:00",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want %v", err, ErrValidation)
	}
}

func TestListScheduleBlocksReturnsOrderedBlocks(t *testing.T) {
	t.Parallel()

	firstDate := time.Date(2026, time.May, 8, 0, 0, 0, 0, time.UTC)
	secondDate := time.Date(2026, time.May, 5, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC)
	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		rows: [][]driver.Value{
			newScheduleBlockRow("550e8400-e29b-41d4-a716-446655440302", "550e8400-e29b-41d4-a716-446655440300", "range", nil, &firstDate, &firstDate, nil, "13:00", "15:00", nil, now, now),
			newScheduleBlockRow("550e8400-e29b-41d4-a716-446655440301", "550e8400-e29b-41d4-a716-446655440300", "single", &secondDate, nil, nil, nil, "09:00", "12:00", nil, now, now),
		},
	}}))

	blocks, err := repo.ListScheduleBlocks(context.Background(), ScheduleBlockFilters{ProfessionalID: "550e8400-e29b-41d4-a716-446655440300"})
	if err != nil {
		t.Fatalf("ListScheduleBlocks error = %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("blocks len = %d, want 2", len(blocks))
	}
	if blocks[0].Scope != "range" || blocks[1].Scope != "single" {
		t.Fatalf("block scopes = [%s %s], want [range single]", blocks[0].Scope, blocks[1].Scope)
	}
}

func TestUpdateScheduleBlockReturnsTemplateScopeBlock(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 16, 12, 0, 0, 0, time.UTC)
	templateID := "550e8400-e29b-41d4-a716-446655440399"
	dayOfWeek := 1
	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newScheduleBlockRow(
			"550e8400-e29b-41d4-a716-446655440301",
			"550e8400-e29b-41d4-a716-446655440300",
			"template",
			nil,
			nil,
			nil,
			&dayOfWeek,
			"10:00",
			"11:00",
			&templateID,
			now,
			now,
		),
	}}))

	block, err := repo.UpdateScheduleBlock(context.Background(), "550e8400-e29b-41d4-a716-446655440301", UpdateScheduleBlockParams{
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440300",
		Scope:          "template",
		DayOfWeek:      &dayOfWeek,
		StartTime:      "10:00",
		EndTime:        "11:00",
		TemplateID:     &templateID,
	})
	if err != nil {
		t.Fatalf("UpdateScheduleBlock error = %v", err)
	}
	if block.DayOfWeek == nil || *block.DayOfWeek != 1 {
		t.Fatalf("day_of_week = %v, want 1", block.DayOfWeek)
	}
	if block.TemplateID == nil || *block.TemplateID != templateID {
		t.Fatalf("template_id = %v, want %q", block.TemplateID, templateID)
	}
}

func TestDeleteScheduleBlockReturnsNotFoundWhenMissing(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{err: sql.ErrNoRows}}))

	err := repo.DeleteScheduleBlock(context.Background(), "550e8400-e29b-41d4-a716-446655440301")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrNotFound)
	}
}

func stringPtr(value string) *string { return &value }

func intPtr(value int) *int { return &value }

func newScheduleBlockRow(id, professionalID, scope string, blockDate, startDate, endDate *time.Time, dayOfWeek *int, startTime, endTime string, templateID *string, createdAt, updatedAt time.Time) []driver.Value {
	var blockDateValue any
	if blockDate != nil {
		blockDateValue = *blockDate
	}

	var startDateValue any
	if startDate != nil {
		startDateValue = *startDate
	}

	var endDateValue any
	if endDate != nil {
		endDateValue = *endDate
	}

	var dayOfWeekValue any
	if dayOfWeek != nil {
		dayOfWeekValue = int64(*dayOfWeek)
	}

	return []driver.Value{id, professionalID, scope, blockDateValue, startDateValue, endDateValue, dayOfWeekValue, startTime, endTime, nullableStringValue(templateID), createdAt, updatedAt}
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

func newTemplateVersionJoinRow(templateID, professionalID string, templateCreatedAt, templateUpdatedAt time.Time, versionID string, versionNumber int, effectiveFrom time.Time, recurrence json.RawMessage, versionCreatedAt time.Time, createdBy, reason *string) []driver.Value {
	return []driver.Value{templateID, professionalID, templateCreatedAt, templateUpdatedAt, versionID, versionNumber, effectiveFrom, []byte(recurrence), versionCreatedAt, nullableStringValue(createdBy), nullableStringValue(reason)}
}

func newTemplateVersionRow(id, templateID string, versionNumber int, effectiveFrom time.Time, recurrence json.RawMessage, createdAt time.Time, createdBy, reason *string) []driver.Value {
	return []driver.Value{id, templateID, versionNumber, effectiveFrom, []byte(recurrence), createdAt, nullableStringValue(createdBy), nullableStringValue(reason)}
}

func nullableStringValue(value *string) any {
	if value == nil {
		return nil
	}

	return *value
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

	rows := result.rows
	if len(rows) == 0 && result.row != nil {
		rows = [][]driver.Value{result.row}
	}

	return &scriptedRows{rows: rows}, nil
}

type scriptedTx struct{}

func (scriptedTx) Commit() error   { return nil }
func (scriptedTx) Rollback() error { return nil }

type scriptedQueryResult struct {
	row  []driver.Value
	rows [][]driver.Value
	err  error
}

type scriptedRows struct {
	rows  [][]driver.Value
	index int
}

func (r *scriptedRows) Columns() []string {
	if len(r.rows) == 0 {
		return []string{}
	}

	columns := make([]string, len(r.rows[0]))
	for i := range columns {
		columns[i] = fmt.Sprintf("col_%d", i)
	}

	return columns
}

func (r *scriptedRows) Close() error { return nil }

func (r *scriptedRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}
