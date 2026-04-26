import { Badge, SectionCard, SummaryTile } from '../../app-shell/AppShell.primitives';
import type { ProfessionalScheduleContextProps } from './weeklyScheduleModels';
import { buildVersionSummaryCard } from './weeklyScheduleState';

export function ProfessionalScheduleContext({
  actor,
  professionalOptions,
  selectedProfessionalId,
  currentVersion,
  futureVersion,
  onProfessionalChange,
}: ProfessionalScheduleContextProps) {
  const currentCard = buildVersionSummaryCard(currentVersion, 'current');
  const futureCard = buildVersionSummaryCard(futureVersion, 'future');

  return (
    <SectionCard className="stack">
      <div className="section-header">
        <span className="hero-kicker">Professional schedule context</span>
        <h2>Contexto profesional</h2>
        <p>Separá quién opera la edición, qué agenda está vigente hoy y si ya existe una versión futura preparada.</p>
      </div>

      <div className="status-bar">
        <Badge tone="info">{actor.kind === 'secretary' ? 'Secretaría' : 'Agenda propia'}</Badge>
        <Badge>{actor.kind === 'secretary' ? actor.actorLabel : 'Edición personal'}</Badge>
      </div>

      {actor.kind === 'secretary' ? (
        <div className="field">
          <label htmlFor="weekly-schedule-professional">Profesional</label>
          <select
            id="weekly-schedule-professional"
            aria-label="Profesional"
            value={selectedProfessionalId}
            onChange={(event) => onProfessionalChange(event.target.value)}
          >
            {professionalOptions.map((professional) => (
              <option key={professional.id} value={professional.id}>
                {professional.label}
                {professional.subtitle ? ` — ${professional.subtitle}` : ''}
              </option>
            ))}
          </select>
          <span className="helper">La secretaría puede cambiar de profesional sin mezclar esta edición con la operatoria diaria.</span>
        </div>
      ) : (
        <div className="inline-note">
          <strong>{actor.professionalName}</strong>
          <span className="helper">Solo podés preparar versiones sobre tu agenda profesional.</span>
        </div>
      )}

      <div className="foundation-content-split" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))' }}>
        {currentCard ? <VersionSummary card={currentCard} /> : null}
        {futureCard ? <VersionSummary card={futureCard} /> : null}
      </div>
    </SectionCard>
  );
}

function VersionSummary({ card }: { card: NonNullable<ReturnType<typeof buildVersionSummaryCard>> }) {
  return (
    <SummaryTile className="stack-tight">
      <span className="summary-label">{card.eyebrow}</span>
      <strong>{card.title}</strong>
      <span>{card.versionLabel}</span>
      <small>Vigencia: {card.effectiveFromLabel}</small>
      <small>{card.scheduleLabel}</small>
      <small>{card.reasonLabel}</small>
    </SummaryTile>
  );
}
