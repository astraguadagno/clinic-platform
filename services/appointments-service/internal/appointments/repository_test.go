package appointments

import (
	"errors"
	"testing"
	"time"
)

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
