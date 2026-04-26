package appointments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestGenerateSlotsForWeekUsesEffectiveTemplateVersionForEachDay(t *testing.T) {
	t.Parallel()

	template := ScheduleTemplate{
		ID:             "550e8400-e29b-41d4-a716-446655440010",
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440000",
		Versions: []ScheduleTemplateVersion{
			{
				ID:            "550e8400-e29b-41d4-a716-446655440021",
				TemplateID:    "550e8400-e29b-41d4-a716-446655440010",
				VersionNumber: 2,
				EffectiveFrom: time.Date(2026, time.April, 8, 0, 0, 0, 0, time.UTC),
				Recurrence:    json.RawMessage(`{"wednesday":{"start_time":"10:00","end_time":"11:00","slot_duration_minutes":30}}`),
			},
			{
				ID:            "550e8400-e29b-41d4-a716-446655440020",
				TemplateID:    "550e8400-e29b-41d4-a716-446655440010",
				VersionNumber: 1,
				EffectiveFrom: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
				Recurrence:    json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}}`),
			},
		},
	}

	slots, err := GenerateSlotsForWeek(template, nil, time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GenerateSlotsForWeek error = %v", err)
	}
	if len(slots) != 4 {
		t.Fatalf("slots len = %d, want 4", len(slots))
	}

	wantStarts := []time.Time{
		time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 6, 9, 30, 0, 0, time.UTC),
		time.Date(2026, time.April, 8, 10, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 8, 10, 30, 0, 0, time.UTC),
	}
	for i, wantStart := range wantStarts {
		if !slots[i].StartTime.Equal(wantStart) {
			t.Fatalf("slots[%d].start_time = %s, want %s", i, slots[i].StartTime, wantStart)
		}
		if slots[i].ProfessionalID != template.ProfessionalID {
			t.Fatalf("slots[%d].professional_id = %q, want %q", i, slots[i].ProfessionalID, template.ProfessionalID)
		}
		if slots[i].Status != "available" {
			t.Fatalf("slots[%d].status = %q, want available", i, slots[i].Status)
		}
	}
}

func TestGenerateSlotsForWeekFiltersSingleRangeAndTemplateBlocks(t *testing.T) {
	t.Parallel()

	templateID := "550e8400-e29b-41d4-a716-446655440010"
	template := ScheduleTemplate{
		ID:             templateID,
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440000",
		Versions: []ScheduleTemplateVersion{{
			ID:            "550e8400-e29b-41d4-a716-446655440020",
			TemplateID:    templateID,
			VersionNumber: 1,
			EffectiveFrom: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
			Recurrence: json.RawMessage(`{
				"monday":{"start_time":"09:00","end_time":"12:00","slot_duration_minutes":30},
				"wednesday":{"start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}
			}`),
		}},
	}

	monday := time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC)
	wednesday := time.Date(2026, time.April, 8, 0, 0, 0, 0, time.UTC)
	startDate := wednesday
	endDate := time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC)
	matchingDay := 1
	otherDay := 1
	otherTemplateID := "550e8400-e29b-41d4-a716-446655440099"

	slots, err := GenerateSlotsForWeek(template, []ScheduleBlock{
		{
			ProfessionalID: template.ProfessionalID,
			Scope:          "single",
			BlockDate:      &monday,
			StartTime:      "10:00",
			EndTime:        "11:00",
		},
		{
			ProfessionalID: template.ProfessionalID,
			Scope:          "range",
			StartDate:      &startDate,
			EndDate:        &endDate,
			StartTime:      "09:00",
			EndTime:        "09:30",
		},
		{
			ProfessionalID: template.ProfessionalID,
			Scope:          "template",
			DayOfWeek:      &matchingDay,
			StartTime:      "11:00",
			EndTime:        "11:30",
			TemplateID:     &templateID,
		},
		{
			ProfessionalID: template.ProfessionalID,
			Scope:          "template",
			DayOfWeek:      &otherDay,
			StartTime:      "09:00",
			EndTime:        "09:30",
			TemplateID:     &otherTemplateID,
		},
		{
			ProfessionalID: "550e8400-e29b-41d4-a716-446655440111",
			Scope:          "single",
			BlockDate:      &monday,
			StartTime:      "09:00",
			EndTime:        "09:30",
		},
	}, monday)
	if err != nil {
		t.Fatalf("GenerateSlotsForWeek error = %v", err)
	}
	if len(slots) != 4 {
		t.Fatalf("slots len = %d, want 4", len(slots))
	}

	wantStarts := []time.Time{
		time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 6, 9, 30, 0, 0, time.UTC),
		time.Date(2026, time.April, 6, 11, 30, 0, 0, time.UTC),
		time.Date(2026, time.April, 8, 9, 30, 0, 0, time.UTC),
	}
	for i, wantStart := range wantStarts {
		if !slots[i].StartTime.Equal(wantStart) {
			t.Fatalf("slots[%d].start_time = %s, want %s", i, slots[i].StartTime, wantStart)
		}
	}
}

func TestComposeWeekAgendaAggregatesTemplatesBlocksConsultationsAndSlots(t *testing.T) {
	t.Parallel()

	professionalID := "550e8400-e29b-41d4-a716-446655440000"
	otherProfessionalID := "550e8400-e29b-41d4-a716-446655440999"
	weekStart := time.Date(2026, time.April, 6, 14, 30, 0, 0, time.UTC)
	templateID := "550e8400-e29b-41d4-a716-446655440010"
	monday := time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC)
	wednesday := time.Date(2026, time.April, 8, 0, 0, 0, 0, time.UTC)
	otherTemplateID := "550e8400-e29b-41d4-a716-446655440011"

	agenda, err := ComposeWeekAgenda(professionalID, weekStart, []ScheduleTemplate{
		{
			ID:             templateID,
			ProfessionalID: professionalID,
			Versions: []ScheduleTemplateVersion{
				{
					ID:            "550e8400-e29b-41d4-a716-446655440020",
					TemplateID:    templateID,
					VersionNumber: 1,
					EffectiveFrom: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
					Recurrence:    json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}}`),
				},
				{
					ID:            "550e8400-e29b-41d4-a716-446655440021",
					TemplateID:    templateID,
					VersionNumber: 2,
					EffectiveFrom: time.Date(2026, time.April, 8, 0, 0, 0, 0, time.UTC),
					Recurrence:    json.RawMessage(`{"wednesday":{"start_time":"10:00","end_time":"11:00","slot_duration_minutes":30}}`),
				},
				{
					ID:            "550e8400-e29b-41d4-a716-446655440022",
					TemplateID:    templateID,
					VersionNumber: 3,
					EffectiveFrom: time.Date(2026, time.April, 20, 0, 0, 0, 0, time.UTC),
					Recurrence:    json.RawMessage(`{"monday":{"start_time":"12:00","end_time":"13:00","slot_duration_minutes":30}}`),
				},
			},
		},
		{
			ID:             otherTemplateID,
			ProfessionalID: otherProfessionalID,
			Versions: []ScheduleTemplateVersion{{
				ID:            "550e8400-e29b-41d4-a716-446655440023",
				TemplateID:    otherTemplateID,
				VersionNumber: 1,
				EffectiveFrom: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
				Recurrence:    json.RawMessage(`{"monday":{"start_time":"15:00","end_time":"16:00","slot_duration_minutes":30}}`),
			}},
		},
	}, []ScheduleBlock{
		{
			ID:             "550e8400-e29b-41d4-a716-446655440030",
			ProfessionalID: professionalID,
			Scope:          "single",
			BlockDate:      &monday,
			StartTime:      "09:30",
			EndTime:        "10:00",
		},
		{
			ID:             "550e8400-e29b-41d4-a716-446655440031",
			ProfessionalID: professionalID,
			Scope:          "template",
			DayOfWeek:      intPtr(3),
			StartTime:      "10:30",
			EndTime:        "11:00",
			TemplateID:     &templateID,
		},
		{
			ID:             "550e8400-e29b-41d4-a716-446655440032",
			ProfessionalID: otherProfessionalID,
			Scope:          "single",
			BlockDate:      &wednesday,
			StartTime:      "10:00",
			EndTime:        "10:30",
		},
	}, []Consultation{
		{
			ID:             "550e8400-e29b-41d4-a716-446655440040",
			ProfessionalID: professionalID,
			PatientID:      "550e8400-e29b-41d4-a716-446655440041",
			Status:         ConsultationStatusScheduled,
			Source:         ConsultationSourceSecretary,
			ScheduledStart: time.Date(2026, time.April, 8, 9, 15, 0, 0, time.UTC),
			ScheduledEnd:   time.Date(2026, time.April, 8, 9, 30, 0, 0, time.UTC),
			CreatedAt:      time.Date(2026, time.April, 6, 8, 0, 0, 0, time.UTC),
		},
		{
			ID:             "550e8400-e29b-41d4-a716-446655440042",
			ProfessionalID: otherProfessionalID,
			PatientID:      "550e8400-e29b-41d4-a716-446655440043",
			Status:         ConsultationStatusScheduled,
			Source:         ConsultationSourceDoctor,
			ScheduledStart: time.Date(2026, time.April, 8, 10, 0, 0, 0, time.UTC),
			ScheduledEnd:   time.Date(2026, time.April, 8, 10, 30, 0, 0, time.UTC),
			CreatedAt:      time.Date(2026, time.April, 6, 7, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("ComposeWeekAgenda error = %v", err)
	}
	if agenda.ProfessionalID != professionalID {
		t.Fatalf("professional_id = %q, want %q", agenda.ProfessionalID, professionalID)
	}
	if agenda.WeekStart != "2026-04-06" {
		t.Fatalf("week_start = %q, want %q", agenda.WeekStart, "2026-04-06")
	}
	if len(agenda.Templates) != 1 {
		t.Fatalf("templates len = %d, want 1", len(agenda.Templates))
	}
	if len(agenda.Templates[0].Versions) != 2 {
		t.Fatalf("template versions len = %d, want 2", len(agenda.Templates[0].Versions))
	}
	if agenda.Templates[0].Versions[0].VersionNumber != 1 || agenda.Templates[0].Versions[1].VersionNumber != 2 {
		t.Fatalf("template version numbers = [%d %d], want [1 2]", agenda.Templates[0].Versions[0].VersionNumber, agenda.Templates[0].Versions[1].VersionNumber)
	}
	if len(agenda.Blocks) != 2 {
		t.Fatalf("blocks len = %d, want 2", len(agenda.Blocks))
	}
	if len(agenda.Consultations) != 1 {
		t.Fatalf("consultations len = %d, want 1", len(agenda.Consultations))
	}
	if len(agenda.Slots) != 2 {
		t.Fatalf("slots len = %d, want 2", len(agenda.Slots))
	}

	wantSlotStarts := []time.Time{
		time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, time.April, 8, 10, 0, 0, 0, time.UTC),
	}
	for i, wantStart := range wantSlotStarts {
		if !agenda.Slots[i].StartTime.Equal(wantStart) {
			t.Fatalf("slots[%d].start_time = %s, want %s", i, agenda.Slots[i].StartTime, wantStart)
		}
	}
}

