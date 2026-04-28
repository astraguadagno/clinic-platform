package appointments

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestConsultationModelMatchesSchemaContract(t *testing.T) {
	t.Parallel()

	consultationType := reflect.TypeOf(Consultation{})

	assertField(t, consultationType, "ID", reflect.TypeOf(""), "id")
	assertField(t, consultationType, "ProfessionalID", reflect.TypeOf(""), "professional_id")
	assertField(t, consultationType, "PatientID", reflect.TypeOf(""), "patient_id")
	assertField(t, consultationType, "Status", reflect.TypeOf(ConsultationStatus("")), "status")
	assertField(t, consultationType, "Source", reflect.TypeOf(ConsultationSource("")), "source")
	assertField(t, consultationType, "SlotID", reflect.TypeOf((*string)(nil)), "slot_id,omitempty")
	assertField(t, consultationType, "ScheduledStart", reflect.TypeOf(time.Time{}), "scheduled_start")
	assertField(t, consultationType, "ScheduledEnd", reflect.TypeOf(time.Time{}), "scheduled_end")
	assertField(t, consultationType, "Notes", reflect.TypeOf((*string)(nil)), "notes,omitempty")
	assertField(t, consultationType, "CheckInTime", reflect.TypeOf((*time.Time)(nil)), "check_in_time,omitempty")
	assertField(t, consultationType, "ReceptionNotes", reflect.TypeOf((*string)(nil)), "reception_notes,omitempty")
	assertField(t, consultationType, "CreatedAt", reflect.TypeOf(time.Time{}), "created_at")
	assertField(t, consultationType, "UpdatedAt", reflect.TypeOf(time.Time{}), "updated_at")
	assertField(t, consultationType, "CancelledAt", reflect.TypeOf((*time.Time)(nil)), "cancelled_at,omitempty")
}

func TestConsultationJSONOptionalMetadata(t *testing.T) {
	t.Parallel()

	checkInTime := time.Date(2026, time.April, 16, 9, 45, 0, 0, time.UTC)
	scheduledStart := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
	scheduledEnd := time.Date(2026, time.April, 16, 10, 30, 0, 0, time.UTC)
	slotID := "slot-123"
	receptionNotes := "Paciente llegó temprano"

	tests := []struct {
		name         string
		consultation Consultation
		wantPresent  []string
		wantAbsent   []string
	}{
		{
			name:         "nil metadata omitted",
			consultation: Consultation{ScheduledStart: scheduledStart, ScheduledEnd: scheduledEnd},
			wantAbsent:   []string{"slot_id", "check_in_time", "reception_notes"},
			wantPresent:  []string{"scheduled_start", "scheduled_end"},
		},
		{
			name: "optional metadata included when set",
			consultation: Consultation{
				SlotID:         &slotID,
				ScheduledStart: scheduledStart,
				ScheduledEnd:   scheduledEnd,
				CheckInTime:    &checkInTime,
				ReceptionNotes: &receptionNotes,
			},
			wantPresent: []string{"slot_id", "scheduled_start", "scheduled_end", "check_in_time", "reception_notes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := marshalJSONMap(t, tt.consultation)

			for _, key := range tt.wantPresent {
				if _, ok := payload[key]; !ok {
					t.Fatalf("expected key %q to be present", key)
				}
			}

			for _, key := range tt.wantAbsent {
				if _, ok := payload[key]; ok {
					t.Fatalf("expected key %q to be omitted", key)
				}
			}
		})
	}
}

func TestConsultationStatusIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status ConsultationStatus
		want   bool
	}{
		{name: "scheduled is valid", status: ConsultationStatusScheduled, want: true},
		{name: "requested is valid", status: ConsultationStatusRequested, want: true},
		{name: "checked_in is valid", status: ConsultationStatusCheckedIn, want: true},
		{name: "completed is valid", status: ConsultationStatusCompleted, want: true},
		{name: "cancelled is valid", status: ConsultationStatusCancelled, want: true},
		{name: "no_show is valid", status: ConsultationStatusNoShow, want: true},
		{name: "legacy booked is invalid", status: ConsultationStatus("booked"), want: false},
		{name: "empty is invalid", status: ConsultationStatus(""), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.status.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConsultationSourceIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source ConsultationSource
		want   bool
	}{
		{name: "online is valid", source: ConsultationSourceOnline, want: true},
		{name: "secretary is valid", source: ConsultationSourceSecretary, want: true},
		{name: "doctor is valid", source: ConsultationSourceDoctor, want: true},
		{name: "patient is valid", source: ConsultationSourcePatient, want: true},
		{name: "empty is invalid", source: ConsultationSource(""), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.source.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func marshalJSONMap(t *testing.T, v any) map[string]any {
	t.Helper()

	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return payload
}
