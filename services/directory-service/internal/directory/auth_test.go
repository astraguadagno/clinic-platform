package directory

import (
	"errors"
	"testing"
)

func TestValidateLoginParams(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		password     string
		wantEmail    string
		wantPassword string
		wantErr      error
	}{
		{
			name:         "valid credentials are normalized",
			email:        "  ADMIN@Clinic.Local ",
			password:     "  admin123 ",
			wantEmail:    "admin@clinic.local",
			wantPassword: "admin123",
		},
		{
			name:     "missing email returns validation error",
			email:    " ",
			password: "admin123",
			wantErr:  ErrValidation,
		},
		{
			name:     "missing password returns validation error",
			email:    "admin@clinic.local",
			password: " ",
			wantErr:  ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEmail, gotPassword, err := validateLoginParams(tt.email, tt.password)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			if gotEmail != tt.wantEmail {
				t.Fatalf("email = %q, want %q", gotEmail, tt.wantEmail)
			}
			if gotPassword != tt.wantPassword {
				t.Fatalf("password = %q, want %q", gotPassword, tt.wantPassword)
			}
		})
	}
}
