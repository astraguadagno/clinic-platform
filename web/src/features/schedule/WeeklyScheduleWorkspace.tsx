import { useEffect, useMemo, useState } from 'react';
import { ApiError } from '../../api/http';
import {
  createScheduleTemplateVersion,
  getScheduleTemplate,
  listScheduleTemplateVersions,
} from '../../api/appointments';
import { listProfessionals } from '../../api/directory';
import { resolveAuthenticatedViewError } from '../../auth/authenticatedViewPolicy';
import type { AgendaMode } from '../../auth/actorCapabilities';
import { Badge, EmptyState, PageContainer } from '../../app-shell/AppShell.primitives';
import type { Professional } from '../../types/directory';
import type { ScheduleTemplateVersion } from '../../types/appointments';
import { WeeklySchedulePage } from './WeeklySchedulePage';
import { buildWeeklyScheduleFlowContext } from './weeklyScheduleFlowAdapter';

const SCHEDULE_LOOKUP_DATE = '2099-12-31';

type WeeklyScheduleWorkspaceProps = {
  agendaMode: AgendaMode;
  onSessionInvalid: () => void;
};

export function WeeklyScheduleWorkspace({ agendaMode, onSessionInvalid }: WeeklyScheduleWorkspaceProps) {
  const [professionals, setProfessionals] = useState<Professional[]>([]);
  const [selectedProfessionalId, setSelectedProfessionalId] = useState('');
  const [isBootstrapping, setIsBootstrapping] = useState(true);
  const [isScheduleLoading, setIsScheduleLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [accessDeniedMessage, setAccessDeniedMessage] = useState('');
  const [successMessage, setSuccessMessage] = useState('');
  const [currentVersion, setCurrentVersion] = useState<ScheduleTemplateVersion | null>(null);
  const [futureVersion, setFutureVersion] = useState<ScheduleTemplateVersion | null>(null);

  useEffect(() => {
    async function bootstrap() {
      if (agendaMode.kind === 'forbidden') {
        setProfessionals([]);
        setSelectedProfessionalId('');
        setAccessDeniedMessage(agendaMode.message);
        setIsBootstrapping(false);
        return;
      }

      try {
        setIsBootstrapping(true);
        setErrorMessage('');
        setAccessDeniedMessage('');

        const response = await listProfessionals();
        const nextProfessionals = response.items.filter((professional) => professional.active);

        setProfessionals(nextProfessionals);
        setSelectedProfessionalId((current) => {
          if (agendaMode.kind === 'doctor-own') {
            return agendaMode.professionalId;
          }

          return nextProfessionals.some((professional) => professional.id === current) ? current : nextProfessionals[0]?.id ?? '';
        });
      } catch (error) {
        const resolution = resolveAuthenticatedViewError(error, onSessionInvalid, 'No se pudo cargar la base profesional.', 'No podés editar esta agenda semanal.');

        if (resolution.kind === 'session-invalid') {
          return;
        }

        if (resolution.kind === 'forbidden') {
          setAccessDeniedMessage(resolution.message);
          return;
        }

        setErrorMessage(resolution.message);
      } finally {
        setIsBootstrapping(false);
      }
    }

    void bootstrap();
  }, [agendaMode, onSessionInvalid]);

  const flowContext = useMemo(
    () =>
      buildWeeklyScheduleFlowContext({
        agendaMode,
        professionals,
        selectedProfessionalId,
      }),
    [agendaMode, professionals, selectedProfessionalId],
  );

  useEffect(() => {
    async function loadScheduleSnapshot() {
      if (agendaMode.kind === 'forbidden' || !selectedProfessionalId) {
        setCurrentVersion(null);
        setFutureVersion(null);
        return;
      }

      try {
        setIsScheduleLoading(true);
        setErrorMessage('');
        setAccessDeniedMessage('');

        const template = await getScheduleTemplate({
          professional_id: selectedProfessionalId,
          effective_date: SCHEDULE_LOOKUP_DATE,
        });
        const versionsResponse = await listScheduleTemplateVersions({ template_id: template.id });
        const timeline = resolveScheduleTimeline(versionsResponse.items, getTodayDate());

        setCurrentVersion(timeline.currentVersion);
        setFutureVersion(timeline.futureVersion);
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) {
          setCurrentVersion(null);
          setFutureVersion(null);
          return;
        }

        const resolution = resolveAuthenticatedViewError(error, onSessionInvalid, 'No se pudo cargar la agenda semanal guardada.', 'No podés editar esta agenda semanal.');

        if (resolution.kind === 'session-invalid') {
          return;
        }

        if (resolution.kind === 'forbidden') {
          setAccessDeniedMessage(resolution.message);
          return;
        }

        setErrorMessage(resolution.message);
      } finally {
        setIsScheduleLoading(false);
      }
    }

    void loadScheduleSnapshot();
  }, [agendaMode, onSessionInvalid, selectedProfessionalId]);

  if (agendaMode.kind === 'forbidden') {
    return (
      <PageContainer className="stack">
        <EmptyState eyebrow="Agenda semanal bloqueada" title="Acceso denegado" description={agendaMode.message} />
      </PageContainer>
    );
  }

  return (
    <PageContainer className="stack">
      {(successMessage || errorMessage || accessDeniedMessage) && (
        <div className="status-bar" aria-live="polite">
          {successMessage ? <span className="badge success">{successMessage}</span> : null}
          {errorMessage ? <span className="badge error">{errorMessage}</span> : null}
          {accessDeniedMessage ? <span className="badge error">Acceso denegado: {accessDeniedMessage}</span> : null}
        </div>
      )}

      <div className="inline-note">
        <strong>Superficie dedicada de agenda semanal</strong>
        <span>Guardá una nueva versión que impacte desde la fecha elegida sin mezclar template con turnos diarios.</span>
      </div>

      {isBootstrapping ? (
        <EmptyState eyebrow="Agenda semanal" title="Cargando contexto profesional" description="Estamos preparando la plantilla semanal antes de editarla." />
      ) : isScheduleLoading ? (
        <EmptyState eyebrow="Agenda semanal" title="Cargando versiones guardadas" description="Estamos trayendo la versión activa y la próxima versión desde backend." />
      ) : (
        <>
          {agendaMode.kind === 'operational-shared' && professionals.length === 0 ? (
            <div className="inline-note inline-note-error">
              <strong>No hay profesionales activos para editar.</strong>
              <span>Necesitás al menos un profesional disponible para preparar la agenda semanal.</span>
            </div>
          ) : null}

          <Badge tone="info">Persistencia real conectada · preview/conflictos siguen acotados</Badge>

          <WeeklySchedulePage
            actor={flowContext.actor}
            professionalOptions={flowContext.professionalOptions}
            initialSelectedProfessionalId={flowContext.initialSelectedProfessionalId}
            currentVersion={currentVersion}
            futureVersion={futureVersion}
            isSaving={isSaving}
            onProfessionalChange={(professionalId) => {
              setSelectedProfessionalId(professionalId);
              setSuccessMessage('');
            }}
            onSave={async (request) => {
              try {
                setIsSaving(true);
                setErrorMessage('');
                setAccessDeniedMessage('');
                setSuccessMessage('');

                await createScheduleTemplateVersion({
                  professional_id: request.professionalId,
                  effective_from: request.effectiveFrom,
                  recurrence: request.recurrence,
                  reason: request.reason || undefined,
                });

                const template = await getScheduleTemplate({
                  professional_id: request.professionalId,
                  effective_date: SCHEDULE_LOOKUP_DATE,
                });
                const versionsResponse = await listScheduleTemplateVersions({ template_id: template.id });
                const timeline = resolveScheduleTimeline(versionsResponse.items, getTodayDate());

                setCurrentVersion(timeline.currentVersion);
                setFutureVersion(timeline.futureVersion);
                setSuccessMessage('Versión semanal guardada y recargada desde backend.');
              } catch (error) {
                const resolution = resolveAuthenticatedViewError(error, onSessionInvalid, 'No se pudo guardar la agenda semanal.', 'No podés guardar esta agenda semanal.');

                if (resolution.kind === 'session-invalid') {
                  return;
                }

                if (resolution.kind === 'forbidden') {
                  setAccessDeniedMessage(resolution.message);
                  return;
                }

                setErrorMessage(resolution.message);
              } finally {
                setIsSaving(false);
              }
            }}
          />
        </>
      )}
    </PageContainer>
  );
}

function resolveScheduleTimeline(versions: ScheduleTemplateVersion[], today: string) {
  const currentVersion = versions.find((version) => version.effective_from <= today) ?? null;
  const futureVersion = versions.find((version) => version.effective_from > today) ?? null;

  return {
    currentVersion,
    futureVersion,
  };
}

function getTodayDate() {
  return new Date().toISOString().slice(0, 10);
}
