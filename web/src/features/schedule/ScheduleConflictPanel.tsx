import { Badge, EmptyState, SectionCard } from '../../app-shell/AppShell.primitives';
import type { ScheduleConflictPanelProps } from './weeklyScheduleModels';

export function ScheduleConflictPanel({ conflicts, effectiveFrom }: ScheduleConflictPanelProps) {
  return (
    <SectionCard className="stack">
      <div className="section-header">
        <span className="hero-kicker">Conflict summary</span>
        <h2>Conflictos futuros</h2>
        <p>
          El panel reserva espacio para advertencias contra consultas ya cargadas
          {effectiveFrom ? ` desde ${effectiveFrom}` : ''}.
        </p>
      </div>

      {conflicts.length === 0 ? (
        <EmptyState
          title="Sin conflictos visibles"
          description="Cuando exista detección real de conflictos, este bloque mostrará fechas y ventanas afectadas antes de guardar."
        />
      ) : (
        <div className="list">
          {conflicts.map((conflict) => (
            <article key={conflict.id} className="appointment-item">
              <div className="status-bar">
                <strong>
                  {conflict.date} · {conflict.startTime}–{conflict.endTime}
                </strong>
                <Badge tone={conflict.severity === 'critical' ? 'error' : 'info'}>
                  {conflict.severity === 'critical' ? 'Crítico' : 'Advertencia'}
                </Badge>
              </div>
              <span>{conflict.summary}</span>
              {conflict.patientLabel ? <small>Paciente afectado: {conflict.patientLabel}</small> : null}
            </article>
          ))}
        </div>
      )}
    </SectionCard>
  );
}
