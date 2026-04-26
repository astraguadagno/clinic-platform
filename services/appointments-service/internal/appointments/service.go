package appointments

import (
	"context"
	"encoding/json"
	"sort"
	"time"
)

type scheduleRepository interface {
	GetActiveTemplate(ctx context.Context, professionalID string, effectiveDate string) (ScheduleTemplateVersion, error)
	GetTemplate(ctx context.Context, templateID string) (ScheduleTemplate, error)
	ListTemplateVersions(ctx context.Context, templateID string) ([]ScheduleTemplateVersion, error)
}

type ScheduleService struct {
	repo scheduleRepository
}

type consultationRepository interface {
	GetConsultation(ctx context.Context, consultationID string) (Consultation, error)
	UpdateConsultationStatus(ctx context.Context, consultationID string, params UpdateConsultationStatusParams) (Consultation, error)
}

type ConsultationStatusUpdateParams struct {
	Status         ConsultationStatus    `json:"status"`
	Source         *ConsultationSource   `json:"source,omitempty"`
	ActorRole      ConsultationActorRole `json:"-"`
	CheckInTime    *time.Time            `json:"check_in_time,omitempty"`
	ReceptionNotes *string               `json:"reception_notes,omitempty"`
}

type ConsultationService struct {
	repo consultationRepository
}

func NewScheduleService(repo scheduleRepository) *ScheduleService {
	return &ScheduleService{repo: repo}
}

func NewConsultationService(repo consultationRepository) *ConsultationService {
	return &ConsultationService{repo: repo}
}

type recurrenceWindow struct {
	StartTime           string `json:"start_time"`
	EndTime             string `json:"end_time"`
	SlotDurationMinutes int    `json:"slot_duration_minutes"`
}

func GenerateSlotsForWeek(template ScheduleTemplate, blocks []ScheduleBlock, weekStart time.Time) ([]AvailabilitySlot, error) {
	weekStart = startOfDay(weekStart)
	slots := make([]AvailabilitySlot, 0)

	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		day := weekStart.AddDate(0, 0, dayOffset)
		version, ok := activeTemplateVersionForDate(template.Versions, day)
		if !ok {
			continue
		}

		recurrence, err := parseTemplateRecurrence(version.Recurrence)
		if err != nil {
			return nil, err
		}

		window, ok := recurrence[weekdayKey(day.Weekday())]
		if !ok {
			continue
		}

		startAt, endAt, duration, err := recurrenceBounds(day, window)
		if err != nil {
			return nil, err
		}

		for current := startAt; current.Before(endAt); current = current.Add(duration) {
			next := current.Add(duration)
			blocked, err := isBlocked(blocks, template, day, current, next)
			if err != nil {
				return nil, err
			}
			if blocked {
				continue
			}

			slots = append(slots, AvailabilitySlot{
				ProfessionalID: template.ProfessionalID,
				StartTime:      current,
				EndTime:        next,
				Status:         "available",
			})
		}
	}

	return slots, nil
}

