package appointments

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"
)

func TestCreateConsultationReturnsScheduledConsultation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 16, 9, 0, 0, 0, time.UTC)
	slotID := "550e8400-e29b-41d4-a716-446655440201"
	notes := "Paciente con estudios previos"
	scheduledStart := time.Date(2026, time.April, 16, 9, 30, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newConsultationRow(
			"550e8400-e29b-41d4-a716-446655440200",
			&slotID,
			"550e8400-e29b-41d4-a716-446655440202",
			"550e8400-e29b-41d4-a716-446655440203",
			ConsultationStatusScheduled,
			ConsultationSourceSecretary,
			&notes,
			scheduledStart,
			scheduledEnd,
			nil,
			nil,
			now,
			now,
			nil,
		),
	}}))

	consultation, err := repo.CreateConsultation(context.Background(), CreateConsultationParams{
		SlotID:         &slotID,
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440202",
		PatientID:      "550e8400-e29b-41d4-a716-446655440203",
		Source:         ConsultationSourceSecretary,
		Notes:          &notes,
	})
	if err != nil {
		t.Fatalf("CreateConsultation error = %v", err)
	}
	if consultation.SlotID == nil || *consultation.SlotID != slotID {
		t.Fatalf("slot_id = %v, want %q", consultation.SlotID, slotID)
	}
	if consultation.Status != ConsultationStatusScheduled {
		t.Fatalf("status = %q, want %q", consultation.Status, ConsultationStatusScheduled)
	}
	if consultation.Source != ConsultationSourceSecretary {
		t.Fatalf("source = %q, want %q", consultation.Source, ConsultationSourceSecretary)
	}
	if consultation.Notes == nil || *consultation.Notes != notes {
		t.Fatalf("notes = %v, want %q", consultation.Notes, notes)
	}
	if !consultation.ScheduledStart.Equal(scheduledStart) {
		t.Fatalf("scheduled_start = %s, want %s", consultation.ScheduledStart, scheduledStart)
	}
	if !consultation.ScheduledEnd.Equal(scheduledEnd) {
		t.Fatalf("scheduled_end = %s, want %s", consultation.ScheduledEnd, scheduledEnd)
	}
	if consultation.CheckInTime != nil {
		t.Fatalf("check_in_time = %v, want nil", consultation.CheckInTime)
	}
	if consultation.ReceptionNotes != nil {
		t.Fatalf("reception_notes = %v, want nil", consultation.ReceptionNotes)
	}
}

func TestCreateConsultationRequiresStandaloneScheduleRangeWhenSlotMissing(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, nil))

	_, err := repo.CreateConsultation(context.Background(), CreateConsultationParams{
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440211",
		PatientID:      "550e8400-e29b-41d4-a716-446655440212",
		Source:         ConsultationSourceDoctor,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want %v", err, ErrValidation)
	}
}

func TestCreateConsultationAllowsNilSlotIDWithStandaloneScheduleRange(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
	scheduledStart := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, time.April, 16, 10, 20, 0, 0, time.UTC)

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newConsultationRow(
			"550e8400-e29b-41d4-a716-446655440210",
			nil,
			"550e8400-e29b-41d4-a716-446655440211",
			"550e8400-e29b-41d4-a716-446655440212",
			ConsultationStatusScheduled,
			ConsultationSourceDoctor,
			nil,
			scheduledStart,
			scheduledEnd,
			nil,
			nil,
			now,
			now,
			nil,
		),
	}}))

	consultation, err := repo.CreateConsultation(context.Background(), CreateConsultationParams{
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440211",
		PatientID:      "550e8400-e29b-41d4-a716-446655440212",
		Source:         ConsultationSourceDoctor,
		ScheduledStart: &scheduledStart,
		ScheduledEnd:   &scheduledEnd,
	})
	if err != nil {
		t.Fatalf("CreateConsultation error = %v", err)
	}
	if consultation.SlotID != nil {
		t.Fatalf("slot_id = %v, want nil", consultation.SlotID)
	}
	if consultation.Notes != nil {
		t.Fatalf("notes = %v, want nil", consultation.Notes)
	}
	if !consultation.ScheduledStart.Equal(scheduledStart) {
		t.Fatalf("scheduled_start = %s, want %s", consultation.ScheduledStart, scheduledStart)
	}
	if !consultation.ScheduledEnd.Equal(scheduledEnd) {
		t.Fatalf("scheduled_end = %s, want %s", consultation.ScheduledEnd, scheduledEnd)
	}
}

