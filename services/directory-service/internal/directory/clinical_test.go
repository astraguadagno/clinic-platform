package directory

import (
	"errors"
	"testing"
	"time"
)

func TestValidateCreateEncounterParams(t *testing.T) {
	now := time.Date(2026, 4, 7, 15, 0, 0, 0, time.UTC)
	patientID := "0f0f6c4d-7bbb-4d8e-94f9-f13fca1d16ca"
	professionalID := "f58d7e2f-c5fc-4884-b7bb-a3d14577a995"

	tests := []struct {
		name     string
		params   CreateEncounterParams
		want     CreateEncounterParams
		wantTime time.Time
		wantErr  error
	}{
		{
			name: "valid encounter trims spaces and preserves occurred at",
			params: CreateEncounterParams{
				PatientID:      " " + patientID + " ",
				ProfessionalID: " " + professionalID + " ",
				OccurredAt:     " 2026-04-07T14:30:00Z ",
				Note:           " Paciente estable ",
			},
			want: CreateEncounterParams{
				PatientID:      patientID,
				ProfessionalID: professionalID,
				OccurredAt:     "2026-04-07T14:30:00Z",
				Note:           "Paciente estable",
			},
			wantTime: time.Date(2026, 4, 7, 14, 30, 0, 0, time.UTC),
		},
		{
			name: "missing occurred at defaults to now",
			params: CreateEncounterParams{
				PatientID:      patientID,
				ProfessionalID: professionalID,
				Note:           "Paciente estable",
			},
			want: CreateEncounterParams{
				PatientID:      patientID,
				ProfessionalID: professionalID,
				Note:           "Paciente estable",
			},
			wantTime: now,
		},
		{
			name: "missing note returns validation error",
			params: CreateEncounterParams{
				PatientID:      patientID,
				ProfessionalID: professionalID,
				Note:           " ",
			},
			wantErr: ErrValidation,
		},
		{
			name: "invalid patient id returns not found",
			params: CreateEncounterParams{
				PatientID:      "bad-id",
				ProfessionalID: professionalID,
				Note:           "Paciente estable",
			},
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotTime, err := validateCreateEncounterParams(tt.params, now)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			if got != tt.want {
				t.Fatalf("params = %+v, want %+v", got, tt.want)
			}
			if !gotTime.Equal(tt.wantTime) {
				t.Fatalf("occurredAt = %v, want %v", gotTime, tt.wantTime)
			}
		})
	}
}
