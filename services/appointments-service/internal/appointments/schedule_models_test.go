package appointments

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestScheduleTemplateModelMatchesSchemaContract(t *testing.T) {
	t.Parallel()

	templateType := reflect.TypeOf(ScheduleTemplate{})

	assertField(t, templateType, "ID", reflect.TypeOf(""), "id")
	assertField(t, templateType, "ProfessionalID", reflect.TypeOf(""), "professional_id")
	assertField(t, templateType, "CreatedAt", reflect.TypeOf(time.Time{}), "created_at")
	assertField(t, templateType, "UpdatedAt", reflect.TypeOf(time.Time{}), "updated_at")
	assertField(t, templateType, "Versions", reflect.TypeOf([]ScheduleTemplateVersion{}), "versions,omitempty")
}

func TestScheduleTemplateVersionModelMatchesSchemaContract(t *testing.T) {
	t.Parallel()

	versionType := reflect.TypeOf(ScheduleTemplateVersion{})

	assertField(t, versionType, "ID", reflect.TypeOf(""), "id")
	assertField(t, versionType, "TemplateID", reflect.TypeOf(""), "template_id")
	assertField(t, versionType, "VersionNumber", reflect.TypeOf(0), "version_number")
	assertField(t, versionType, "EffectiveFrom", reflect.TypeOf(time.Time{}), "effective_from")
	assertField(t, versionType, "Recurrence", reflect.TypeOf(json.RawMessage{}), "recurrence")
	assertField(t, versionType, "CreatedAt", reflect.TypeOf(time.Time{}), "created_at")
	assertField(t, versionType, "CreatedBy", reflect.TypeOf((*string)(nil)), "created_by,omitempty")
	assertField(t, versionType, "Reason", reflect.TypeOf((*string)(nil)), "reason,omitempty")
}

func TestScheduleBlockModelMatchesSchemaContract(t *testing.T) {
	t.Parallel()

	blockType := reflect.TypeOf(ScheduleBlock{})

	assertField(t, blockType, "ID", reflect.TypeOf(""), "id")
	assertField(t, blockType, "ProfessionalID", reflect.TypeOf(""), "professional_id")
	assertField(t, blockType, "Scope", reflect.TypeOf(""), "scope")
	assertField(t, blockType, "BlockDate", reflect.TypeOf((*time.Time)(nil)), "block_date,omitempty")
	assertField(t, blockType, "StartDate", reflect.TypeOf((*time.Time)(nil)), "start_date,omitempty")
	assertField(t, blockType, "EndDate", reflect.TypeOf((*time.Time)(nil)), "end_date,omitempty")
	assertField(t, blockType, "DayOfWeek", reflect.TypeOf((*int)(nil)), "day_of_week,omitempty")
	assertField(t, blockType, "StartTime", reflect.TypeOf(""), "start_time")
	assertField(t, blockType, "EndTime", reflect.TypeOf(""), "end_time")
	assertField(t, blockType, "TemplateID", reflect.TypeOf((*string)(nil)), "template_id,omitempty")
	assertField(t, blockType, "CreatedAt", reflect.TypeOf(time.Time{}), "created_at")
	assertField(t, blockType, "UpdatedAt", reflect.TypeOf(time.Time{}), "updated_at")
}

func assertField(t *testing.T, typ reflect.Type, name string, wantType reflect.Type, wantJSONTag string) {
	t.Helper()

	field, ok := typ.FieldByName(name)
	if !ok {
		t.Fatalf("missing field %s", name)
	}
	if field.Type != wantType {
		t.Fatalf("field %s type = %v, want %v", name, field.Type, wantType)
	}
	if got := field.Tag.Get("json"); got != wantJSONTag {
		t.Fatalf("field %s json tag = %q, want %q", name, got, wantJSONTag)
	}
}
