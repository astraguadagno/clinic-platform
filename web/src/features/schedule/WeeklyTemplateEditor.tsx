import { SectionCard } from '../../app-shell/AppShell.primitives';
import type { DayTemplateCardProps, ScheduleWindowRowProps, WeeklyTemplateEditorProps } from './weeklyScheduleModels';

const SLOT_DURATION_OPTIONS = [15, 20, 30, 60] as const;

export function WeeklyTemplateEditor({ draft, onToggleDay, onAddWindow, onRemoveWindow, onWindowChange }: WeeklyTemplateEditorProps) {
  return (
    <SectionCard className="stack">
      <div className="section-header">
        <span className="hero-kicker">Weekly template editor</span>
        <h2>Editor semanal</h2>
        <p>Trabajá día por día. Este bloque define template semanal, no reservas ni cancelaciones puntuales.</p>
        <p>Primer contrato real: una ventana por día. Cuando backend soporte más, recién ahí abrimos multi-ventana.</p>
      </div>

      <div className="list">
        {Object.values(draft.days).map((day) => (
          <DayTemplateCard
            key={day.dayKey}
            day={day}
            onToggleDay={(nextValue) => onToggleDay(day.dayKey, nextValue)}
            onAddWindow={() => onAddWindow(day.dayKey)}
            onRemoveWindow={(windowId) => onRemoveWindow(day.dayKey, windowId)}
            onWindowChange={(windowId, field, value) => onWindowChange(day.dayKey, windowId, field, value)}
          />
        ))}
      </div>
    </SectionCard>
  );
}

export function DayTemplateCard({ day, onToggleDay, onAddWindow, onRemoveWindow, onWindowChange }: DayTemplateCardProps) {
  return (
    <article className="card stack-tight">
      <div className="status-bar">
        <div className="field" style={{ margin: 0 }}>
          <label>
            <input type="checkbox" checked={day.isEnabled} onChange={(event) => onToggleDay(event.target.checked)} /> Habilitar {day.label}
          </label>
        </div>
        <span className="helper">{day.isEnabled ? 'Ventanas activas en template semanal.' : 'Día sin atención configurada.'}</span>
      </div>

      {day.isEnabled ? (
        <>
          <div className="list">
            {day.windows.map((window) => (
              <ScheduleWindowRow
                key={window.id}
                dayLabel={day.label}
                window={window}
                canRemove={day.windows.length > 1}
                onRemove={() => onRemoveWindow(window.id)}
                onChange={(field, value) => onWindowChange(window.id, field, value)}
              />
            ))}
          </div>
        </>
      ) : null}
    </article>
  );
}

export function ScheduleWindowRow({ dayLabel, window, canRemove, onRemove, onChange }: ScheduleWindowRowProps) {
  return (
    <div className="form-grid" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))' }}>
      <div className="field">
        <label htmlFor={`${window.id}-start`}>{dayLabel} · Desde</label>
        <input id={`${window.id}-start`} type="time" value={window.startTime} onChange={(event) => onChange('startTime', event.target.value)} />
      </div>
      <div className="field">
        <label htmlFor={`${window.id}-end`}>{dayLabel} · Hasta</label>
        <input id={`${window.id}-end`} type="time" value={window.endTime} onChange={(event) => onChange('endTime', event.target.value)} />
      </div>
      <div className="field">
        <label htmlFor={`${window.id}-duration`}>{dayLabel} · Slot</label>
        <select id={`${window.id}-duration`} value={window.slotDurationMinutes} onChange={(event) => onChange('slotDurationMinutes', event.target.value)}>
          {SLOT_DURATION_OPTIONS.map((option) => (
            <option key={option} value={option}>
              {option} minutos
            </option>
          ))}
        </select>
      </div>
      <div className="field">
        <label>&nbsp;</label>
        <button type="button" className="button ghost" onClick={onRemove} disabled={!canRemove}>
          Quitar ventana
        </button>
      </div>
    </div>
  );
}