func ComposeWeekAgenda(professionalID string, weekStart time.Time, templates []ScheduleTemplate, blocks []ScheduleBlock, consultations []Consultation) (WeekAgenda, error) {
	weekStart = startOfDay(weekStart)
	agenda := WeekAgenda{
		ProfessionalID: professionalID,
		WeekStart:      weekStart.Format("2006-01-02"),
		Templates:      make([]ScheduleTemplate, 0),
		Blocks:         make([]ScheduleBlock, 0),
		Consultations:  filterWeekConsultations(consultations, professionalID, weekStart),
		Slots:          make([]AvailabilitySlot, 0),
	}

	relevantTemplateIDs := make(map[string]struct{})
	for _, template := range templates {
		if template.ProfessionalID != professionalID {
			continue
		}

		relevantVersions := activeTemplateVersionsForWeek(template.Versions, weekStart)
		if len(relevantVersions) == 0 {
			continue
		}

		template.Versions = relevantVersions
		relevantTemplateIDs[template.ID] = struct{}{}
		agenda.Templates = append(agenda.Templates, template)

		slots, err := GenerateSlotsForWeek(template, blocks, weekStart)
		if err != nil {
			return WeekAgenda{}, err
		}
		agenda.Slots = append(agenda.Slots, slots...)
	}

	agenda.Blocks = filterWeekBlocks(blocks, professionalID, relevantTemplateIDs, weekStart)
	sortWeekAgenda(&agenda)

	return agenda, nil
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

func (s *ConsultationService) UpdateStatus(ctx context.Context, consultationID string, params ConsultationStatusUpdateParams) (Consultation, error) {
	consultation, err := s.repo.GetConsultation(ctx, consultationID)
	if err != nil {
		return Consultation{}, err
	}

	if err := validateConsultationStatusTransition(consultation.Status, params.Status); err != nil {
		return Consultation{}, err
	}
	if err := validateConsultationStatusPermission(params.ActorRole, params.Status); err != nil {
		return Consultation{}, err
	}
	if err := validateConsultationSourceImmutable(consultation.Source, params.Source); err != nil {
		return Consultation{}, err
	}

	return s.repo.UpdateConsultationStatus(ctx, consultationID, UpdateConsultationStatusParams{
		Status:         params.Status,
		CheckInTime:    params.CheckInTime,
		ReceptionNotes: params.ReceptionNotes,
	})
}

func validateConsultationStatusTransition(current, next ConsultationStatus) error {
	if !next.IsValid() {
		return ErrValidation
	}
	if current == ConsultationStatusCompleted && next != ConsultationStatusCompleted {
		return ErrValidation
	}

	return nil
}

func validateConsultationStatusPermission(actorRole ConsultationActorRole, next ConsultationStatus) error {
	if next != ConsultationStatusCompleted {
		return nil
	}
	if !actorRole.IsValid() {
		return ErrValidation
	}
	if actorRole != ConsultationActorRoleDoctor {
		return ErrValidation
	}

	return nil
}

func validateConsultationSourceImmutable(current ConsultationSource, next *ConsultationSource) error {
	if next == nil {
		return nil
	}
	if !next.IsValid() {
		return ErrValidation
	}
	if current != *next {
		return ErrValidation
	}

	return nil
}

func activeTemplateVersionsForWeek(versions []ScheduleTemplateVersion, weekStart time.Time) []ScheduleTemplateVersion {
	selected := make(map[string]ScheduleTemplateVersion)
	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		version, ok := activeTemplateVersionForDate(versions, weekStart.AddDate(0, 0, dayOffset))
		if !ok {
			continue
		}
		selected[version.ID] = version
	}

	relevant := make([]ScheduleTemplateVersion, 0, len(selected))
	for _, version := range selected {
		relevant = append(relevant, version)
	}

	sort.Slice(relevant, func(i, j int) bool {
		left := startOfDay(relevant[i].EffectiveFrom)
		right := startOfDay(relevant[j].EffectiveFrom)
		if !left.Equal(right) {
			return left.Before(right)
		}
		return relevant[i].VersionNumber < relevant[j].VersionNumber
	})

	return relevant
}

func activeTemplateVersionForDate(versions []ScheduleTemplateVersion, date time.Time) (ScheduleTemplateVersion, bool) {
	date = startOfDay(date)
	var (
		selected ScheduleTemplateVersion
		found    bool
	)

	for _, version := range versions {
		effectiveFrom := startOfDay(version.EffectiveFrom)
		if effectiveFrom.After(date) {
			continue
		}
		if !found || effectiveFrom.After(startOfDay(selected.EffectiveFrom)) {
			selected = version
			found = true
		}
	}

	return selected, found
}

func filterWeekBlocks(blocks []ScheduleBlock, professionalID string, templateIDs map[string]struct{}, weekStart time.Time) []ScheduleBlock {
	filtered := make([]ScheduleBlock, 0)
	for _, block := range blocks {
		if block.ProfessionalID != professionalID {
			continue
		}
		if !blockIntersectsWeek(block, templateIDs, weekStart) {
			continue
		}
		filtered = append(filtered, block)
	}

	return filtered
}

