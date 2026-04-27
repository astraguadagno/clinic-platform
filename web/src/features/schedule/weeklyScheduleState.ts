import type { ScheduleRecurrence, ScheduleTemplateVersion } from '../../types/appointments';
import type {
  VersionSummaryCard,
  WeeklyScheduleConflict,
  WeeklyScheduleDayDraft,
  WeeklyScheduleDayKey,
  WeeklyScheduleDraft,
  WeeklySchedulePreviewDay,
  WeeklyScheduleSaveRequest,
  WeeklyScheduleWindowDraft,
} from './weeklyScheduleModels';
import { WEEKLY_SCHEDULE_DAY_ORDER } from './weeklyScheduleModels';

const DAY_LABELS: Record<WeeklyScheduleDayKey, { label: string; shortLabel: string }> = {
  monday: { label: 'Lunes', shortLabel: 'Lun' },
  tuesday: { label: 'Martes', shortLabel: 'Mar' },
  wednesday: { label: 'Miércoles', shortLabel: 'Mié' },
  thursday: { label: 'Jueves', shortLabel: 'Jue' },
  friday: { label: 'Viernes', shortLabel: 'Vie' },
  saturday: { label: 'Sábado', shortLabel: 'Sáb' },
  sunday: { label: 'Domingo', shortLabel: 'Dom' },
};

export function createWeeklyScheduleDraftFromVersion(version?: ScheduleTemplateVersion | null): WeeklyScheduleDraft {
  const days = WEEKLY_SCHEDULE_DAY_ORDER.reduce<Record<WeeklyScheduleDayKey, WeeklyScheduleDayDraft>>((accumulator, dayKey) => {
    const recurrenceWindow = version?.recurrence[dayKey];

    accumulator[dayKey] = {
      dayKey,
      label: DAY_LABELS[dayKey].label,
      shortLabel: DAY_LABELS[dayKey].shortLabel,
      isEnabled: Boolean(recurrenceWindow),
      windows: recurrenceWindow ? [adaptWindow(dayKey, recurrenceWindow, 1)] : [],
    };

    return accumulator;
  }, {} as Record<WeeklyScheduleDayKey, WeeklyScheduleDayDraft>);

  return {
    effectiveFrom: version?.effective_from ?? '',
    reason: version?.reason ?? '',
    days,
  };
}

export function cloneDraftForNewVersion(version?: ScheduleTemplateVersion | null) {
  const draft = createWeeklyScheduleDraftFromVersion(version);

  return {
    ...draft,
    effectiveFrom: '',
    reason: '',
  };
}

export function toggleDraftDay(draft: WeeklyScheduleDraft, dayKey: WeeklyScheduleDayKey, nextValue: boolean): WeeklyScheduleDraft {
  const nextWindows = nextValue
    ? draft.days[dayKey].windows.length > 0
      ? draft.days[dayKey].windows
      : [createDefaultWindow(dayKey, 1)]
    : [];

  return {
    ...draft,
    days: {
      ...draft.days,
      [dayKey]: {
        ...draft.days[dayKey],
        isEnabled: nextValue,
        windows: nextWindows,
      },
    },
  };
}

export function addWindowToDraftDay(draft: WeeklyScheduleDraft, dayKey: WeeklyScheduleDayKey): WeeklyScheduleDraft {
  const day = draft.days[dayKey];
  const nextOrdinal = day.windows.length + 1;

  return {
    ...draft,
    days: {
      ...draft.days,
      [dayKey]: {
        ...day,
        isEnabled: true,
        windows: [...day.windows, createDefaultWindow(dayKey, nextOrdinal)],
      },
    },
  };
}

export function removeWindowFromDraftDay(draft: WeeklyScheduleDraft, dayKey: WeeklyScheduleDayKey, windowId: string): WeeklyScheduleDraft {
  const day = draft.days[dayKey];
  const nextWindows = day.windows.filter((window) => window.id !== windowId);

  return {
    ...draft,
    days: {
      ...draft.days,
      [dayKey]: {
        ...day,
        isEnabled: nextWindows.length > 0 ? day.isEnabled : false,
        windows: nextWindows,
      },
    },
  };
}

export function updateDraftWindow(
  draft: WeeklyScheduleDraft,
  dayKey: WeeklyScheduleDayKey,
  windowId: string,
  field: keyof Pick<WeeklyScheduleWindowDraft, 'startTime' | 'endTime' | 'slotDurationMinutes'>,
  value: string,
): WeeklyScheduleDraft {
  const day = draft.days[dayKey];

  return {
    ...draft,
    days: {
      ...draft.days,
      [dayKey]: {
        ...day,
        windows: day.windows.map((window) =>
          window.id === windowId
            ? {
                ...window,
                [field]: field === 'slotDurationMinutes' ? Number(value) : value,
              }
            : window,
        ),
      },
    },
  };
}

