import { useCallback, useEffect, useMemo, useState } from 'react';
import { type PatientsMode } from '../../auth/actorCapabilities';
import { resolveAuthenticatedViewError } from '../../auth/authenticatedViewPolicy';
import { createPatientEncounter, listPatientEncounters } from '../../api/clinical';
import { listPatients } from '../../api/directory';
import type { CreateEncounterPayload, Encounter, Patient } from '../../types/clinical';

type PatientsWorkspaceProps = {
  patientsMode: PatientsMode;
  onSessionInvalid: () => void;
};

type EncounterFormState = {
  note: string;
  occurred_at: string;
};

const EMPTY_ENCOUNTER_FORM: EncounterFormState = {
  note: '',
  occurred_at: '',
};

export function PatientsWorkspace({ patientsMode, onSessionInvalid }: PatientsWorkspaceProps) {
  const [patients, setPatients] = useState<Patient[]>([]);
  const [selectedPatientId, setSelectedPatientId] = useState('');
  const [encounters, setEncounters] = useState<Encounter[]>([]);
  const [form, setForm] = useState<EncounterFormState>(EMPTY_ENCOUNTER_FORM);
  const [isBootstrapping, setIsBootstrapping] = useState(true);
  const [isLoadingEncounters, setIsLoadingEncounters] = useState(false);
  const [isCreatingEncounter, setIsCreatingEncounter] = useState(false);
  const [patientsError, setPatientsError] = useState('');
  const [encountersError, setEncountersError] = useState('');
  const [formError, setFormError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');

  const activePatients = useMemo(() => patients.filter((patient) => patient.active), [patients]);
  const selectedPatient = activePatients.find((patient) => patient.id === selectedPatientId) ?? null;
  const canAccessClinical = patientsMode.kind === 'doctor-clinical';
  const clinicalDeniedMessage =
    patientsMode.kind === 'secretary-operational'
      ? 'Este perfil secretaría puede buscar y seleccionar pacientes, pero no ver ni registrar encounters clínicos.'
      : patientsMode.kind === 'forbidden'
        ? patientsMode.message
        : '';

  const resolveViewError = useCallback(
    (error: unknown, fallbackMessage: string, forbiddenFallbackMessage: string) => {
      const resolution = resolveAuthenticatedViewError(error, onSessionInvalid, fallbackMessage, forbiddenFallbackMessage);

      if (resolution.kind === 'session-invalid') {
        return { errorMessage: '', deniedMessage: '' };
      }

      if (resolution.kind === 'forbidden') {
        return { errorMessage: '', deniedMessage: resolution.message };
      }

      return { errorMessage: resolution.message, deniedMessage: '' };
    },
    [onSessionInvalid],
  );

  const bootstrap = useCallback(async () => {
    if (patientsMode.kind === 'forbidden') {
      setPatients([]);
      setSelectedPatientId('');
      setEncounters([]);
      setPatientsError('');
      setEncountersError(patientsMode.message);
      setIsBootstrapping(false);
      return;
    }

    try {
      setIsBootstrapping(true);
      setPatientsError('');
      setEncountersError(patientsMode.kind === 'secretary-operational' ? clinicalDeniedMessage : '');

      const response = await listPatients();
      setPatients(response.items);
    } catch (error) {
      const nextError = resolveViewError(
        error,
        'No se pudieron cargar los pacientes activos.',
        'No tenés permiso para consultar pacientes.',
      );
      setPatientsError(nextError.errorMessage);
      setEncountersError(nextError.deniedMessage);
    } finally {
      setIsBootstrapping(false);
    }
  }, [clinicalDeniedMessage, patientsMode, resolveViewError]);

  const loadEncounters = useCallback(
    async (patientId: string) => {
      if (!canAccessClinical) {
        setEncounters([]);
        setEncountersError(clinicalDeniedMessage);
        return;
      }

      try {
        setIsLoadingEncounters(true);
        setEncountersError('');

        const response = await listPatientEncounters(patientId);
        setEncounters(sortEncounters(response.items));
      } catch (error) {
        setEncounters([]);
        const nextError = resolveViewError(
          error,
          'No se pudieron cargar las evoluciones del paciente.',
          'No tenés permiso para consultar encounters clínicos.',
        );
        setEncountersError(nextError.deniedMessage || nextError.errorMessage);
      } finally {
        setIsLoadingEncounters(false);
      }
    },
    [canAccessClinical, clinicalDeniedMessage, resolveViewError],
  );

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  useEffect(() => {
    setSelectedPatientId((current) => {
      if (current && activePatients.some((patient) => patient.id === current)) {
        return current;
      }

      return activePatients[0]?.id ?? '';
    });
  }, [activePatients]);

  useEffect(() => {
    setSuccessMessage('');
    setFormError('');

    if (!selectedPatientId) {
      setEncounters([]);
      setEncountersError(canAccessClinical ? '' : clinicalDeniedMessage);
      return;
    }

    if (!canAccessClinical) {
      setEncounters([]);
      setEncountersError(clinicalDeniedMessage);
      return;
    }

    void loadEncounters(selectedPatientId);
  }, [canAccessClinical, clinicalDeniedMessage, loadEncounters, selectedPatientId]);

  async function handleCreateEncounter() {
    if (!canAccessClinical) {
      setFormError(clinicalDeniedMessage);
      return;
    }

    if (!selectedPatientId) {
      setFormError('Elegí un paciente antes de registrar una nota.');
      return;
    }

    const note = form.note.trim();
    if (!note) {
      setFormError('Escribí una nota breve antes de guardar.');
      return;
    }

    const payload: CreateEncounterPayload = { note };

    if (form.occurred_at) {
      const occurredAt = new Date(form.occurred_at);
      if (Number.isNaN(occurredAt.getTime())) {
        setFormError('La fecha/hora ingresada no es válida.');
        return;
      }

      payload.occurred_at = occurredAt.toISOString();
    }

    try {
      setIsCreatingEncounter(true);
      setFormError('');
      setSuccessMessage('');

      const encounter = await createPatientEncounter(selectedPatientId, payload);

      setEncounters((current) => sortEncounters([encounter, ...current]));
      setForm(EMPTY_ENCOUNTER_FORM);
      setSuccessMessage('Nota inicial registrada correctamente.');
    } catch (error) {
      const nextError = resolveViewError(
        error,
        'No se pudo registrar la nota clínica.',
        'No tenés permiso para registrar encounters clínicos.',
      );
      setFormError(nextError.deniedMessage || nextError.errorMessage);
    } finally {
      setIsCreatingEncounter(false);
    }
  }

  if (patientsMode.kind === 'forbidden') {
    return (
      <section className="card stack" aria-live="polite">
        <div className="hero-kicker">Pacientes bloqueado</div>
        <h2>Acceso denegado</h2>
        <p>{patientsMode.message}</p>
      </section>
    );
  }

  return (
    <div className="stack">
      <section className="card stack">
        <div className="status-bar">
          <span className="badge neutral">Pacientes activos: {activePatients.length}</span>
          <span className="badge neutral">Encounters visibles: {encounters.length}</span>
          <span className={`badge ${canAccessClinical ? 'info' : 'neutral'}`}>
            {canAccessClinical ? 'Modo clínico habilitado' : 'Modo operativo sin clinical encounters'}
          </span>
          {successMessage ? <span className="badge success">{successMessage}</span> : null}
          {patientsError ? <span className="badge error">{patientsError}</span> : null}
        </div>
        <div className="inline-note">
          {patientsMode.kind === 'doctor-clinical'
            ? 'Elegí paciente, revisá encounters y registrá una evolución corta sin inventar una historia clínica completa.'
            : 'Este cuerpo mantiene foco operativo: búsqueda y selección para secretaría sin habilitar trabajo clínico.'}
        </div>
      </section>

      <div className="patients-workspace-grid">
        <section className="card stack patients-sidebar">
          <div className="section-header">
            <div>
              <h3>Pacientes activos</h3>
              <p>Lista corta para entrar rápido a una atención o resolver la agenda sin meter navegación extra.</p>
            </div>
            <button className="button secondary" type="button" onClick={() => void bootstrap()} disabled={isBootstrapping}>
              {isBootstrapping ? 'Actualizando...' : 'Actualizar'}
            </button>
          </div>

          {isBootstrapping ? (
            <div className="empty-state empty-state-soft">Cargando pacientes...</div>
          ) : activePatients.length === 0 ? (
            <div className="empty-state">
              <strong>No hay pacientes activos</strong>
              <span>Cargalos desde Directorio para que esta vista tenga casos reales de demo.</span>
            </div>
          ) : (
            <div className="list compact-list patient-selector-list">
              {activePatients.map((patient) => {
                const isSelected = patient.id === selectedPatientId;

                return (
                  <button
                    key={patient.id}
                    className={`patient-selector-card${isSelected ? ' selected' : ''}`}
                    type="button"
                    onClick={() => setSelectedPatientId(patient.id)}
                  >
                    <span className="surface-tab-eyebrow">Paciente</span>
                    <strong>
                      {patient.first_name} {patient.last_name}
                    </strong>
                    <small>Documento {patient.document}</small>
                    <small>{patient.phone || 'Sin teléfono cargado'}</small>
                  </button>
                );
              })}
            </div>
          )}
        </section>

        <div className="stack patients-main">
          <section className="card stack">
            <div className="section-header">
              <div>
                <h3>Resumen del paciente</h3>
                <p>Ficha básica para contexto rápido antes de una tarea clínica u operativa.</p>
              </div>
              {selectedPatient ? <span className="pill">Activo</span> : null}
            </div>

            {!selectedPatient ? (
              <div className="empty-state">
                <strong>Seleccioná un paciente</strong>
                <span>Cuando elijas uno, vas a ver su resumen y el alcance habilitado para este actor.</span>
              </div>
            ) : (
              <>
                <div className="patient-summary-grid">
                  <article className="overview-card">
                    <span className="summary-label">Paciente</span>
                    <strong>
                      {selectedPatient.first_name} {selectedPatient.last_name}
                    </strong>
                    <small>ID {selectedPatient.id}</small>
                  </article>
                  <article className="overview-card">
                    <span className="summary-label">Documento</span>
                    <strong>{selectedPatient.document}</strong>
                    <small>Nacimiento: {formatDate(selectedPatient.birth_date)}</small>
                  </article>
                  <article className="overview-card">
                    <span className="summary-label">Contacto</span>
                    <strong>{selectedPatient.phone || 'Sin teléfono'}</strong>
                    <small>{selectedPatient.email || 'Sin email cargado'}</small>
                  </article>
                  <article className="overview-card">
                    <span className="summary-label">Modo</span>
                    <strong>{canAccessClinical ? 'Clínico' : 'Operativo'}</strong>
                    <small>
                      {canAccessClinical
                        ? `professional_id ${patientsMode.professionalId}`
                        : 'Sin lectura ni escritura de encounters clínicos'}
                    </small>
                  </article>
                </div>

                <div className="inline-note">
                  {canAccessClinical
                    ? 'Este alcance es INTENCIONALMENTE chico: solo encounters con nota inicial. Nada de timeline clínica completa, recetas ni estudios todavía.'
                    : 'Secretaría puede seleccionar y verificar datos del paciente, pero NO abrir trabajo clínico ni registrar evoluciones.'}
                </div>
              </>
            )}
          </section>

          <section className="card stack">
            <div className="section-header">
              <div>
                <h3>Encounters</h3>
                <p>Listado descendente por fecha para ver la actividad clínica cuando el actor lo tiene habilitado.</p>
              </div>
              <button
                className="button secondary"
                type="button"
                onClick={() => (selectedPatientId ? void loadEncounters(selectedPatientId) : undefined)}
                disabled={!selectedPatientId || !canAccessClinical || isLoadingEncounters}
              >
                {isLoadingEncounters ? 'Actualizando...' : 'Recargar'}
              </button>
            </div>

            {encountersError ? <div className="inline-note inline-note-error">{encountersError}</div> : null}

            {!selectedPatient ? (
              <div className="empty-state empty-state-soft">Elegí un paciente para ver su alcance disponible.</div>
            ) : !canAccessClinical ? (
              <div className="empty-state">
                <strong>Encounters clínicos bloqueados</strong>
                <span>{clinicalDeniedMessage}</span>
              </div>
            ) : isLoadingEncounters ? (
              <div className="empty-state empty-state-soft">Cargando encounters...</div>
            ) : encounters.length === 0 ? (
              <div className="empty-state">
                <strong>Sin encounters todavía</strong>
                <span>Registrá abajo una nota breve para dejar la primera evolución de esta demo.</span>
              </div>
            ) : (
              <div className="list">
                {encounters.map((encounter) => (
                  <article key={encounter.id} className="encounter-card">
                    <div className="section-header">
                      <div>
                        <span className="surface-tab-eyebrow">Encounter</span>
                        <h4>{formatDateTime(encounter.occurred_at)}</h4>
                      </div>
                      <span className="badge neutral">{encounter.initial_note.kind}</span>
                    </div>
                    <p className="encounter-note">{encounter.initial_note.content}</p>
                    <div className="appointment-meta">
                      <span className="muted">Creado: {formatDateTime(encounter.created_at)}</span>
                      <span className="muted">Profesional: {encounter.professional_id}</span>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </section>

          <section className="card stack">
            <div className="section-header">
              <div>
                <h3>Nueva nota / evolución</h3>
                <p>Formulario corto para registrar una observación inicial solo cuando el actor tiene alcance clínico.</p>
              </div>
            </div>

            <div className="form-grid">
              <div className="field">
                <label htmlFor="encounter-occurred-at">Fecha y hora</label>
                <input
                  id="encounter-occurred-at"
                  type="datetime-local"
                  value={form.occurred_at}
                  onChange={(event) => setForm((current) => ({ ...current, occurred_at: event.target.value }))}
                  disabled={!canAccessClinical}
                />
              </div>

              <div className="field form-grid-span-full">
                <label htmlFor="encounter-note">Nota breve</label>
                <textarea
                  id="encounter-note"
                  value={form.note}
                  onChange={(event) => setForm((current) => ({ ...current, note: event.target.value }))}
                  placeholder="Ej: Paciente compensado, tolera medicación, se indican controles en 72 h."
                  disabled={!canAccessClinical}
                />
              </div>
            </div>

            {formError ? <div className="inline-note inline-note-error">{formError}</div> : null}

            <div className="toolbar">
              <button
                className="button"
                type="button"
                onClick={() => void handleCreateEncounter()}
                disabled={!selectedPatientId || !canAccessClinical || isCreatingEncounter}
              >
                {isCreatingEncounter ? 'Guardando...' : 'Guardar nota'}
              </button>
              <span className="helper helper-inline">
                {canAccessClinical ? 'Sin router, sin wizard, sin historia clínica completa.' : 'Modo operativo: sin escritura clínica.'}
              </span>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

function sortEncounters(encounters: Encounter[]) {
  return [...encounters].sort((left, right) => {
    const leftTime = Date.parse(left.occurred_at) || Date.parse(left.created_at) || 0;
    const rightTime = Date.parse(right.occurred_at) || Date.parse(right.created_at) || 0;
    return rightTime - leftTime;
  });
}

function formatDate(value: string) {
  const parsedDate = new Date(value);

  if (Number.isNaN(parsedDate.getTime())) {
    return value;
  }

  return parsedDate.toLocaleDateString('es-AR', {
    dateStyle: 'medium',
  });
}

function formatDateTime(value: string) {
  const parsedDate = new Date(value);

  if (Number.isNaN(parsedDate.getTime())) {
    return value;
  }

  return parsedDate.toLocaleString('es-AR', {
    dateStyle: 'medium',
    timeStyle: 'short',
  });
}