func TestComposeWeekAgendaKeepsOnlyConsultationsIntersectingWeek(t *testing.T) {
	t.Parallel()

	professionalID := "550e8400-e29b-41d4-a716-446655440099"
	weekStart := time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC)

	agenda, err := ComposeWeekAgenda(professionalID, weekStart, nil, nil, []Consultation{
		{
			ID:             "550e8400-e29b-41d4-a716-446655440100",
			ProfessionalID: professionalID,
			PatientID:      "550e8400-e29b-41d4-a716-446655440101",
			Status:         ConsultationStatusScheduled,
			Source:         ConsultationSourceDoctor,
			ScheduledStart: time.Date(2026, time.April, 8, 15, 0, 0, 0, time.UTC),
			ScheduledEnd:   time.Date(2026, time.April, 8, 15, 20, 0, 0, time.UTC),
		},
		{
			ID:             "550e8400-e29b-41d4-a716-446655440102",
			ProfessionalID: professionalID,
			PatientID:      "550e8400-e29b-41d4-a716-446655440103",
			Status:         ConsultationStatusScheduled,
			Source:         ConsultationSourceSecretary,
			ScheduledStart: time.Date(2026, time.April, 13, 9, 0, 0, 0, time.UTC),
			ScheduledEnd:   time.Date(2026, time.April, 13, 9, 20, 0, 0, time.UTC),
		},
		{
			ID:             "550e8400-e29b-41d4-a716-446655440104",
			ProfessionalID: professionalID,
			PatientID:      "550e8400-e29b-41d4-a716-446655440105",
			Status:         ConsultationStatusScheduled,
			Source:         ConsultationSourceSecretary,
			ScheduledStart: time.Date(2026, time.April, 5, 23, 50, 0, 0, time.UTC),
			ScheduledEnd:   time.Date(2026, time.April, 6, 0, 10, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("ComposeWeekAgenda error = %v", err)
	}
	if len(agenda.Consultations) != 2 {
		t.Fatalf("consultations len = %d, want 2", len(agenda.Consultations))
	}
	if agenda.Consultations[0].ID != "550e8400-e29b-41d4-a716-446655440104" || agenda.Consultations[1].ID != "550e8400-e29b-41d4-a716-446655440100" {
		t.Fatalf("consultation ids = [%s %s], want [550e8400-e29b-41d4-a716-446655440104 550e8400-e29b-41d4-a716-446655440100]", agenda.Consultations[0].ID, agenda.Consultations[1].ID)
	}
}

func TestComposeWeekAgendaKeepsOnlyWeekRelevantTemplatesAndBlocks(t *testing.T) {
	t.Parallel()

	professionalID := "550e8400-e29b-41d4-a716-446655440000"
	weekStart := time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC)
	templateID := "550e8400-e29b-41d4-a716-446655440010"
	beforeWeek := time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC)
	afterWeek := time.Date(2026, time.April, 13, 0, 0, 0, 0, time.UTC)

	agenda, err := ComposeWeekAgenda(professionalID, weekStart, []ScheduleTemplate{{
		ID:             templateID,
		ProfessionalID: professionalID,
		Versions: []ScheduleTemplateVersion{{
			ID:            "550e8400-e29b-41d4-a716-446655440020",
			TemplateID:    templateID,
			VersionNumber: 1,
			EffectiveFrom: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
			Recurrence:    json.RawMessage(`{"monday":{"start_time":"08:00","end_time":"09:00","slot_duration_minutes":30}}`),
		}, {
			ID:            "550e8400-e29b-41d4-a716-446655440021",
			TemplateID:    templateID,
			VersionNumber: 2,
			EffectiveFrom: time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC),
			Recurrence:    json.RawMessage(`{"monday":{"start_time":"09:00","end_time":"10:00","slot_duration_minutes":30}}`),
		}, {
			ID:            "550e8400-e29b-41d4-a716-446655440022",
			TemplateID:    templateID,
			VersionNumber: 3,
			EffectiveFrom: time.Date(2026, time.April, 20, 0, 0, 0, 0, time.UTC),
			Recurrence:    json.RawMessage(`{"monday":{"start_time":"11:00","end_time":"12:00","slot_duration_minutes":30}}`),
		}},
	}}, []ScheduleBlock{
		{
			ID:             "550e8400-e29b-41d4-a716-446655440030",
			ProfessionalID: professionalID,
			Scope:          "single",
			BlockDate:      &beforeWeek,
			StartTime:      "09:00",
			EndTime:        "09:30",
		},
		{
			ID:             "550e8400-e29b-41d4-a716-446655440031",
			ProfessionalID: professionalID,
			Scope:          "single",
			BlockDate:      &afterWeek,
			StartTime:      "09:00",
			EndTime:        "09:30",
		},
	}, nil)
	if err != nil {
		t.Fatalf("ComposeWeekAgenda error = %v", err)
	}
	if len(agenda.Templates) != 1 {
		t.Fatalf("templates len = %d, want 1", len(agenda.Templates))
	}
	if len(agenda.Templates[0].Versions) != 1 {
		t.Fatalf("template versions len = %d, want 1", len(agenda.Templates[0].Versions))
	}
	if agenda.Templates[0].Versions[0].VersionNumber != 2 {
		t.Fatalf("template version number = %d, want 2", agenda.Templates[0].Versions[0].VersionNumber)
	}
	if len(agenda.Blocks) != 0 {
		t.Fatalf("blocks len = %d, want 0", len(agenda.Blocks))
	}
	if len(agenda.Slots) != 2 {
		t.Fatalf("slots len = %d, want 2", len(agenda.Slots))
	}
}