export function buildWeeklySchedulePreview(draft: WeeklyScheduleDraft): WeeklySchedulePreviewDay[] {
  return WEEKLY_SCHEDULE_DAY_ORDER.map((dayKey) => {
    const day = draft.days[dayKey];
    const validWindows = day.windows.filter(isWindowValid);

    return {
      dayKey,
      label: day.label,
      shortLabel: day.shortLabel,
      isEnabled: day.isEnabled,
      windowCount: validWindows.length,
      slotCount: validWindows.reduce((total, window) => total + calculateWindowSlots(window), 0),
      windows: validWindows.map((window) => `${window.startTime}–${window.endTime} · ${window.slotDurationMinutes} min`),
    };
  });
}

export function filterVisibleScheduleConflicts(conflicts: WeeklyScheduleConflict[], effectiveFrom: string) {
  if (!effectiveFrom) {
    return conflicts;
  }

  return conflicts.filter((conflict) => conflict.date >= effectiveFrom);
}

export function isWeeklyScheduleReadyToSave(draft: WeeklyScheduleDraft) {
  if (!draft.effectiveFrom) {
    return false;
  }

  const activeDays = WEEKLY_SCHEDULE_DAY_ORDER.map((dayKey) => draft.days[dayKey]).filter((day) => day.isEnabled);

  if (activeDays.length === 0) {
    return false;
  }

  return activeDays.every((day) => day.windows.length > 0 && day.windows.every(isWindowValid));
}

export function buildVersionSummaryCard(
  version: ScheduleTemplateVersion | null | undefined,
  type: 'current' | 'future',
): VersionSummaryCard | null {
  if (!version) {
    return null;
  }

  const preview = buildWeeklySchedulePreview(createWeeklyScheduleDraftFromVersion(version));
  const enabledDays = preview.filter((day) => day.windowCount > 0);

  return {
    title: type === 'current' ? 'Agenda activa' : 'Versión futura cargada',
    eyebrow: type === 'current' ? 'Vigente ahora' : 'Cambio pendiente',
    versionLabel: `Versión ${version.version_number}`,
    effectiveFromLabel: version.effective_from,
    scheduleLabel: enabledDays.length > 0 ? enabledDays.map((day) => day.shortLabel ?? day.label).join(' · ') : 'Sin días configurados',
    reasonLabel: version.reason?.trim() || 'Sin motivo registrado',
  };
}

export function createWeeklyScheduleSaveRequest(args: {
  professionalId: string;
  draft: WeeklyScheduleDraft;
  conflicts: WeeklyScheduleConflict[];
}): WeeklyScheduleSaveRequest {
  return {
    professionalId: args.professionalId,
    effectiveFrom: args.draft.effectiveFrom,
    reason: args.draft.reason.trim(),
    recurrence: buildRecurrence(args.draft),
    previewDays: buildWeeklySchedulePreview(args.draft),
    visibleConflicts: filterVisibleScheduleConflicts(args.conflicts, args.draft.effectiveFrom),
  };
}

function buildRecurrence(draft: WeeklyScheduleDraft): ScheduleRecurrence {
  return WEEKLY_SCHEDULE_DAY_ORDER.reduce<ScheduleRecurrence>((recurrence, dayKey) => {
    const day = draft.days[dayKey];
    const firstWindow = day.windows.find(isWindowValid);

    if (!day.isEnabled || !firstWindow) {
      return recurrence;
    }

    recurrence[dayKey] = {
      start_time: firstWindow.startTime,
      end_time: firstWindow.endTime,
      slot_duration_minutes: firstWindow.slotDurationMinutes,
    };

    return recurrence;
  }, {});
}

function calculateWindowSlots(window: WeeklyScheduleWindowDraft) {
  const startMinutes = parseTime(window.startTime);
  const endMinutes = parseTime(window.endTime);

  if (endMinutes <= startMinutes) {
    return 0;
  }

  return Math.floor((endMinutes - startMinutes) / window.slotDurationMinutes);
}

function isWindowValid(window: WeeklyScheduleWindowDraft) {
  return Boolean(window.startTime && window.endTime) && calculateWindowSlots(window) > 0;
}

function parseTime(value: string) {
  const [hours, minutes] = value.split(':').map(Number);
  return hours * 60 + minutes;
}

function adaptWindow(dayKey: WeeklyScheduleDayKey, window: NonNullable<ScheduleTemplateVersion['recurrence'][WeeklyScheduleDayKey]>, ordinal: number) {
  return {
    id: `${dayKey}-${ordinal}`,
    startTime: window.start_time,
    endTime: window.end_time,
    slotDurationMinutes: window.slot_duration_minutes,
  };
}

function createDefaultWindow(dayKey: WeeklyScheduleDayKey, ordinal: number): WeeklyScheduleWindowDraft {
  return {
    id: `${dayKey}-${ordinal}`,
    startTime: '09:00',
    endTime: '12:00',
    slotDurationMinutes: 30,
  };
}
