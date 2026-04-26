package appointments

import (
	"encoding/json"
	"time"
)

type ScheduleTemplate struct {
	ID             string                    `json:"id"`
	ProfessionalID string                    `json:"professional_id"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
	Versions       []ScheduleTemplateVersion `json:"versions,omitempty"`
}

type ScheduleTemplateVersion struct {
	ID            string          `json:"id"`
	TemplateID    string          `json:"template_id"`
	VersionNumber int             `json:"version_number"`
	EffectiveFrom time.Time       `json:"effective_from"`
	Recurrence    json.RawMessage `json:"recurrence"`
	CreatedAt     time.Time       `json:"created_at"`
	CreatedBy     *string         `json:"created_by,omitempty"`
	Reason        *string         `json:"reason,omitempty"`
}

type ScheduleBlock struct {
	ID             string     `json:"id"`
	ProfessionalID string     `json:"professional_id"`
	Scope          string     `json:"scope"`
	BlockDate      *time.Time `json:"block_date,omitempty"`
	StartDate      *time.Time `json:"start_date,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	DayOfWeek      *int       `json:"day_of_week,omitempty"`
	StartTime      string     `json:"start_time"`
	EndTime        string     `json:"end_time"`
	TemplateID     *string    `json:"template_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type WeekAgenda struct {
	ProfessionalID string             `json:"professional_id"`
	WeekStart      string             `json:"week_start"`
	Templates      []ScheduleTemplate `json:"templates"`
	Blocks         []ScheduleBlock    `json:"blocks"`
	Consultations  []Consultation     `json:"consultations"`
	Slots          []AvailabilitySlot `json:"slots"`
}