func TestScheduleServiceGetScheduleReturnsActiveTemplateForDate(t *testing.T) {
	t.Parallel()

	activeVersion := ScheduleTemplateVersion{
		ID:            "550e8400-e29b-41d4-a716-446655440021",
		TemplateID:    "550e8400-e29b-41d4-a716-446655440010",
		VersionNumber: 2,
		EffectiveFrom: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
	}
	baseTemplate := ScheduleTemplate{
		ID:             activeVersion.TemplateID,
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440000",
		CreatedAt:      time.Date(2026, time.April, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, time.April, 20, 12, 0, 0, 0, time.UTC),
		Versions: []ScheduleTemplateVersion{{
			ID:            "550e8400-e29b-41d4-a716-446655440020",
			TemplateID:    activeVersion.TemplateID,
			VersionNumber: 1,
		}},
	}

	repo := &stubScheduleRepository{
		getActiveTemplateFn: func(_ context.Context, professionalID string, effectiveDate string) (ScheduleTemplateVersion, error) {
			if professionalID != "550e8400-e29b-41d4-a716-446655440000" {
				t.Fatalf("professionalID = %q, want expected professional", professionalID)
			}
			if effectiveDate != "2026-06-15" {
				t.Fatalf("effectiveDate = %q, want %q", effectiveDate, "2026-06-15")
			}
			return activeVersion, nil
		},
		getTemplateFn: func(_ context.Context, templateID string) (ScheduleTemplate, error) {
			if templateID != activeVersion.TemplateID {
				t.Fatalf("templateID = %q, want active version template id", templateID)
			}
			return baseTemplate, nil
		},
	}

	service := NewScheduleService(repo)

	template, err := service.GetSchedule(context.Background(), "550e8400-e29b-41d4-a716-446655440000", "2026-06-15")
	if err != nil {
		t.Fatalf("GetSchedule error = %v", err)
	}
	if repo.getActiveTemplateCalls != 1 {
		t.Fatalf("GetActiveTemplate calls = %d, want 1", repo.getActiveTemplateCalls)
	}
	if repo.getTemplateCalls != 1 {
		t.Fatalf("GetTemplate calls = %d, want 1", repo.getTemplateCalls)
	}
	if template.ID != activeVersion.TemplateID {
		t.Fatalf("template id = %q, want %q", template.ID, activeVersion.TemplateID)
	}
	if len(template.Versions) != 1 {
		t.Fatalf("versions len = %d, want 1", len(template.Versions))
	}
	if template.Versions[0].VersionNumber != activeVersion.VersionNumber {
		t.Fatalf("version number = %d, want %d", template.Versions[0].VersionNumber, activeVersion.VersionNumber)
	}
}

