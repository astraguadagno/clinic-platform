import { SectionCard } from '../../app-shell/AppShell.primitives';
import type { EffectiveFromPanelProps } from './weeklyScheduleModels';

export function EffectiveFromPanel({ effectiveFrom, reason, onEffectiveFromChange, onReasonChange }: EffectiveFromPanelProps) {
  return (
    <SectionCard className="stack">
      <div className="section-header">
        <span className="hero-kicker">Effective from</span>
        <h2>Vigencia de la nueva versión</h2>
        <p>La fecha de activación es parte central del flujo. No queda escondida como un dato técnico.</p>
      </div>

      <div className="field">
        <label htmlFor="weekly-schedule-effective-from">Vigencia desde</label>
        <input
          id="weekly-schedule-effective-from"
          aria-label="Vigencia desde"
          type="date"
          value={effectiveFrom}
          onChange={(event) => onEffectiveFromChange(event.target.value)}
        />
        <span className="helper">Todo cambio impacta la agenda futura desde esta fecha en adelante.</span>
      </div>

      <div className="field">
        <label htmlFor="weekly-schedule-reason">Motivo visible para el equipo</label>
        <textarea
          id="weekly-schedule-reason"
          rows={3}
          value={reason}
          onChange={(event) => onReasonChange(event.target.value)}
          placeholder="Ej.: cambio de consultorio, nueva disponibilidad, reducción horaria"
        />
      </div>
    </SectionCard>
  );
}
