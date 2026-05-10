package appointments

import (
	"bufio"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestConsultationOpenAPIEnumsMatchDomainContract(t *testing.T) {
	t.Parallel()

	document := readAppointmentsOpenAPI(t)

	wantStatuses := []string{
		string(ConsultationStatusScheduled),
		string(ConsultationStatusRequested),
		string(ConsultationStatusCheckedIn),
		string(ConsultationStatusCompleted),
		string(ConsultationStatusCancelled),
		string(ConsultationStatusNoShow),
	}
	wantSources := []string{
		string(ConsultationSourceOnline),
		string(ConsultationSourceSecretary),
		string(ConsultationSourceDoctor),
		string(ConsultationSourcePatient),
	}

	tests := []struct {
		name     string
		schema   string
		property string
		want     []string
	}{
		{name: "consultation response status", schema: "Consultation", property: "status", want: wantStatuses},
		{name: "consultation response source", schema: "Consultation", property: "source", want: wantSources},
		{name: "create consultation source", schema: "CreateConsultationRequest", property: "source", want: wantSources},
		{name: "update consultation status", schema: "UpdateConsultationStatusRequest", property: "status", want: wantStatuses},
		{name: "update consultation source echo", schema: "UpdateConsultationStatusRequest", property: "source", want: wantSources},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := openAPIEnumValues(t, document, tt.schema, tt.property)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("%s.%s enum = %#v, want %#v", tt.schema, tt.property, got, tt.want)
			}
		})
	}
}

func readAppointmentsOpenAPI(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile("../../openapi/openapi.yaml")
	if err != nil {
		t.Fatalf("read appointments OpenAPI document: %v", err)
	}
	return string(raw)
}

func openAPIEnumValues(t *testing.T, document, schema, property string) []string {
	t.Helper()

	scanner := bufio.NewScanner(strings.NewReader(document))
	schemaHeader := "    " + schema + ":"
	propertyHeader := "        " + property + ":"

	inSchema := false
	inProperty := false
	inEnum := false
	values := []string{}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inSchema {
			if line == schemaHeader {
				inSchema = true
			}
			continue
		}

		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") && line != schemaHeader {
			break
		}

		if !inProperty {
			if line == propertyHeader {
				inProperty = true
			}
			continue
		}

		if strings.HasPrefix(line, "        ") && !strings.HasPrefix(line, "          ") && line != propertyHeader {
			break
		}

		if !inEnum {
			if trimmed == "enum:" {
				inEnum = true
			}
			continue
		}

		if !strings.HasPrefix(trimmed, "- ") {
			break
		}
		values = append(values, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("scan appointments OpenAPI document: %v", err)
	}
	if len(values) == 0 {
		t.Fatalf("enum not found for %s.%s", schema, property)
	}
	return values
}
