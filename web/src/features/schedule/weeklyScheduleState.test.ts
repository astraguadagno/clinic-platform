import { describe, expect, it } from 'vitest';
import type { ScheduleTemplateVersion } from '../../types/appointments';
import type { WeeklyScheduleConflict } from './weeklyScheduleModels';
import {
  buildWeeklySchedulePreview,
  createWeeklyScheduleDraftFromVersion,
  createWeeklyScheduleSaveRequest,
  filterVisibleScheduleConflicts,
  isWeeklyScheduleReadyToSave,
} from './weeklyScheduleState';

describe('weeklyScheduleState', () => {
  it('maps an existing template version into editable day cards and preview slot counts', () => {
    const draft = createWeeklyScheduleDraftFromVersion(activeVersion());
    const preview = buildWeeklySchedulePreview(draft);

    expect(draft.days.monday.isEnabled).toBe(true);
    expect(draft.days.tuesday.isEnabled).toBe(false);
    expect(draft.days.wednesday.windows).toHaveLength(1);
    expect(preview.find((day) => day.dayKey === 'monday')).toMatchObject({
      windowCount: 1,
      slotCount: 6,
    });
    expect(preview.find((day) => day.dayKey === 'wednesday')).toMatchObject({
      windowCount: 1,
      slotCount: 4,
    });
  });

  it('requires effective_from before save and filters visible conflicts from that date onward', () => {
    const baseDraft = createWeeklyScheduleDraftFromVersion(activeVersion());
    const conflicts: WeeklyScheduleConflict[] = [
      {
        id: 'conflict-before',
        date: '2026-04-28',
        startTime: '09:00',
        endTime: '09:30',
        patientLabel: 'Juan Pérez',
        summary: 'Consulta ya reservada',
        severity: 'warning',
      },
      {
        id: 'conflict-after',
        date: '2026-05-06',
        startTime: '10:00',
        endTime: '10:30',
        patientLabel: 'María Gómez',
        summary: 'El nuevo horario deja sin cobertura el turno reservado',
        severity: 'critical',
      },
    ];

    expect(isWeeklyScheduleReadyToSave({ ...baseDraft, effectiveFrom: '' })).toBe(false);

    const draft = { ...baseDraft, effectiveFrom: '2026-05-01', reason: 'Nuevo esquema de consultorio' };
    const visibleConflicts = filterVisibleScheduleConflicts(conflicts, draft.effectiveFrom);
    const request = createWeeklyScheduleSaveRequest({
      professionalId: 'professional-1',
      draft,
      conflicts,
    });

    expect(visibleConflicts.map((conflict) => conflict.id)).toEqual(['conflict-after']);
    expect(isWeeklyScheduleReadyToSave(draft)).toBe(true);
    expect(request).toMatchObject({
      professionalId: 'professional-1',
      effectiveFrom: '2026-05-01',
      reason: 'Nuevo esquema de consultorio',
    });
    expect(request.recurrence.monday).toMatchObject({
      start_time: '09:00',
      end_time: '12:00',
      slot_duration_minutes: 30,
    });
    expect(request.recurrence.tuesday).toBeUndefined();
    expect(request.visibleConflicts.map((conflict) => conflict.id)).toEqual(['conflict-after']);
  });
});

function activeVersion(): ScheduleTemplateVersion {
  return {
    id: 'version-1',
    template_id: 'template-1',
    version_number: 3,
    effective_from: '2026-04-01',
    recurrence: {
      monday: {
        start_time: '09:00',
        end_time: '12:00',
        slot_duration_minutes: 30,
      },
      wednesday: {
        start_time: '14:00',
        end_time: '16:00',
        slot_duration_minutes: 30,
      },
    },
    created_at: '2026-03-25T10:00:00Z',
    created_by: 'secretary-1',
    reason: 'Base operativa vigente',
  };
}
