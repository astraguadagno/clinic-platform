package appointments

import (
	"context"
	"testing"
)

func TestRepositoryIntegrationScheduleBlocksMigrationCreatesScopedConstraints(t *testing.T) {
	_, db := newPostgresIntegrationRepository(t)

	ctx := context.Background()
	professionalID := "550e8400-e29b-41d4-a716-446655440130"
	templateID := "550e8400-e29b-41d4-a716-446655440131"

	if _, err := db.ExecContext(ctx, `
		INSERT INTO schedule_templates (id, professional_id)
		VALUES ($1, $2)
	`, templateID, professionalID); err != nil {
		t.Fatalf("seed schedule template: %v", err)
	}

	for _, tc := range []struct {
		name  string
		query string
		args  []any
	}{
		{
			name: "single scope",
			query: `
				INSERT INTO schedule_blocks (professional_id, scope, block_date, start_time, end_time)
				VALUES ($1, 'single', '2026-05-05', '09:00', '12:00')
			`,
			args: []any{professionalID},
		},
		{
			name: "range scope",
			query: `
				INSERT INTO schedule_blocks (professional_id, scope, start_date, end_date, start_time, end_time)
				VALUES ($1, 'range', '2026-05-06', '2026-05-10', '13:00', '15:00')
			`,
			args: []any{professionalID},
		},
		{
			name: "template scope",
			query: `
				INSERT INTO schedule_blocks (professional_id, scope, day_of_week, start_time, end_time, template_id)
				VALUES ($1, 'template', 1, '10:00', '11:00', $2)
			`,
			args: []any{professionalID, templateID},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := db.ExecContext(ctx, tc.query, tc.args...); err != nil {
				t.Fatalf("insert %s block: %v", tc.name, err)
			}
		})
	}

	for _, tc := range []struct {
		name  string
		query string
		args  []any
	}{
		{
			name: "invalid scope value",
			query: `
				INSERT INTO schedule_blocks (professional_id, scope, block_date, start_time, end_time)
				VALUES ($1, 'weekly', '2026-05-12', '09:00', '12:00')
			`,
			args: []any{professionalID},
		},
		{
			name: "single scope without date",
			query: `
				INSERT INTO schedule_blocks (professional_id, scope, start_time, end_time)
				VALUES ($1, 'single', '09:00', '12:00')
			`,
			args: []any{professionalID},
		},
		{
			name: "template scope without template reference",
			query: `
				INSERT INTO schedule_blocks (professional_id, scope, day_of_week, start_time, end_time)
				VALUES ($1, 'template', 1, '10:00', '11:00')
			`,
			args: []any{professionalID},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := db.ExecContext(ctx, tc.query, tc.args...); err == nil {
				t.Fatalf("expected %s insert to fail", tc.name)
			}
		})
	}

	if got := countRows(t, db, "schedule_blocks"); got != 3 {
		t.Fatalf("schedule blocks persisted = %d, want 3", got)
	}
}