func TestScheduleServiceGetSchedulePropagatesLookupErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "validation", repoErr: ErrValidation},
		{name: "not found", repoErr: ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &stubScheduleRepository{
				getActiveTemplateFn: func(context.Context, string, string) (ScheduleTemplateVersion, error) {
					return ScheduleTemplateVersion{}, tt.repoErr
				},
			}

			service := NewScheduleService(repo)

			_, err := service.GetSchedule(context.Background(), "550e8400-e29b-41d4-a716-446655440000", "2026-06-15")
			if !errors.Is(err, tt.repoErr) {
				t.Fatalf("err = %v, want %v", err, tt.repoErr)
			}
			if repo.getTemplateCalls != 0 {
				t.Fatalf("GetTemplate calls = %d, want 0", repo.getTemplateCalls)
			}
		})
	}
}

func TestScheduleServiceGetSchedulePropagatesTemplateLoadError(t *testing.T) {
	t.Parallel()

	repo := &stubScheduleRepository{
		getActiveTemplateFn: func(context.Context, string, string) (ScheduleTemplateVersion, error) {
			return ScheduleTemplateVersion{
				TemplateID: "550e8400-e29b-41d4-a716-446655440010",
			}, nil
		},
		getTemplateFn: func(context.Context, string) (ScheduleTemplate, error) {
			return ScheduleTemplate{}, ErrNotFound
		},
	}

	service := NewScheduleService(repo)

	_, err := service.GetSchedule(context.Background(), "550e8400-e29b-41d4-a716-446655440000", "2026-06-15")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrNotFound)
	}
	if repo.getTemplateCalls != 1 {
		t.Fatalf("GetTemplate calls = %d, want 1", repo.getTemplateCalls)
	}
}