func filterWeekConsultations(consultations []Consultation, professionalID string, weekStart time.Time) []Consultation {
	weekStart = startOfDay(weekStart)
	weekEnd := weekStart.AddDate(0, 0, 7)

	filtered := make([]Consultation, 0)
	for _, consultation := range consultations {
		if consultation.ProfessionalID != professionalID {
			continue
		}
		if consultation.ScheduledStart.IsZero() || consultation.ScheduledEnd.IsZero() {
			continue
		}
		if !consultation.ScheduledEnd.After(weekStart) || !consultation.ScheduledStart.Before(weekEnd) {
			continue
		}
		filtered = append(filtered, consultation)
	}

	return filtered
}

func blockIntersectsWeek(block ScheduleBlock, templateIDs map[string]struct{}, weekStart time.Time) bool {
	weekStart = startOfDay(weekStart)
	weekEnd := weekStart.AddDate(0, 0, 7)

	switch block.Scope {
	case "single":
		if block.BlockDate == nil {
			return false
		}
		blockDate := startOfDay(*block.BlockDate)
		return (blockDate.Equal(weekStart) || blockDate.After(weekStart)) && blockDate.Before(weekEnd)
	case "range":
		if block.StartDate == nil || block.EndDate == nil {
			return false
		}
		startDate := startOfDay(*block.StartDate)
		endDate := startOfDay(*block.EndDate)
		return !endDate.Before(weekStart) && startDate.Before(weekEnd)
	case "template":
		if block.TemplateID == nil || block.DayOfWeek == nil {
			return false
		}
		if *block.DayOfWeek < 1 || *block.DayOfWeek > 7 {
			return false
		}
		_, ok := templateIDs[*block.TemplateID]
		return ok
	default:
		return false
	}
}

func sortWeekAgenda(agenda *WeekAgenda) {
	sort.Slice(agenda.Templates, func(i, j int) bool {
		left := agenda.Templates[i]
		right := agenda.Templates[j]
		leftFrom := time.Time{}
		rightFrom := time.Time{}
		if len(left.Versions) > 0 {
			leftFrom = startOfDay(left.Versions[0].EffectiveFrom)
		}
		if len(right.Versions) > 0 {
			rightFrom = startOfDay(right.Versions[0].EffectiveFrom)
		}
		if !leftFrom.Equal(rightFrom) {
			return leftFrom.Before(rightFrom)
		}
		return left.ID < right.ID
	})

	sort.Slice(agenda.Blocks, func(i, j int) bool {
		left := blockSortDate(agenda.Blocks[i])
		right := blockSortDate(agenda.Blocks[j])
		if !left.Equal(right) {
			return left.Before(right)
		}
		if agenda.Blocks[i].StartTime != agenda.Blocks[j].StartTime {
			return agenda.Blocks[i].StartTime < agenda.Blocks[j].StartTime
		}
		return agenda.Blocks[i].ID < agenda.Blocks[j].ID
	})

	sort.Slice(agenda.Consultations, func(i, j int) bool {
		left := agenda.Consultations[i]
		right := agenda.Consultations[j]
		if !left.ScheduledStart.Equal(right.ScheduledStart) {
			return left.ScheduledStart.Before(right.ScheduledStart)
		}
		return left.ID < right.ID
	})

	sort.Slice(agenda.Slots, func(i, j int) bool {
		if !agenda.Slots[i].StartTime.Equal(agenda.Slots[j].StartTime) {
			return agenda.Slots[i].StartTime.Before(agenda.Slots[j].StartTime)
		}
		if !agenda.Slots[i].EndTime.Equal(agenda.Slots[j].EndTime) {
			return agenda.Slots[i].EndTime.Before(agenda.Slots[j].EndTime)
		}
		return agenda.Slots[i].ID < agenda.Slots[j].ID
	})
}

