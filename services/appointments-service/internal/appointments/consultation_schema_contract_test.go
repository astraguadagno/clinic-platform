package appointments

import (
	"strings"
	"testing"
)

func TestCurrentBootstrapConsultationsSchemaSupportsRuntimeContract(t *testing.T) {
	migration := readAppointmentsMigrationFile(t, "001_init.sql")

	checks := []struct {
		name string
		want string
	}{
		{name: "nullable slot for standalone consultations", want: "slot_id UUID"},
		{name: "scheduled start range", want: "scheduled_start TIMESTAMPTZ NOT NULL"},
		{name: "scheduled end range", want: "scheduled_end TIMESTAMPTZ NOT NULL"},
		{name: "check-in metadata", want: "check_in_time TIMESTAMPTZ"},
		{name: "reception metadata", want: "reception_notes TEXT"},
	}

	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(migration, tt.want) {
				t.Fatalf("001_init.sql missing %q", tt.want)
			}
		})
	}

	if strings.Contains(migration, "slot_id UUID NOT NULL") {
		t.Fatal("001_init.sql keeps consultations.slot_id NOT NULL, want nullable for standalone consultations")
	}
}

func TestForwardConsultationEntityMigrationAddsRuntimeMetadataAndStandaloneSupport(t *testing.T) {
	migration := readAppointmentsMigrationFile(t, "006_consultation_entity.sql")

	checks := []struct {
		name string
		want string
	}{
		{name: "check-in metadata", want: "ADD COLUMN check_in_time TIMESTAMPTZ"},
		{name: "reception metadata", want: "ADD COLUMN reception_notes TEXT"},
		{name: "standalone consultations", want: "ALTER COLUMN slot_id DROP NOT NULL"},
	}

	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(migration, tt.want) {
				t.Fatalf("006_consultation_entity.sql missing %q", tt.want)
			}
		})
	}
}