func TestScheduleServiceListTemplateVersionsReturnsTemplateHistory(t *testing.T) {
	t.Parallel()

	templateID := "550e8400-e29b-41d4-a716-446655440010"
	template := ScheduleTemplate{
		ID:             templateID,
		ProfessionalID: "550e8400-e29b-41d4-a716-446655440000",
	}
	versions := []ScheduleTemplateVersion{{
		ID:            "550e8400-e29b-41d4-a716-446655440021",
		TemplateID:    templateID,
		VersionNumber: 2,
		EffectiveFrom: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
	}, {
		ID:            "550e8400-e29b-41d4-a716-446655440020",
		TemplateID:    templateID,
		VersionNumber: 1,
		EffectiveFrom: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
	}}

	repo := &stubScheduleRepository{
		getTemplateFn: func(_ context.Context, gotTemplateID string) (ScheduleTemplate, error) {
			if gotTemplateID != templateID {
				t.Fatalf("templateID = %q, want %q", gotTemplateID, templateID)
			}
			return template, nil
		},
		listTemplateVersionsFn: func(_ context.Context, gotTemplateID string) ([]ScheduleTemplateVersion, error) {
			if gotTemplateID != templateID {
				t.Fatalf("templateID = %q, want %q", gotTemplateID, templateID)
			}
			return versions, nil
		},
	}

	service := NewScheduleService(repo)

	got, err := service.ListTemplateVersions(context.Background(), templateID)
	if err != nil {
		t.Fatalf("ListTemplateVersions error = %v", err)
	}
	if repo.getTemplateCalls != 1 {
		t.Fatalf("GetTemplate calls = %d, want 1", repo.getTemplateCalls)
	}
	if repo.listTemplateVersionsCalls != 1 {
		t.Fatalf("ListTemplateVersions calls = %d, want 1", repo.listTemplateVersionsCalls)
	}
	if got.ID != template.ID {
		t.Fatalf("template id = %q, want %q", got.ID, template.ID)
	}
	if len(got.Versions) != len(versions) {
		t.Fatalf("versions len = %d, want %d", len(got.Versions), len(versions))
	}
	if got.Versions[0].VersionNumber != 2 || got.Versions[1].VersionNumber != 1 {
		t.Fatalf("version order = [%d %d], want [2 1]", got.Versions[0].VersionNumber, got.Versions[1].VersionNumber)
	}
}

