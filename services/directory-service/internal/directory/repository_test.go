package directory

import (
	"errors"
	"testing"
	"time"
)

func TestValidateCreatePatientParams(t *testing.T) {
	tests := []struct {
		name     string
		params   CreatePatientParams
		want     CreatePatientParams
		wantDate time.Time
		wantErr  error
	}{
		{
			name: "valid patient",
			params: CreatePatientParams{
				FirstName: "Ada",
				LastName:  "Lovelace",
				Document:  "123",
				BirthDate: "1990-10-10",
				Phone:     "555",
				Email:     "ada@example.com",
			},
			want: CreatePatientParams{
				FirstName: "Ada",
				LastName:  "Lovelace",
				Document:  "123",
				BirthDate: "1990-10-10",
				Phone:     "555",
				Email:     "ada@example.com",
			},
			wantDate: time.Date(1990, 10, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "valid patient trims spaces",
			params: CreatePatientParams{
				FirstName: "  Ada  ",
				LastName:  "  Lovelace ",
				Document:  " 123 ",
				BirthDate: " 1990-10-10 ",
				Phone:     " 555 ",
				Email:     " ada@example.com ",
			},
			want: CreatePatientParams{
				FirstName: "Ada",
				LastName:  "Lovelace",
				Document:  "123",
				BirthDate: "1990-10-10",
				Phone:     "555",
				Email:     "ada@example.com",
			},
			wantDate: time.Date(1990, 10, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "missing required field",
			params: CreatePatientParams{
				FirstName: " ",
				LastName:  "Lovelace",
				Document:  "123",
				BirthDate: "1990-10-10",
				Phone:     "555",
			},
			wantErr: ErrValidation,
		},
		{
			name: "invalid birth date",
			params: CreatePatientParams{
				FirstName: "Ada",
				LastName:  "Lovelace",
				Document:  "123",
				BirthDate: "10-10-1990",
				Phone:     "555",
			},
			wantErr: ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotDate, err := validateCreatePatientParams(tt.params)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			if got != tt.want {
				t.Fatalf("params = %+v, want %+v", got, tt.want)
			}
			if !gotDate.Equal(tt.wantDate) {
				t.Fatalf("birthDate = %v, want %v", gotDate, tt.wantDate)
			}
		})
	}
}

func TestValidateCreateProfessionalParams(t *testing.T) {
	tests := []struct {
		name    string
		params  CreateProfessionalParams
		want    CreateProfessionalParams
		wantErr error
	}{
		{
			name: "valid professional",
			params: CreateProfessionalParams{
				FirstName: "Ana",
				LastName:  "Lopez",
				Specialty: "cardiology",
			},
			want: CreateProfessionalParams{
				FirstName: "Ana",
				LastName:  "Lopez",
				Specialty: "cardiology",
			},
		},
		{
			name: "valid professional trims spaces",
			params: CreateProfessionalParams{
				FirstName: "  Ana ",
				LastName:  " Lopez  ",
				Specialty: " cardiology ",
			},
			want: CreateProfessionalParams{
				FirstName: "Ana",
				LastName:  "Lopez",
				Specialty: "cardiology",
			},
		},
		{
			name: "missing specialty",
			params: CreateProfessionalParams{
				FirstName: "Ana",
				LastName:  "Lopez",
				Specialty: " ",
			},
			wantErr: ErrValidation,
		},
		{
			name: "missing first name",
			params: CreateProfessionalParams{
				FirstName: " ",
				LastName:  "Lopez",
				Specialty: "cardiology",
			},
			wantErr: ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateCreateProfessionalParams(tt.params)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			if got != tt.want {
				t.Fatalf("params = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestValidatePatientDocumentLookup(t *testing.T) {
	tests := []struct {
		name     string
		document string
		want     string
		wantErr  error
	}{
		{name: "trims document", document: "  12345678  ", want: "12345678"},
		{name: "rejects blank document", document: "   ", wantErr: ErrValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validatePatientDocumentLookup(tt.document)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got != tt.want {
				t.Fatalf("document = %q, want %q", got, tt.want)
			}
		})
	}
}