func TestCreateConsultationAllowsRequestedPatientSource(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
	scheduledStart := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, time.April, 16, 10, 1, 0, 0, time.UTC)
	notes := "Prefiero turno por la tarde"

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newConsultationRow(
			"550e8400-e29b-41d4-a716-446655440214",
			nil,
			"550e8400-e29b-41d4-a716-446655440211",
			"550e8400-e29b-41d4-a716-446655440212",
			ConsultationStatusRequested,
			ConsultationSourcePatient,
			&notes,
			scheduledStart,
			scheduledEnd,
			nil,
			nil,
			now,
			now,
			nil,
		),
	}}))

	consultation, err := repo.CreateConsultation(context.Background(), CreateConsultationParams{
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440211",
		PatientID:      "550e8400-e29b-41d4-a716-446655440212",
		Status:         ConsultationStatusRequested,
		Source:         ConsultationSourcePatient,
		ScheduledStart: &scheduledStart,
		ScheduledEnd:   &scheduledEnd,
		Notes:          &notes,
	})
	if err != nil {
		t.Fatalf("CreateConsultation error = %v", err)
	}
	if consultation.Status != ConsultationStatusRequested {
		t.Fatalf("status = %q, want %q", consultation.Status, ConsultationStatusRequested)
	}
	if consultation.Source != ConsultationSourcePatient {
		t.Fatalf("source = %q, want %q", consultation.Source, ConsultationSourcePatient)
	}
	if consultation.SlotID != nil {
		t.Fatalf("slot_id = %v, want nil", consultation.SlotID)
	}
}

func TestGetConsultationReturnsConsultation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC)
	slotID := "550e8400-e29b-41d4-a716-446655440221"
	scheduledStart := time.Date(2026, time.April, 16, 11, 0, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, time.April, 16, 11, 30, 0, 0, time.UTC)
	checkInTime := time.Date(2026, time.April, 16, 10, 45, 0, 0, time.UTC)
	receptionNotes := "Paciente ya está en sala"

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newConsultationRow(
			"550e8400-e29b-41d4-a716-446655440220",
			&slotID,
			"550e8400-e29b-41d4-a716-446655440222",
			"550e8400-e29b-41d4-a716-446655440223",
			ConsultationStatusCheckedIn,
			ConsultationSourceOnline,
			nil,
			scheduledStart,
			scheduledEnd,
			&checkInTime,
			&receptionNotes,
			now,
			now,
			nil,
		),
	}}))

	consultation, err := repo.GetConsultation(context.Background(), "550e8400-e29b-41d4-a716-446655440220")
	if err != nil {
		t.Fatalf("GetConsultation error = %v", err)
	}
	if consultation.CheckInTime == nil || !consultation.CheckInTime.Equal(checkInTime) {
		t.Fatalf("check_in_time = %v, want %s", consultation.CheckInTime, checkInTime)
	}
	if !consultation.ScheduledStart.Equal(scheduledStart) {
		t.Fatalf("scheduled_start = %s, want %s", consultation.ScheduledStart, scheduledStart)
	}
	if !consultation.ScheduledEnd.Equal(scheduledEnd) {
		t.Fatalf("scheduled_end = %s, want %s", consultation.ScheduledEnd, scheduledEnd)
	}
	if consultation.ReceptionNotes == nil || *consultation.ReceptionNotes != receptionNotes {
		t.Fatalf("reception_notes = %v, want %q", consultation.ReceptionNotes, receptionNotes)
	}
}

func TestGetConsultationReturnsNotFoundWhenMissing(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{err: sql.ErrNoRows}}))

	_, err := repo.GetConsultation(context.Background(), "550e8400-e29b-41d4-a716-446655440230")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrNotFound)
	}
}