func TestScheduleServiceListTemplateVersionsPropagatesErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		getTemplateErr          error
		listTemplateVersionsErr error
		wantGetTemplateCalls    int
		wantListTemplateCalls   int
	}{
		{
			name:                  "template lookup validation error",
			getTemplateErr:        ErrValidation,
			wantGetTemplateCalls:  1,
			wantListTemplateCalls: 0,
		},
		{
			name:                    "list versions error",
			listTemplateVersionsErr: ErrNotFound,
			wantGetTemplateCalls:    1,
			wantListTemplateCalls:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &stubScheduleRepository{
				getTemplateFn: func(context.Context, string) (ScheduleTemplate, error) {
					if tt.getTemplateErr != nil {
						return ScheduleTemplate{}, tt.getTemplateErr
					}
					return ScheduleTemplate{ID: "550e8400-e29b-41d4-a716-446655440010"}, nil
				},
				listTemplateVersionsFn: func(context.Context, string) ([]ScheduleTemplateVersion, error) {
					return nil, tt.listTemplateVersionsErr
				},
			}

			service := NewScheduleService(repo)

			_, err := service.ListTemplateVersions(context.Background(), "550e8400-e29b-41d4-a716-446655440010")
			wantErr := tt.getTemplateErr
			if wantErr == nil {
				wantErr = tt.listTemplateVersionsErr
			}
			if !errors.Is(err, wantErr) {
				t.Fatalf("err = %v, want %v", err, wantErr)
			}
			if repo.getTemplateCalls != tt.wantGetTemplateCalls {
				t.Fatalf("GetTemplate calls = %d, want %d", repo.getTemplateCalls, tt.wantGetTemplateCalls)
			}
			if repo.listTemplateVersionsCalls != tt.wantListTemplateCalls {
				t.Fatalf("ListTemplateVersions calls = %d, want %d", repo.listTemplateVersionsCalls, tt.wantListTemplateCalls)
			}
		})
	}
}

type stubScheduleRepository struct {
	getActiveTemplateFn       func(ctx context.Context, professionalID string, effectiveDate string) (ScheduleTemplateVersion, error)
	getTemplateFn             func(ctx context.Context, templateID string) (ScheduleTemplate, error)
	listTemplateVersionsFn    func(ctx context.Context, templateID string) ([]ScheduleTemplateVersion, error)
	getActiveTemplateCalls    int
	getTemplateCalls          int
	listTemplateVersionsCalls int
}

func (s *stubScheduleRepository) GetActiveTemplate(ctx context.Context, professionalID string, effectiveDate string) (ScheduleTemplateVersion, error) {
	s.getActiveTemplateCalls++
	if s.getActiveTemplateFn == nil {
		return ScheduleTemplateVersion{}, nil
	}
	return s.getActiveTemplateFn(ctx, professionalID, effectiveDate)
}

