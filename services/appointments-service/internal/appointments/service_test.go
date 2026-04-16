package appointments

import (
	"context"
	"errors"
	"testing"
	"time"
)

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