func TestUpdateConsultationStatusReturnsUpdatedConsultation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 16, 12, 0, 0, 0, time.UTC)
	slotID := "550e8400-e29b-41d4-a716-446655440241"
	scheduledStart := time.Date(2026, time.April, 16, 12, 0, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, time.April, 16, 12, 30, 0, 0, time.UTC)
	checkInTime := time.Date(2026, time.April, 16, 11, 55, 0, 0, time.UTC)
	receptionNotes := "Ingresó con 5 minutos de demora"

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newConsultationRow(
			"550e8400-e29b-41d4-a716-446655440240",
			&slotID,
			"550e8400-e29b-41d4-a716-446655440242",
			"550e8400-e29b-41d4-a716-446655440243",
			ConsultationStatusCheckedIn,
			ConsultationSourceSecretary,
			nil,
			scheduledStart,
			scheduledEnd,
			&checkInTime,
			&receptionNotes,
			now,
			now,
			nil,
		),
	}}))

	consultation, err := repo.UpdateConsultationStatus(context.Background(), "550e8400-e29b-41d4-a716-446655440240", UpdateConsultationStatusParams{
		Status:         ConsultationStatusCheckedIn,
		CheckInTime:    &checkInTime,
		ReceptionNotes: &receptionNotes,
	})
	if err != nil {
		t.Fatalf("UpdateConsultationStatus error = %v", err)
	}
	if consultation.Status != ConsultationStatusCheckedIn {
		t.Fatalf("status = %q, want %q", consultation.Status, ConsultationStatusCheckedIn)
	}
	if !consultation.ScheduledStart.Equal(scheduledStart) {
		t.Fatalf("scheduled_start = %s, want %s", consultation.ScheduledStart, scheduledStart)
	}
	if !consultation.ScheduledEnd.Equal(scheduledEnd) {
		t.Fatalf("scheduled_end = %s, want %s", consultation.ScheduledEnd, scheduledEnd)
	}
	if consultation.CheckInTime == nil || !consultation.CheckInTime.Equal(checkInTime) {
		t.Fatalf("check_in_time = %v, want %s", consultation.CheckInTime, checkInTime)
	}
	if consultation.ReceptionNotes == nil || *consultation.ReceptionNotes != receptionNotes {
		t.Fatalf("reception_notes = %v, want %q", consultation.ReceptionNotes, receptionNotes)
	}
}

func TestUpdateConsultationStatusClearsOptionalMetadataWhenOmitted(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 16, 13, 0, 0, 0, time.UTC)
	scheduledStart := time.Date(2026, time.April, 16, 13, 0, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, time.April, 16, 13, 30, 0, 0, time.UTC)

	repo := NewRepository(newScriptedDB(t, []scriptedQueryResult{{
		row: newConsultationRow(
			"550e8400-e29b-41d4-a716-446655440250",
			nil,
			"550e8400-e29b-41d4-a716-446655440251",
			"550e8400-e29b-41d4-a716-446655440252",
			ConsultationStatusCompleted,
			ConsultationSourceDoctor,
			nil,
			scheduledStart,
			scheduledEnd,
			nil,
			nil,
			now,
			now,
			nil,
		),
	}}))

	consultation, err := repo.UpdateConsultationStatus(context.Background(), "550e8400-e29b-41d4-a716-446655440250", UpdateConsultationStatusParams{
		Status: ConsultationStatusCompleted,
	})
	if err != nil {
		t.Fatalf("UpdateConsultationStatus error = %v", err)
	}
	if consultation.CheckInTime != nil {
		t.Fatalf("check_in_time = %v, want nil", consultation.CheckInTime)
	}
	if !consultation.ScheduledStart.Equal(scheduledStart) {
		t.Fatalf("scheduled_start = %s, want %s", consultation.ScheduledStart, scheduledStart)
	}
	if !consultation.ScheduledEnd.Equal(scheduledEnd) {
		t.Fatalf("scheduled_end = %s, want %s", consultation.ScheduledEnd, scheduledEnd)
	}
	if consultation.ReceptionNotes != nil {
		t.Fatalf("reception_notes = %v, want nil", consultation.ReceptionNotes)
	}
}

func TestUpdateConsultationStatusReturnsValidationOnBadStatus(t *testing.T) {
	t.Parallel()

	repo := NewRepository(newScriptedDB(t, nil))

	_, err := repo.UpdateConsultationStatus(context.Background(), "550e8400-e29b-41d4-a716-446655440260", UpdateConsultationStatusParams{
		Status: ConsultationStatus("booked"),
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want %v", err, ErrValidation)
	}
}

func newConsultationRow(id string, slotID *string, professionalID, patientID string, status ConsultationStatus, source ConsultationSource, notes *string, scheduledStart, scheduledEnd time.Time, checkInTime *time.Time, receptionNotes *string, createdAt, updatedAt time.Time, cancelledAt *time.Time) []driver.Value {
	return []driver.Value{
		id,
		nullableStringValue(slotID),
		professionalID,
		patientID,
		string(status),
		string(source),
		nullableStringValue(notes),
		scheduledStart,
		scheduledEnd,
		nullableTimeValue(checkInTime),
		nullableStringValue(receptionNotes),
		createdAt,
		updatedAt,
		nullableTimeValue(cancelledAt),
	}
}

func nullableTimeValue(value *time.Time) any {
	if value == nil {
		return nil
	}

	return *value
}