func (s *stubScheduleRepository) GetTemplate(ctx context.Context, templateID string) (ScheduleTemplate, error) {
	s.getTemplateCalls++
	if s.getTemplateFn == nil {
		return ScheduleTemplate{}, nil
	}
	return s.getTemplateFn(ctx, templateID)
}

func (s *stubScheduleRepository) ListTemplateVersions(ctx context.Context, templateID string) ([]ScheduleTemplateVersion, error) {
	s.listTemplateVersionsCalls++
	if s.listTemplateVersionsFn == nil {
		return nil, nil
	}
	return s.listTemplateVersionsFn(ctx, templateID)
}

func TestConsultationServiceUpdateStatusAllowsValidTransition(t *testing.T) {
	t.Parallel()

	checkInTime := time.Date(2026, time.April, 16, 8, 55, 0, 0, time.UTC)
	receptionNotes := "Paciente ya presente"
	tests := []struct {
		name          string
		currentStatus ConsultationStatus
		update        ConsultationStatusUpdateParams
		wantStatus    ConsultationStatus
	}{
		{
			name:          "scheduled to checked in",
			currentStatus: ConsultationStatusScheduled,
			update: ConsultationStatusUpdateParams{
				Status:         ConsultationStatusCheckedIn,
				CheckInTime:    &checkInTime,
				ReceptionNotes: &receptionNotes,
			},
			wantStatus: ConsultationStatusCheckedIn,
		},
		{
			name:          "checked in to completed",
			currentStatus: ConsultationStatusCheckedIn,
			update: ConsultationStatusUpdateParams{
				Status:    ConsultationStatusCompleted,
				ActorRole: ConsultationActorRoleDoctor,
			},
			wantStatus: ConsultationStatusCompleted,
		},
		{
			name:          "same source is accepted",
			currentStatus: ConsultationStatusScheduled,
			update: ConsultationStatusUpdateParams{
				Status: ConsultationStatusCancelled,
				Source: sourcePtr(ConsultationSourceSecretary),
			},
			wantStatus: ConsultationStatusCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			consultationID := "550e8400-e29b-41d4-a716-446655440300"
			repo := &stubConsultationRepository{
				getConsultationFn: func(_ context.Context, gotConsultationID string) (Consultation, error) {
					if gotConsultationID != consultationID {
						t.Fatalf("consultationID = %q, want %q", gotConsultationID, consultationID)
					}
					return Consultation{
						ID:     consultationID,
						Status: tt.currentStatus,
						Source: ConsultationSourceSecretary,
					}, nil
				},
				updateConsultationStatusFn: func(_ context.Context, gotConsultationID string, params UpdateConsultationStatusParams) (Consultation, error) {
					if gotConsultationID != consultationID {
						t.Fatalf("consultationID = %q, want %q", gotConsultationID, consultationID)
					}
					if params.Status != tt.update.Status {
						t.Fatalf("status = %q, want %q", params.Status, tt.update.Status)
					}
					if !timesEqual(params.CheckInTime, tt.update.CheckInTime) {
						t.Fatalf("check_in_time = %v, want %v", params.CheckInTime, tt.update.CheckInTime)
					}
					if !stringsEqual(params.ReceptionNotes, tt.update.ReceptionNotes) {
						t.Fatalf("reception_notes = %v, want %v", params.ReceptionNotes, tt.update.ReceptionNotes)
					}
					return Consultation{
						ID:             consultationID,
						Status:         tt.wantStatus,
						Source:         ConsultationSourceSecretary,
						CheckInTime:    params.CheckInTime,
						ReceptionNotes: params.ReceptionNotes,
					}, nil
				},
			}

			service := NewConsultationService(repo)

			consultation, err := service.UpdateStatus(context.Background(), consultationID, tt.update)
			if err != nil {
				t.Fatalf("UpdateStatus error = %v", err)
			}
			if repo.getConsultationCalls != 1 {
				t.Fatalf("GetConsultation calls = %d, want 1", repo.getConsultationCalls)
			}
			if repo.updateConsultationStatusCalls != 1 {
				t.Fatalf("UpdateConsultationStatus calls = %d, want 1", repo.updateConsultationStatusCalls)
			}
			if consultation.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", consultation.Status, tt.wantStatus)
			}
			if consultation.Source != ConsultationSourceSecretary {
				t.Fatalf("source = %q, want %q", consultation.Source, ConsultationSourceSecretary)
			}
		})
	}
}

