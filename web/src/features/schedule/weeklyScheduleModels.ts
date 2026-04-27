import type { ScheduleRecurrence, ScheduleTemplateVersion, ScheduleTemplateWindow } from '../../types/appointments';

export const WEEKLY_SCHEDULE_DAY_ORDER = [
  'monday',
  'tuesday',
  'wednesday',
  'thursday',
  'friday',
  'saturday',
  'sunday',
] as const;

export type WeeklyScheduleDayKey = (typeof WEEKLY_SCHEDULE_DAY_ORDER)[number];

export type WeeklyScheduleWindowDraft = {
  id: string;
  startTime: string;
  endTime: string;
  slotDurationMinutes: ScheduleTemplateWindow['slot_duration_minutes'];
};

export type WeeklyScheduleDayDraft = {
  dayKey: WeeklyScheduleDayKey;
  label: string;
  shortLabel: string;
  isEnabled: boolean;
  windows: WeeklyScheduleWindowDraft[];
};

export type WeeklyScheduleDraft = {
  effectiveFrom: string;
  reason: string;
  days: Record<WeeklyScheduleDayKey, WeeklyScheduleDayDraft>;
};

export type WeeklySchedulePreviewDay = {
  dayKey: WeeklyScheduleDayKey;
  label: string;
  shortLabel: string;
  isEnabled: boolean;
  windowCount: number;
  slotCount: number;
  windows: string[];
};

export type WeeklyScheduleConflict = {
  id: string;
  date: string;
  startTime: string;
  endTime: string;
  patientLabel?: string;
  summary: string;
  severity: 'warning' | 'critical';
};

export type ProfessionalScheduleOption = {
  id: string;
  label: string;
  subtitle?: string;
};

export type WeeklyScheduleActorContext =
  | {
      kind: 'secretary';
      actorLabel: string;
      workspaceLabel?: string;
    }
  | {
      kind: 'doctor';
      actorLabel: string;
      professionalId: string;
      professionalName: string;
    };

export type WeeklyScheduleSaveRequest = {
  professionalId: string;
  effectiveFrom: string;
  reason: string;
  recurrence: ScheduleRecurrence;
  previewDays: WeeklySchedulePreviewDay[];
  visibleConflicts: WeeklyScheduleConflict[];
};

export type WeeklySchedulePageProps = {
  actor: WeeklyScheduleActorContext;
  professionalOptions?: ProfessionalScheduleOption[];
  initialSelectedProfessionalId?: string;
  currentVersion?: ScheduleTemplateVersion | null;
  futureVersion?: ScheduleTemplateVersion | null;
  knownConflicts?: WeeklyScheduleConflict[];
  isSaving?: boolean;
  onCancel?: () => void;
  onProfessionalChange?: (professionalId: string) => void;
  onSave?: (request: WeeklyScheduleSaveRequest) => void;
};

export type VersionSummaryCard = {
  title: string;
  eyebrow: string;
  versionLabel: string;
  effectiveFromLabel: string;
  scheduleLabel: string;
  reasonLabel: string;
};

export type ProfessionalScheduleContextProps = {
  actor: WeeklyScheduleActorContext;
  professionalOptions: ProfessionalScheduleOption[];
  selectedProfessionalId: string;
  currentVersion?: ScheduleTemplateVersion | null;
  futureVersion?: ScheduleTemplateVersion | null;
  onProfessionalChange: (professionalId: string) => void;
};

export type WeeklyTemplateEditorProps = {
  draft: WeeklyScheduleDraft;
  onToggleDay: (dayKey: WeeklyScheduleDayKey, nextValue: boolean) => void;
  onAddWindow: (dayKey: WeeklyScheduleDayKey) => void;
  onRemoveWindow: (dayKey: WeeklyScheduleDayKey, windowId: string) => void;
  onWindowChange: (
    dayKey: WeeklyScheduleDayKey,
    windowId: string,
    field: keyof Pick<WeeklyScheduleWindowDraft, 'startTime' | 'endTime' | 'slotDurationMinutes'>,
    value: string,
  ) => void;
};

export type DayTemplateCardProps = {
  day: WeeklyScheduleDayDraft;
  onToggleDay: (nextValue: boolean) => void;
  onAddWindow: () => void;
  onRemoveWindow: (windowId: string) => void;
  onWindowChange: (
    windowId: string,
    field: keyof Pick<WeeklyScheduleWindowDraft, 'startTime' | 'endTime' | 'slotDurationMinutes'>,
    value: string,
  ) => void;
};

export type ScheduleWindowRowProps = {
  dayLabel: string;
  window: WeeklyScheduleWindowDraft;
  canRemove: boolean;
  onRemove: () => void;
  onChange: (field: keyof Pick<WeeklyScheduleWindowDraft, 'startTime' | 'endTime' | 'slotDurationMinutes'>, value: string) => void;
};

export type EffectiveFromPanelProps = {
  effectiveFrom: string;
  reason: string;
  onEffectiveFromChange: (value: string) => void;
  onReasonChange: (value: string) => void;
};

export type WeeklySchedulePreviewProps = {
  previewDays: WeeklySchedulePreviewDay[];
};

export type ScheduleConflictPanelProps = {
  conflicts: WeeklyScheduleConflict[];
  effectiveFrom: string;
};

export type WeeklyScheduleActionsProps = {
  canSave: boolean;
  isSaving?: boolean;
  onSave: () => void;
  onCancel?: () => void;
};
