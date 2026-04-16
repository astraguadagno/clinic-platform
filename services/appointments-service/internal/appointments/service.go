package appointments

import "context"

type scheduleRepository interface {
	GetActiveTemplate(ctx context.Context, professionalID string, effectiveDate string) (ScheduleTemplateVersion, error)
	GetTemplate(ctx context.Context, templateID string) (ScheduleTemplate, error)
	ListTemplateVersions(ctx context.Context, templateID string) ([]ScheduleTemplateVersion, error)
}

type ScheduleService struct {
	repo scheduleRepository
}

func NewScheduleService(repo scheduleRepository) *ScheduleService {
	return &ScheduleService{repo: repo}
}

func (s *ScheduleService) GetSchedule(ctx context.Context, professionalID string, effectiveDate string) (ScheduleTemplate, error) {
	activeVersion, err := s.repo.GetActiveTemplate(ctx, professionalID, effectiveDate)
	if err != nil {
		return ScheduleTemplate{}, err
	}

	template, err := s.repo.GetTemplate(ctx, activeVersion.TemplateID)
	if err != nil {
		return ScheduleTemplate{}, err
	}

	template.Versions = []ScheduleTemplateVersion{activeVersion}

	return template, nil
}

func (s *ScheduleService) ListTemplateVersions(ctx context.Context, templateID string) (ScheduleTemplate, error) {
	template, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return ScheduleTemplate{}, err
	}

	versions, err := s.repo.ListTemplateVersions(ctx, templateID)
	if err != nil {
		return ScheduleTemplate{}, err
	}

	template.Versions = versions

	return template, nil
}
