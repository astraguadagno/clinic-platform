import { EmptyState, SectionCard, SummaryTile } from '../../app-shell/AppShell.primitives';
import type { WeeklySchedulePreviewProps } from './weeklyScheduleModels';

export function WeeklySchedulePreview({ previewDays }: WeeklySchedulePreviewProps) {
  const activeDays = previewDays.filter((day) => day.windowCount > 0);
  const totalSlots = activeDays.reduce((total, day) => total + day.slotCount, 0);

  return (
    <SectionCard className="stack">
      <div className="section-header">
        <span className="hero-kicker">Weekly preview</span>
        <h2>Preview semanal</h2>
        <p>Mostrá la forma futura de la agenda antes de confirmar la nueva versión.</p>
      </div>

      <div className="foundation-content-split" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))' }}>
        <SummaryTile className="stack-tight">
          <span className="summary-label">Días activos</span>
          <strong>{activeDays.length}</strong>
        </SummaryTile>
        <SummaryTile className="stack-tight">
          <span className="summary-label">Ventanas válidas</span>
          <strong>{activeDays.reduce((total, day) => total + day.windowCount, 0)}</strong>
        </SummaryTile>
        <SummaryTile className="stack-tight">
          <span className="summary-label">Slots estimados</span>
          <strong>{totalSlots}</strong>
        </SummaryTile>
      </div>

      {activeDays.length === 0 ? (
        <EmptyState title="Sin días activos todavía" description="Activá al menos un día y definí una ventana para obtener vista previa." />
      ) : (
        <div className="list">
          {activeDays.map((day) => (
            <article key={day.dayKey} className="appointment-item">
              <strong>{day.label}</strong>
              <span>{day.windows.join(' · ')}</span>
              <small>{day.slotCount} slots estimados</small>
            </article>
          ))}
        </div>
      )}
    </SectionCard>
  );
}
