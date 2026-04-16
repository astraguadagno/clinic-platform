package appointments

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

func TestRepositoryIntegrationScheduleBlockLifecyclePersistsScopedBlocks(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440140"
	createdTemplate, err := repo.CreateTemplate(ctx, CreateTemplateParams{
		ProfessionalID: professionalID,
		EffectiveFrom:  "2026-05-01",
		Recurrence:     json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}}`),
	})
	if err != nil {
		t.Fatalf("create template for schedule block: %v", err)
	}

	created, err := repo.CreateScheduleBlock(ctx, CreateScheduleBlockParams{
		ProfessionalID: professionalID,
		Scope:          "single",
		BlockDate:      stringPtr("2026-05-05"),
		StartTime:      "09:00",
		EndTime:        "12:00",
	})
	if err != nil {
		t.Fatalf("create schedule block: %v", err)
	}
	if created.BlockDate == nil || created.BlockDate.Format("2006-01-02") != "2026-05-05" {
		t.Fatalf("created block_date = %v, want 2026-05-05", created.BlockDate)
	}

	updated, err := repo.UpdateScheduleBlock(ctx, created.ID, UpdateScheduleBlockParams{
		ProfessionalID: professionalID,
		Scope:          "template",
		DayOfWeek:      intPtr(1),
		StartTime:      "10:00",
		EndTime:        "11:00",
		TemplateID:     &createdTemplate.ID,
	})
	if err != nil {
		t.Fatalf("update schedule block: %v", err)
	}
	if updated.Scope != "template" {
		t.Fatalf("updated scope = %q, want template", updated.Scope)
	}
	if updated.TemplateID == nil || *updated.TemplateID != createdTemplate.ID {
		t.Fatalf("updated template_id = %v, want %q", updated.TemplateID, createdTemplate.ID)
	}
	if updated.DayOfWeek == nil || *updated.DayOfWeek != 1 {
		t.Fatalf("updated day_of_week = %v, want 1", updated.DayOfWeek)
	}

	persisted, err := repo.GetScheduleBlock(ctx, created.ID)
	if err != nil {
		t.Fatalf("get schedule block: %v", err)
	}
	if persisted.Scope != "template" {
		t.Fatalf("persisted scope = %q, want template", persisted.Scope)
	}
	if persisted.StartTime != "10:00" || persisted.EndTime != "11:00" {
		t.Fatalf("persisted time range = %s-%s, want 10:00-11:00", persisted.StartTime, persisted.EndTime)
	}

	blocks, err := repo.ListScheduleBlocks(ctx, ScheduleBlockFilters{ProfessionalID: professionalID})
	if err != nil {
		t.Fatalf("list schedule blocks: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("listed blocks len = %d, want 1", len(blocks))
	}
	if blocks[0].ID != created.ID {
		t.Fatalf("listed block id = %q, want %q", blocks[0].ID, created.ID)
	}

	if got := countRows(t, db, "schedule_blocks"); got != 1 {
		t.Fatalf("schedule blocks persisted = %d, want 1", got)
	}

	if err := repo.DeleteScheduleBlock(ctx, created.ID); err != nil {
		t.Fatalf("delete schedule block: %v", err)
	}
	if got := countRows(t, db, "schedule_blocks"); got != 0 {
		t.Fatalf("schedule blocks persisted after delete = %d, want 0", got)
	}

	_, err = repo.GetScheduleBlock(ctx, created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("post-delete get err = %v, want %v", err, ErrNotFound)
	}
}

func TestRepositoryIntegrationCreateTemplatePersistsTemplateAndVersions(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440100"
	createdBy := "550e8400-e29b-41d4-a716-446655440101"
	firstReason := "initial rollout"
	secondReason := "winter hours"

	firstRecurrence := json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}}`)
	created, err := repo.CreateTemplate(ctx, CreateTemplateParams{
		ProfessionalID: professionalID,
		EffectiveFrom:  "2026-05-01",
		Recurrence:     firstRecurrence,
		CreatedBy:      &createdBy,
		Reason:         &firstReason,
	})
	if err != nil {
		t.Fatalf("create initial template: %v", err)
	}
	if len(created.Versions) != 1 {
		t.Fatalf("created versions len = %d, want 1", len(created.Versions))
	}
	if created.Versions[0].VersionNumber != 1 {
		t.Fatalf("initial version number = %d, want 1", created.Versions[0].VersionNumber)
	}

	secondRecurrence := json.RawMessage(`{"monday":{"start_time":"08:00","end_time":"12:00","slot_duration_minutes":30},"wednesday":{"start_time":"10:00","end_time":"13:00","slot_duration_minutes":30}}`)
	updated, err := repo.CreateTemplate(ctx, CreateTemplateParams{
		ProfessionalID: professionalID,
		EffectiveFrom:  "2026-06-01",
		Recurrence:     secondRecurrence,
		Reason:         &secondReason,
	})
	if err != nil {
		t.Fatalf("create second template version: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("updated template id = %q, want %q", updated.ID, created.ID)
	}
	if updated.Versions[0].VersionNumber != 2 {
		t.Fatalf("second version number = %d, want 2", updated.Versions[0].VersionNumber)
	}

	persistedTemplate, err := repo.GetTemplate(ctx, created.ID)
	if err != nil {
		t.Fatalf("get template: %v", err)
	}
	if persistedTemplate.ProfessionalID != professionalID {
		t.Fatalf("persisted professional_id = %q, want %q", persistedTemplate.ProfessionalID, professionalID)
	}

	versions, err := repo.ListTemplateVersions(ctx, created.ID)
	if err != nil {
		t.Fatalf("list template versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("persisted versions len = %d, want 2", len(versions))
	}
	if versions[0].VersionNumber != 2 || versions[1].VersionNumber != 1 {
		t.Fatalf("persisted version order = [%d %d], want [2 1]", versions[0].VersionNumber, versions[1].VersionNumber)
	}
	if versions[1].CreatedBy == nil || *versions[1].CreatedBy != createdBy {
		t.Fatalf("persisted created_by = %v, want %q", versions[1].CreatedBy, createdBy)
	}
	if versions[0].Reason == nil || *versions[0].Reason != secondReason {
		t.Fatalf("persisted reason = %v, want %q", versions[0].Reason, secondReason)
	}
	if got := countRows(t, db, "schedule_templates"); got != 1 {
		t.Fatalf("templates persisted = %d, want 1", got)
	}
	if got := countRows(t, db, "schedule_template_versions"); got != 2 {
		t.Fatalf("template versions persisted = %d, want 2", got)
	}
}

func TestRepositoryIntegrationGetActiveTemplateSelectsLatestApplicableVersion(t *testing.T) {
	repo, _ := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440102"

	first, err := repo.CreateTemplate(ctx, CreateTemplateParams{
		ProfessionalID: professionalID,
		EffectiveFrom:  "2026-05-01",
		Recurrence:     json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30}}`),
	})
	if err != nil {
		t.Fatalf("create first template version: %v", err)
	}

	second, err := repo.CreateTemplate(ctx, CreateTemplateParams{
		ProfessionalID: professionalID,
		EffectiveFrom:  "2026-06-01",
		Recurrence:     json.RawMessage(`{"monday":{"start_time":"08:00","end_time":"12:00","slot_duration_minutes":30}}`),
	})
	if err != nil {
		t.Fatalf("create second template version: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("template family id = %q, want %q", second.ID, first.ID)
	}

	mayVersion, err := repo.GetActiveTemplate(ctx, professionalID, "2026-05-31")
	if err != nil {
		t.Fatalf("get active template for may: %v", err)
	}
	if mayVersion.VersionNumber != 1 {
		t.Fatalf("may version number = %d, want 1", mayVersion.VersionNumber)
	}

	juneVersion, err := repo.GetActiveTemplate(ctx, professionalID, "2026-06-01")
	if err != nil {
		t.Fatalf("get active template for june boundary: %v", err)
	}
	if juneVersion.VersionNumber != 2 {
		t.Fatalf("june version number = %d, want 2", juneVersion.VersionNumber)
	}

	_, err = repo.GetActiveTemplate(ctx, professionalID, "2026-04-30")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("pre-history err = %v, want %v", err, ErrNotFound)
	}
}

func TestRepositoryIntegrationCreateSlotsBulkRejectsOverlapInPostgres(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440000"

	initialSlots, err := repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-10",
		StartTime:           "09:00",
		EndTime:             "10:00",
		SlotDurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("seed slots: %v", err)
	}
	if len(initialSlots) != 2 {
		t.Fatalf("initial slots len = %d, want 2", len(initialSlots))
	}

	_, err = repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-10",
		StartTime:           "09:15",
		EndTime:             "09:45",
		SlotDurationMinutes: 30,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v, want %v", err, ErrConflict)
	}

	if got := countSlots(t, db); got != 2 {
		t.Fatalf("slots persisted = %d, want 2", got)
	}
}

func TestRepositoryIntegrationAppointmentLifecyclePersistsAppointmentAndSlotState(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440010"
	patientID := "550e8400-e29b-41d4-a716-446655440011"

	slots, err := repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-11",
		StartTime:           "11:00",
		EndTime:             "11:30",
		SlotDurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create slots: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("slots len = %d, want 1", len(slots))
	}

	created, err := repo.CreateAppointment(ctx, CreateAppointmentParams{
		SlotID:         slots[0].ID,
		PatientID:      patientID,
		ProfessionalID: professionalID,
	})
	if err != nil {
		t.Fatalf("create appointment: %v", err)
	}
	if created.Status != "booked" {
		t.Fatalf("created status = %q, want booked", created.Status)
	}

	persistedCreatedAppointment := fetchAppointmentByID(t, db, created.ID)
	if persistedCreatedAppointment.Status != "booked" {
		t.Fatalf("persisted appointment status = %q, want booked", persistedCreatedAppointment.Status)
	}
	if persistedCreatedAppointment.CancelledAt != nil {
		t.Fatalf("persisted appointment cancelled_at = %v, want nil", persistedCreatedAppointment.CancelledAt)
	}

	persistedBookedSlot := fetchSlotByID(t, db, slots[0].ID)
	if persistedBookedSlot.Status != "booked" {
		t.Fatalf("slot status after booking = %q, want booked", persistedBookedSlot.Status)
	}

	cancelled, err := repo.CancelAppointment(ctx, created.ID)
	if err != nil {
		t.Fatalf("cancel appointment: %v", err)
	}
	if cancelled.Status != "cancelled" {
		t.Fatalf("cancelled status = %q, want cancelled", cancelled.Status)
	}
	if cancelled.CancelledAt == nil {
		t.Fatal("cancelled appointment missing cancelled_at")
	}

	persistedCancelledAppointment := fetchAppointmentByID(t, db, created.ID)
	if persistedCancelledAppointment.Status != "cancelled" {
		t.Fatalf("persisted cancelled status = %q, want cancelled", persistedCancelledAppointment.Status)
	}
	if persistedCancelledAppointment.CancelledAt == nil {
		t.Fatal("persisted cancelled appointment missing cancelled_at")
	}

	persistedAvailableSlot := fetchSlotByID(t, db, slots[0].ID)
	if persistedAvailableSlot.Status != "available" {
		t.Fatalf("slot status after cancellation = %q, want available", persistedAvailableSlot.Status)
	}
	if !persistedAvailableSlot.UpdatedAt.Equal(persistedCancelledAppointment.UpdatedAt) {
		t.Fatalf("slot updated_at = %s, want %s", persistedAvailableSlot.UpdatedAt, persistedCancelledAppointment.UpdatedAt)
	}

	rebookedPatientID := "550e8400-e29b-41d4-a716-446655440012"
	rebooked, err := repo.CreateAppointment(ctx, CreateAppointmentParams{
		SlotID:         slots[0].ID,
		PatientID:      rebookedPatientID,
		ProfessionalID: professionalID,
	})
	if err != nil {
		t.Fatalf("rebook appointment: %v", err)
	}
	if rebooked.Status != "booked" {
		t.Fatalf("rebooked status = %q, want booked", rebooked.Status)
	}
	if rebooked.ID == created.ID {
		t.Fatal("rebooked appointment reused cancelled appointment id")
	}

	persistedRebookedAppointment := fetchAppointmentByID(t, db, rebooked.ID)
	if persistedRebookedAppointment.Status != "booked" {
		t.Fatalf("persisted rebooked appointment status = %q, want booked", persistedRebookedAppointment.Status)
	}
	if persistedRebookedAppointment.PatientID != rebookedPatientID {
		t.Fatalf("persisted rebooked patient_id = %q, want %q", persistedRebookedAppointment.PatientID, rebookedPatientID)
	}

	persistedRebookedSlot := fetchSlotByID(t, db, slots[0].ID)
	if persistedRebookedSlot.Status != "booked" {
		t.Fatalf("slot status after rebooking = %q, want booked", persistedRebookedSlot.Status)
	}

	persistedAppointments, err := repo.ListAppointments(ctx, AppointmentFilters{ProfessionalID: professionalID})
	if err != nil {
		t.Fatalf("list appointments after rebooking: %v", err)
	}
	if len(persistedAppointments) != 2 {
		t.Fatalf("appointments listed after rebooking = %d, want 2", len(persistedAppointments))
	}

	activeAppointments := 0
	for _, appointment := range persistedAppointments {
		if appointment.SlotID != slots[0].ID {
			t.Fatalf("appointment slot_id = %q, want %q", appointment.SlotID, slots[0].ID)
		}
		if appointment.Status == "booked" {
			activeAppointments++
		}
	}
	if activeAppointments != 1 {
		t.Fatalf("active appointments for slot = %d, want 1", activeAppointments)
	}
}

func TestRepositoryIntegrationCreateAppointmentRejectsDoubleBookingForSameSlot(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440020"
	firstPatientID := "550e8400-e29b-41d4-a716-446655440021"
	secondPatientID := "550e8400-e29b-41d4-a716-446655440022"

	slots, err := repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-12",
		StartTime:           "14:00",
		EndTime:             "14:30",
		SlotDurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create slots: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("slots len = %d, want 1", len(slots))
	}

	created, err := repo.CreateAppointment(ctx, CreateAppointmentParams{
		SlotID:         slots[0].ID,
		PatientID:      firstPatientID,
		ProfessionalID: professionalID,
	})
	if err != nil {
		t.Fatalf("first booking: %v", err)
	}

	_, err = repo.CreateAppointment(ctx, CreateAppointmentParams{
		SlotID:         slots[0].ID,
		PatientID:      secondPatientID,
		ProfessionalID: professionalID,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("second booking err = %v, want %v", err, ErrConflict)
	}

	if got := countAppointments(t, db); got != 1 {
		t.Fatalf("appointments persisted = %d, want 1", got)
	}

	persistedAppointment := fetchAppointmentByID(t, db, created.ID)
	if persistedAppointment.PatientID != firstPatientID {
		t.Fatalf("persisted patient_id = %q, want %q", persistedAppointment.PatientID, firstPatientID)
	}
	if persistedAppointment.Status != "booked" {
		t.Fatalf("persisted appointment status = %q, want booked", persistedAppointment.Status)
	}

	persistedSlot := fetchSlotByID(t, db, slots[0].ID)
	if persistedSlot.Status != "booked" {
		t.Fatalf("slot status after double booking attempt = %q, want booked", persistedSlot.Status)
	}
	if !persistedSlot.UpdatedAt.Equal(persistedAppointment.UpdatedAt) {
		t.Fatalf("slot updated_at = %s, want %s", persistedSlot.UpdatedAt, persistedAppointment.UpdatedAt)
	}
}

func TestRepositoryIntegrationCreateAppointmentConcurrentSameSlotReturnsOneSuccessAndOneConflict(t *testing.T) {
	repo, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440030"
	firstPatientID := "550e8400-e29b-41d4-a716-446655440031"
	secondPatientID := "550e8400-e29b-41d4-a716-446655440032"

	slots, err := repo.CreateSlotsBulk(ctx, BulkCreateSlotsParams{
		ProfessionalID:      professionalID,
		Date:                "2026-04-13",
		StartTime:           "15:00",
		EndTime:             "15:30",
		SlotDurationMinutes: 30,
	})
	if err != nil {
		t.Fatalf("create slots: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("slots len = %d, want 1", len(slots))
	}

	type bookingResult struct {
		appointment Appointment
		err         error
	}

	start := make(chan struct{})
	results := make(chan bookingResult, 2)

	var ready sync.WaitGroup
	ready.Add(2)

	book := func(patientID string) {
		ready.Done()
		<-start

		appointment, err := repo.CreateAppointment(context.Background(), CreateAppointmentParams{
			SlotID:         slots[0].ID,
			PatientID:      patientID,
			ProfessionalID: professionalID,
		})

		results <- bookingResult{appointment: appointment, err: err}
	}

	go book(firstPatientID)
	go book(secondPatientID)

	ready.Wait()
	close(start)

	firstResult := <-results
	secondResult := <-results

	var (
		successes []Appointment
		conflicts int
	)

	for _, result := range []bookingResult{firstResult, secondResult} {
		switch {
		case result.err == nil:
			successes = append(successes, result.appointment)
		case errors.Is(result.err, ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected create appointment error: %v", result.err)
		}
	}

	if len(successes) != 1 {
		t.Fatalf("successful bookings = %d, want 1", len(successes))
	}
	if conflicts != 1 {
		t.Fatalf("conflicting bookings = %d, want 1", conflicts)
	}

	persistedAppointments, err := repo.ListAppointments(ctx, AppointmentFilters{ProfessionalID: professionalID})
	if err != nil {
		t.Fatalf("list appointments: %v", err)
	}
	if len(persistedAppointments) != 1 {
		t.Fatalf("appointments listed = %d, want 1", len(persistedAppointments))
	}
	if got := countAppointments(t, db); got != 1 {
		t.Fatalf("appointments persisted = %d, want 1", got)
	}

	persistedAppointment := persistedAppointments[0]
	if persistedAppointment.Status != "booked" {
		t.Fatalf("persisted appointment status = %q, want booked", persistedAppointment.Status)
	}
	if persistedAppointment.SlotID != slots[0].ID {
		t.Fatalf("persisted appointment slot_id = %q, want %q", persistedAppointment.SlotID, slots[0].ID)
	}
	if persistedAppointment.PatientID != firstPatientID && persistedAppointment.PatientID != secondPatientID {
		t.Fatalf("persisted appointment patient_id = %q, want one of concurrent patients", persistedAppointment.PatientID)
	}

	persistedSlot := fetchSlotByID(t, db, slots[0].ID)
	if persistedSlot.Status != "booked" {
		t.Fatalf("slot status after concurrent booking = %q, want booked", persistedSlot.Status)
	}
	if successes[0].ID != persistedAppointment.ID {
		t.Fatalf("successful appointment id = %q, want %q", successes[0].ID, persistedAppointment.ID)
	}
}