func blockSortDate(block ScheduleBlock) time.Time {
	if block.BlockDate != nil {
		return startOfDay(*block.BlockDate)
	}
	if block.StartDate != nil {
		return startOfDay(*block.StartDate)
	}
	return time.Time{}
}

func parseTemplateRecurrence(payload json.RawMessage) (map[string]recurrenceWindow, error) {
	var recurrence map[string]recurrenceWindow
	if err := json.Unmarshal(payload, &recurrence); err != nil {
		return nil, ErrValidation
	}
	if recurrence == nil {
		return nil, ErrValidation
	}

	return recurrence, nil
}

func recurrenceBounds(day time.Time, window recurrenceWindow) (time.Time, time.Time, time.Duration, error) {
	startAt, err := combineDateAndClock(day, window.StartTime)
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	endAt, err := combineDateAndClock(day, window.EndTime)
	if err != nil {
		return time.Time{}, time.Time{}, 0, err
	}
	if !endAt.After(startAt) || window.SlotDurationMinutes <= 0 {
		return time.Time{}, time.Time{}, 0, ErrValidation
	}

	duration := time.Duration(window.SlotDurationMinutes) * time.Minute
	if endAt.Sub(startAt)%duration != 0 {
		return time.Time{}, time.Time{}, 0, ErrValidation
	}

	return startAt, endAt, duration, nil
}

func isBlocked(blocks []ScheduleBlock, template ScheduleTemplate, day, slotStart, slotEnd time.Time) (bool, error) {
	for _, block := range blocks {
		applies, err := blockAppliesToSlot(block, template, day, slotStart, slotEnd)
		if err != nil {
			return false, err
		}
		if applies {
			return true, nil
		}
	}

	return false, nil
}

func blockAppliesToSlot(block ScheduleBlock, template ScheduleTemplate, day, slotStart, slotEnd time.Time) (bool, error) {
	if block.ProfessionalID != "" && block.ProfessionalID != template.ProfessionalID {
		return false, nil
	}

	if !blockMatchesDay(block, template.ID, day) {
		return false, nil
	}

	blockStart, err := combineDateAndClock(day, block.StartTime)
	if err != nil {
		return false, err
	}
	blockEnd, err := combineDateAndClock(day, block.EndTime)
	if err != nil {
		return false, err
	}
	if !blockEnd.After(blockStart) {
		return false, ErrValidation
	}

	return slotStart.Before(blockEnd) && blockStart.Before(slotEnd), nil
}

func blockMatchesDay(block ScheduleBlock, templateID string, day time.Time) bool {
	targetDay := startOfDay(day)

	switch block.Scope {
	case "single":
		return block.BlockDate != nil && startOfDay(*block.BlockDate).Equal(targetDay)
	case "range":
		if block.StartDate == nil || block.EndDate == nil {
			return false
		}
		startDate := startOfDay(*block.StartDate)
		endDate := startOfDay(*block.EndDate)
		return (targetDay.Equal(startDate) || targetDay.After(startDate)) && (targetDay.Equal(endDate) || targetDay.Before(endDate))
	case "template":
		if block.DayOfWeek == nil || block.TemplateID == nil || *block.TemplateID != templateID {
			return false
		}
		return isoWeekday(targetDay) == *block.DayOfWeek
	default:
		return false
	}
}

func combineDateAndClock(day time.Time, clock string) (time.Time, error) {
	parsed, err := time.Parse("15:04", clock)
	if err != nil {
		return time.Time{}, ErrValidation
	}

	day = startOfDay(day)
	return time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), 0, 0, day.Location()), nil
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func weekdayKey(weekday time.Weekday) string {
	switch weekday {
	case time.Monday:
		return "monday"
	case time.Tuesday:
		return "tuesday"
	case time.Wednesday:
		return "wednesday"
	case time.Thursday:
		return "thursday"
	case time.Friday:
		return "friday"
	case time.Saturday:
		return "saturday"
	default:
		return "sunday"
	}
}

func isoWeekday(day time.Time) int {
	if day.Weekday() == time.Sunday {
		return 7
	}
	return int(day.Weekday())
}