func TestConsultationServiceUpdateStatusRejectsInvalidTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		currentStatus ConsultationStatus
		nextStatus    ConsultationStatus
	}{
		{name: "cannot move completed back to scheduled", currentStatus: ConsultationStatusCompleted, nextStatus: ConsultationStatusScheduled},
		{name: "cannot move completed to cancelled", currentStatus: ConsultationStatusCompleted, nextStatus: ConsultationStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &stubConsultationRepository{
				getConsultationFn: func(context.Context, string) (Consultation, error) {
					return Consultation{
						ID:     "550e8400-e29b-41d4-a716-446655440301",
						Status: tt.currentStatus,
						Source: ConsultationSourceDoctor,
					}, nil
				},
			}

			service := NewConsultationService(repo)

			_, err := service.UpdateStatus(context.Background(), "550e8400-e29b-41d4-a716-446655440301", ConsultationStatusUpdateParams{
				Status: tt.nextStatus,
			})
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want %v", err, ErrValidation)
			}
			if repo.updateConsultationStatusCalls != 0 {
				t.Fatalf("UpdateConsultationStatus calls = %d, want 0", repo.updateConsultationStatusCalls)
			}
		})
	}
}

func TestConsultationServiceUpdateStatusRejectsSecretaryCompletingConsultation(t *testing.T) {
	t.Parallel()

	repo := &stubConsultationRepository{
		getConsultationFn: func(context.Context, string) (Consultation, error) {
			return Consultation{
				ID:     "550e8400-e29b-41d4-a716-446655440311",
				Status: ConsultationStatusCheckedIn,
				Source: ConsultationSourceSecretary,
			}, nil
		},
	}

	service := NewConsultationService(repo)

	_, err := service.UpdateStatus(context.Background(), "550e8400-e29b-41d4-a716-446655440311", ConsultationStatusUpdateParams{
		Status:    ConsultationStatusCompleted,
		ActorRole: ConsultationActorRoleSecretary,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want %v", err, ErrValidation)
	}
	if repo.updateConsultationStatusCalls != 0 {
		t.Fatalf("UpdateConsultationStatus calls = %d, want 0", repo.updateConsultationStatusCalls)
	}
}

func TestConsultationServiceUpdateStatusRejectsSourceChangeAfterCreate(t *testing.T) {
	t.Parallel()

	repo := &stubConsultationRepository{
		getConsultationFn: func(context.Context, string) (Consultation, error) {
			return Consultation{
				ID:     "550e8400-e29b-41d4-a716-446655440302",
				Status: ConsultationStatusScheduled,
				Source: ConsultationSourceSecretary,
			}, nil
		},
	}

	service := NewConsultationService(repo)
	updatedSource := ConsultationSourceOnline

	_, err := service.UpdateStatus(context.Background(), "550e8400-e29b-41d4-a716-446655440302", ConsultationStatusUpdateParams{
		Status:    ConsultationStatusCompleted,
		Source:    &updatedSource,
		ActorRole: ConsultationActorRoleDoctor,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want %v", err, ErrValidation)
	}
	if repo.updateConsultationStatusCalls != 0 {
		t.Fatalf("UpdateConsultationStatus calls = %d, want 0", repo.updateConsultationStatusCalls)
	}
}

type stubConsultationRepository struct {
	getConsultationFn             func(ctx context.Context, consultationID string) (Consultation, error)
	updateConsultationStatusFn    func(ctx context.Context, consultationID string, params UpdateConsultationStatusParams) (Consultation, error)
	getConsultationCalls          int
	updateConsultationStatusCalls int
}

func (s *stubConsultationRepository) GetConsultation(ctx context.Context, consultationID string) (Consultation, error) {
	s.getConsultationCalls++
	if s.getConsultationFn == nil {
		return Consultation{}, nil
	}
	return s.getConsultationFn(ctx, consultationID)
}

func (s *stubConsultationRepository) UpdateConsultationStatus(ctx context.Context, consultationID string, params UpdateConsultationStatusParams) (Consultation, error) {
	s.updateConsultationStatusCalls++
	if s.updateConsultationStatusFn == nil {
		return Consultation{}, nil
	}
	return s.updateConsultationStatusFn(ctx, consultationID, params)
}

func sourcePtr(source ConsultationSource) *ConsultationSource {
	return &source
}

func timesEqual(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return left.Equal(*right)
}

func stringsEqual(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}
