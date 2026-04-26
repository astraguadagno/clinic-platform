import { ActionBar, InlineNotice, SectionCard } from '../../app-shell/AppShell.primitives';
import type { WeeklyScheduleActionsProps } from './weeklyScheduleModels';

export function WeeklyScheduleActions({ canSave, isSaving = false, onSave, onCancel }: WeeklyScheduleActionsProps) {
  const isSaveDisabled = !canSave || isSaving;

  return (
    <SectionCard className="stack">
      <div className="section-header">
        <span className="hero-kicker">Weekly schedule actions</span>
        <h2>Acciones</h2>
        <p>La skeleton deja explícito qué falta completar antes de intentar persistir una nueva versión.</p>
      </div>

      {!canSave ? (
        <InlineNotice>
          Completá una fecha de vigencia y al menos un día con ventana válida para habilitar el guardado de la versión semanal.
        </InlineNotice>
      ) : null}

      {isSaving ? (
        <InlineNotice>
          Guardando la nueva versión semanal contra el backend real y recargando el estado visible.
        </InlineNotice>
      ) : null}

      <ActionBar style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(180px, max-content))' }}>
        <button type="button" className="button" onClick={onSave} disabled={isSaveDisabled}>
          {isSaving ? 'Guardando versión semanal…' : 'Guardar versión semanal'}
        </button>
        {onCancel ? (
          <button type="button" className="button secondary" onClick={onCancel}>
            Cancelar edición
          </button>
        ) : null}
      </ActionBar>
    </SectionCard>
  );
}
