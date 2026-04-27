import { useEffect, useMemo, useState } from 'react';
import { ContentSplit, PageContainer } from '../../app-shell/AppShell.primitives';
import { EffectiveFromPanel } from './EffectiveFromPanel';
import { ProfessionalScheduleContext } from './ProfessionalScheduleContext';
import { ScheduleConflictPanel } from './ScheduleConflictPanel';
import { WeeklyScheduleActions } from './WeeklyScheduleActions';
import { WeeklySchedulePreview } from './WeeklySchedulePreview';
import { WeeklyTemplateEditor } from './WeeklyTemplateEditor';
import type { WeeklySchedulePageProps } from './weeklyScheduleModels';
import {
  addWindowToDraftDay,
  cloneDraftForNewVersion,
  createWeeklyScheduleSaveRequest,
  filterVisibleScheduleConflicts,
  isWeeklyScheduleReadyToSave,
  removeWindowFromDraftDay,
  toggleDraftDay,
  updateDraftWindow,
  buildWeeklySchedulePreview,
} from './weeklyScheduleState';

export function WeeklySchedulePage({
  actor,
  professionalOptions = [],
  initialSelectedProfessionalId,
  currentVersion,
  futureVersion,
  knownConflicts = [],
  isSaving = false,
  onCancel,
  onProfessionalChange,
  onSave,
}: WeeklySchedulePageProps) {
  const resolvedProfessionalOptions =
    actor.kind === 'doctor'
      ? [{ id: actor.professionalId, label: actor.professionalName }]
      : professionalOptions;
  const fallbackProfessionalId =
    actor.kind === 'doctor' ? actor.professionalId : initialSelectedProfessionalId ?? resolvedProfessionalOptions[0]?.id ?? '';
  const seedVersion = futureVersion ?? currentVersion;

  const [selectedProfessionalId, setSelectedProfessionalId] = useState(fallbackProfessionalId);
  const [draft, setDraft] = useState(() => cloneDraftForNewVersion(seedVersion));

  useEffect(() => {
    setDraft(cloneDraftForNewVersion(seedVersion));
  }, [seedVersion]);

  useEffect(() => {
    setSelectedProfessionalId(fallbackProfessionalId);
  }, [fallbackProfessionalId]);

  function handleProfessionalChange(professionalId: string) {
    setSelectedProfessionalId(professionalId);
    onProfessionalChange?.(professionalId);
  }

  const previewDays = useMemo(() => buildWeeklySchedulePreview(draft), [draft]);
  const visibleConflicts = useMemo(
    () => filterVisibleScheduleConflicts(knownConflicts, draft.effectiveFrom),
    [draft.effectiveFrom, knownConflicts],
  );
  const canSave = isWeeklyScheduleReadyToSave(draft) && Boolean(selectedProfessionalId);

  function handleSave() {
    if (!onSave || !canSave) {
      return;
    }

    onSave(
      createWeeklyScheduleSaveRequest({
        professionalId: selectedProfessionalId,
        draft,
        conflicts: knownConflicts,
      }),
    );
  }

  return (
    <PageContainer className="stack">
      <ProfessionalScheduleContext
        actor={actor}
        professionalOptions={resolvedProfessionalOptions}
        selectedProfessionalId={selectedProfessionalId}
        currentVersion={currentVersion}
        futureVersion={futureVersion}
        onProfessionalChange={handleProfessionalChange}
      />

      <ContentSplit style={{ gridTemplateColumns: 'minmax(0, 1.7fr) minmax(320px, 1fr)' }}>
        <div className="stack">
          <WeeklyTemplateEditor
            draft={draft}
            onToggleDay={(dayKey, nextValue) => setDraft((current) => toggleDraftDay(current, dayKey, nextValue))}
            onAddWindow={(dayKey) => setDraft((current) => addWindowToDraftDay(current, dayKey))}
            onRemoveWindow={(dayKey, windowId) => setDraft((current) => removeWindowFromDraftDay(current, dayKey, windowId))}
            onWindowChange={(dayKey, windowId, field, value) =>
              setDraft((current) => updateDraftWindow(current, dayKey, windowId, field, value))
            }
          />
          <WeeklyScheduleActions canSave={canSave} isSaving={isSaving} onSave={handleSave} onCancel={onCancel} />
        </div>

        <div className="stack">
          <EffectiveFromPanel
            effectiveFrom={draft.effectiveFrom}
            reason={draft.reason}
            onEffectiveFromChange={(value) => setDraft((current) => ({ ...current, effectiveFrom: value }))}
            onReasonChange={(value) => setDraft((current) => ({ ...current, reason: value }))}
          />
          <WeeklySchedulePreview previewDays={previewDays} />
          <ScheduleConflictPanel conflicts={visibleConflicts} effectiveFrom={draft.effectiveFrom} />
        </div>
      </ContentSplit>
    </PageContainer>
  );
}
