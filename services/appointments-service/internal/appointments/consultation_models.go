package appointments

import "time"

type ConsultationStatus string

const (
	ConsultationStatusScheduled ConsultationStatus = "scheduled"
	ConsultationStatusCheckedIn ConsultationStatus = "checked_in"
	ConsultationStatusCompleted ConsultationStatus = "completed"
	ConsultationStatusCancelled ConsultationStatus = "cancelled"
	ConsultationStatusNoShow    ConsultationStatus = "no_show"
)

func (status ConsultationStatus) IsValid() bool {
	switch status {
	case ConsultationStatusScheduled,
		ConsultationStatusCheckedIn,
		ConsultationStatusCompleted,
		ConsultationStatusCancelled,
		ConsultationStatusNoShow:
		return true
	default:
		return false
	}
}

type ConsultationSource string

type ConsultationActorRole string

const (
	ConsultationActorRoleSecretary ConsultationActorRole = "secretary"
	ConsultationActorRoleDoctor    ConsultationActorRole = "doctor"
)

func (role ConsultationActorRole) IsValid() bool {
	switch role {
	case ConsultationActorRoleSecretary,
		ConsultationActorRoleDoctor:
		return true
	default:
		return false
	}
}

const (
	ConsultationSourceOnline    ConsultationSource = "online"
	ConsultationSourceSecretary ConsultationSource = "secretary"
	ConsultationSourceDoctor    ConsultationSource = "doctor"
)

func (source ConsultationSource) IsValid() bool {
	switch source {
	case ConsultationSourceOnline,
		ConsultationSourceSecretary,
		ConsultationSourceDoctor:
		return true
	default:
		return false
	}
}

type Consultation struct {
	ID             string             `json:"id"`
	SlotID         *string            `json:"slot_id,omitempty"`
	ProfessionalID string             `json:"professional_id"`
	PatientID      string             `json:"patient_id"`
	Status         ConsultationStatus `json:"status"`
	Source         ConsultationSource `json:"source"`
	ScheduledStart time.Time          `json:"scheduled_start"`
	ScheduledEnd   time.Time          `json:"scheduled_end"`
	Notes          *string            `json:"notes,omitempty"`
	CheckInTime    *time.Time         `json:"check_in_time,omitempty"`
	ReceptionNotes *string            `json:"reception_notes,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	CancelledAt    *time.Time         `json:"cancelled_at,omitempty"`
}
