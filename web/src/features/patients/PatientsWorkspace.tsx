import { useCallback, useEffect, useMemo, useState } from 'react';
import { createPatientEncounter, listPatientEncounters } from '../../api/clinical';
import { listPatients } from '../../api/directory';
import { ApiError } from '../../api/http';
import type { AuthUser } from '../../types/auth';
import type { CreateEncounterPayload, Encounter, Patient } from '../../types/clinical';

type PatientsWorkspaceProps = {
  currentUser: AuthUser;
};

type EncounterFormState = {
  note: string;
  occurred_at: string;
};

const EMPTY_ENCOUNTER_FORM: EncounterFormState = {
  note: '',
  occurred_at: '',
};

export function PatientsWorkspace({ currentUser }: PatientsWorkspaceProps) {
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

  const canAccessClinical = Boolean(currentUser.professional_id);
  const activePatients = useMemo(() => patients.filter((patient) => patient.active), [patients]);
  const selectedPatient = activePatients.find((patient) => patient.id === selectedPatientId) ?? null;

  const bootstrap = useCallback(async () => {
    try {
      setIsBootstrapping(true);
      setPatientsError('');

      const response = await listPatients();
      setPatients(response.items);
    } catch (error) {
      setPatientsError(getErrorMessage(error, 'No se pudieron cargar los pacientes activos.'));
    } finally {
      setIsBootstrapping(false);
    }
  }, []);

  const loadEncounters = useCallback(async (patientId: string) => {
    try {
      setIsLoadingEncounters(true);
      setEncountersError('');

      const response = await listPatientEncounters(patientId);
      setEncounters(sortEncounters(response.items));
    } catch (error) {
      setEncounters([]);
      setEncountersError(getErrorMessage(error, 'No se pudieron cargar las evoluciones del paciente.'));
    } finally {
      setIsLoadingEncounters(false);
    }
  }, []);

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

    if (!selectedPatientId || !canAccessClinical) {
      setEncounters([]);
      setEncountersError(
        canAccessClinical ? '' : 'Tu usuario doctor no tiene professional_id, así que no puede abrir ni registrar encounters.',
      );
      return;
    }

    void loadEncounters(selectedPatientId);
  }, [canAccessClinical, loadEncounters, selectedPatientId]);

  async function handleCreateEncounter() {
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
      setFormError(getErrorMessage(error, 'No se pudo registrar la nota clínica.'));
    } finally {
      setIsCreatingEncounter(false);
    }
  }

  return (
    <div className="stack">
      <header className="hero section-hero section-hero-card card">
        <div className="hero-kicker">Doctor workspace · alcance mínimo</div>
        <h2>Pacientes activos y nota inicial</h2>
        <p>
          Esta superficie reemplaza el placeholder y deja un flujo demo-friendly: elegir paciente, ver resumen,
          revisar encounters y registrar una evolución corta sin inventar una historia clínica completa.
        </p>

        <div className="status-bar">
          <span className="badge neutral">Pacientes activos: {activePatients.length}</span>
          <span className="badge neutral">Encounters visibles: {encounters.length}</span>
          <span className={`badge ${canAccessClinical ? 'info' : 'error'}`}>
            {canAccessClinical ? 'Bearer listo para endpoints clínicos' : 'Falta professional_id en la sesión'}
          </span>
          {successMessage ? <span className="badge success">{successMessage}</span> : null}
          {patientsError ? <span className="badge error">{patientsError}</span> : null}
        </div>
      </header>

      <div className="patients-workspace-grid">
        <section className="card stack patients-sidebar">
          <div className="section-header">
            <div>
              <h3>Pacientes activos</h3>
              <p>Lista corta para entrar rápido a una atención sin meter navegación extra.</p>
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
              <span>Cargalos desde Directorio para que esta vista doctor tenga casos reales de demo.</span>
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
                <p>Ficha básica para contexto rápido antes de mirar o escribir una evolución.</p>
              </div>
              {selectedPatient ? <span className="pill">Activo</span> : null}
            </div>

            {!selectedPatient ? (
              <div className="empty-state">
                <strong>Seleccioná un paciente</strong>
                <span>Cuando elijas uno, vas a ver su resumen y las notas iniciales disponibles para tu usuario doctor.</span>
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
                    <span className="summary-label">Profesional actual</span>
                    <strong>{currentUser.email}</strong>
                    <small>{currentUser.professional_id ? `professional_id ${currentUser.professional_id}` : 'Sin vínculo profesional'}</small>
                  </article>
                </div>

                <div className="inline-note">
                  Este alcance es INTENCIONALMENTE chico: solo encounters con nota inicial. Nada de timeline clínica completa,
                  recetas ni estudios todavía.
                </div>
              </>
            )}
          </section>

          <section className="card stack">
            <div className="section-header">
              <div>
                <h3>Encounters</h3>
                <p>Listado descendente por fecha para ver la actividad mínima del paciente seleccionado.</p>
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
              <div className="empty-state empty-state-soft">Elegí un paciente para ver sus encounters.</div>
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
                <p>Formulario corto para registrar una observación inicial y mantener la demo enfocada.</p>
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
                />
              </div>

              <div className="field form-grid-span-full">
                <label htmlFor="encounter-note">Nota breve</label>
                <textarea
                  id="encounter-note"
                  value={form.note}
                  onChange={(event) => setForm((current) => ({ ...current, note: event.target.value }))}
                  placeholder="Ej: Paciente compensado, tolera medicación, se indican controles en 72 h."
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
              <span className="helper helper-inline">Sin router, sin wizard, sin historia clínica completa.</span>
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

function getErrorMessage(error: unknown, fallbackMessage: string) {
  if (error instanceof ApiError) {
    if (error.status === 401) {
      return 'La sesión no llegó al backend clínico. Volvé a iniciar sesión.';
    }

    if (error.status === 403) {
      return 'Este usuario no tiene perfil profesional para operar encounters.';
    }

    return error.message;
  }

  return fallbackMessage;
}
