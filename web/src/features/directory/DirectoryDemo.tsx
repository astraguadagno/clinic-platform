import { useCallback, useEffect, useState } from 'react';
import { EmptyState, PageContainer, SectionCard } from '../../app-shell/AppShell.primitives';
import { type DirectoryMode } from '../../auth/actorCapabilities';
import { resolveAuthenticatedViewError } from '../../auth/authenticatedViewPolicy';
import { createPatient, createProfessional, listPatients, listProfessionals } from '../../api/directory';
import type { CreatePatientPayload, CreateProfessionalPayload, Patient, Professional } from '../../types/directory';

const EMPTY_PATIENT_FORM: CreatePatientPayload = {
  first_name: '',
  last_name: '',
  document: '',
  birth_date: '',
  phone: '',
  email: '',
};

const EMPTY_PROFESSIONAL_FORM: CreateProfessionalPayload = {
  first_name: '',
  last_name: '',
  specialty: '',
};

type DirectoryDemoProps = {
  directoryMode: DirectoryMode;
  onSessionInvalid: () => void;
};

export function DirectoryDemo({ directoryMode, onSessionInvalid }: DirectoryDemoProps) {
  const [patients, setPatients] = useState<Patient[]>([]);
  const [professionals, setProfessionals] = useState<Professional[]>([]);
  const [patientForm, setPatientForm] = useState<CreatePatientPayload>(EMPTY_PATIENT_FORM);
  const [professionalForm, setProfessionalForm] = useState<CreateProfessionalPayload>(EMPTY_PROFESSIONAL_FORM);
  const [isBootstrapping, setIsBootstrapping] = useState(true);
  const [isCreatingPatient, setIsCreatingPatient] = useState(false);
  const [isCreatingProfessional, setIsCreatingProfessional] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [accessDeniedMessage, setAccessDeniedMessage] = useState('');
  const [successMessage, setSuccessMessage] = useState('');

  const handleApiFailure = useCallback(
    (error: unknown, fallbackMessage: string, forbiddenFallbackMessage: string) => {
      const resolution = resolveAuthenticatedViewError(error, onSessionInvalid, fallbackMessage, forbiddenFallbackMessage);

      if (resolution.kind === 'session-invalid') {
        return { errorMessage: '', accessDeniedMessage: '' };
      }

      if (resolution.kind === 'forbidden') {
        return { errorMessage: '', accessDeniedMessage: resolution.message };
      }

      return { errorMessage: resolution.message, accessDeniedMessage: '' };
    },
    [onSessionInvalid],
  );

  const bootstrap = useCallback(async () => {
    if (directoryMode.kind === 'forbidden') {
      setPatients([]);
      setProfessionals([]);
      setErrorMessage('');
      setAccessDeniedMessage(directoryMode.message);
      setIsBootstrapping(false);
      return;
    }

    try {
      setIsBootstrapping(true);
      setErrorMessage('');
      setAccessDeniedMessage('');

      const [patientsResponse, professionalsResponse] = await Promise.all([listPatients(), listProfessionals()]);

      setPatients(patientsResponse.items);
      setProfessionals(professionalsResponse.items);
    } catch (error) {
      const nextError = handleApiFailure(error, 'No se pudo cargar el directorio demo.', 'No tenés permiso para abrir el directorio.');
      setErrorMessage(nextError.errorMessage);
      setAccessDeniedMessage(nextError.accessDeniedMessage);
    } finally {
      setIsBootstrapping(false);
    }
  }, [directoryMode, handleApiFailure]);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  async function handleCreatePatient() {
    try {
      setIsCreatingPatient(true);
      setErrorMessage('');
      setAccessDeniedMessage('');
      setSuccessMessage('');

      const patient = await createPatient(patientForm);

      setPatients((current) => [patient, ...current]);
      setPatientForm(EMPTY_PATIENT_FORM);
      setSuccessMessage('Paciente creado correctamente.');
    } catch (error) {
      const nextError = handleApiFailure(error, 'No se pudo crear el paciente.', 'No tenés permiso para crear pacientes.');
      setErrorMessage(nextError.errorMessage);
      setAccessDeniedMessage(nextError.accessDeniedMessage);
    } finally {
      setIsCreatingPatient(false);
    }
  }

  async function handleCreateProfessional() {
    try {
      setIsCreatingProfessional(true);
      setErrorMessage('');
      setAccessDeniedMessage('');
      setSuccessMessage('');

      const professional = await createProfessional(professionalForm);

      setProfessionals((current) => [professional, ...current]);
      setProfessionalForm(EMPTY_PROFESSIONAL_FORM);
      setSuccessMessage('Profesional creado correctamente.');
    } catch (error) {
      const nextError = handleApiFailure(error, 'No se pudo crear el profesional.', 'No tenés permiso para crear profesionales.');
      setErrorMessage(nextError.errorMessage);
      setAccessDeniedMessage(nextError.accessDeniedMessage);
    } finally {
      setIsCreatingProfessional(false);
    }
  }

  if (directoryMode.kind === 'forbidden') {
    return (
      <PageContainer>
        <EmptyState eyebrow="Directorio bloqueado" title="Acceso denegado" description={directoryMode.message} />
      </PageContainer>
    );
  }

  return (
    <PageContainer className="stack">
      <SectionCard className="stack">
        <div className="status-bar">
          <span className="badge neutral">Pacientes: {patients.length}</span>
          <span className="badge neutral">Profesionales: {professionals.length}</span>
          <span className="badge info">Superficie de configuración liviana</span>
          {successMessage ? <span className="badge success">{successMessage}</span> : null}
          {errorMessage ? <span className="badge error">{errorMessage}</span> : null}
          {accessDeniedMessage ? <span className="badge error">Acceso denegado: {accessDeniedMessage}</span> : null}
        </div>
        <div className="inline-note">Alta rápida y listados claros para poblar la demo sin mezclar esta superficie con la operación diaria.</div>
      </SectionCard>

      <div className="directory-grid">
        <SectionCard className="stack">
          <div className="section-header">
            <div>
              <h3>Pacientes</h3>
              <p>Formulario corto para cargar personas reales del demo y verlas enseguida en la lista.</p>
            </div>
            <span className="badge neutral">{patients.length} total</span>
          </div>

          <div className="form-grid">
            <div className="field">
              <label htmlFor="patient-first-name">Nombre</label>
              <input
                id="patient-first-name"
                value={patientForm.first_name}
                onChange={(event) => setPatientForm((current) => ({ ...current, first_name: event.target.value }))}
              />
            </div>

            <div className="field">
              <label htmlFor="patient-last-name">Apellido</label>
              <input
                id="patient-last-name"
                value={patientForm.last_name}
                onChange={(event) => setPatientForm((current) => ({ ...current, last_name: event.target.value }))}
              />
            </div>

            <div className="field">
              <label htmlFor="patient-document">Documento</label>
              <input
                id="patient-document"
                value={patientForm.document}
                onChange={(event) => setPatientForm((current) => ({ ...current, document: event.target.value }))}
              />
            </div>

            <div className="field">
              <label htmlFor="patient-birth-date">Fecha de nacimiento</label>
              <input
                id="patient-birth-date"
                type="date"
                value={patientForm.birth_date}
                onChange={(event) => setPatientForm((current) => ({ ...current, birth_date: event.target.value }))}
              />
            </div>

            <div className="field">
              <label htmlFor="patient-phone">Teléfono</label>
              <input
                id="patient-phone"
                value={patientForm.phone}
                onChange={(event) => setPatientForm((current) => ({ ...current, phone: event.target.value }))}
              />
            </div>

            <div className="field">
              <label htmlFor="patient-email">Email</label>
              <input
                id="patient-email"
                type="email"
                value={patientForm.email}
                onChange={(event) => setPatientForm((current) => ({ ...current, email: event.target.value }))}
              />
            </div>
          </div>

          <div className="toolbar">
            <button className="button" type="button" onClick={() => void handleCreatePatient()} disabled={isBootstrapping || isCreatingPatient}>
              {isCreatingPatient ? 'Guardando...' : 'Crear paciente'}
            </button>
            <span className="helper helper-inline">Ideal para poblar rápido la agenda demo.</span>
          </div>

          {isBootstrapping ? (
            <div className="empty-state empty-state-soft">Cargando pacientes...</div>
          ) : patients.length === 0 ? (
            <div className="empty-state">
              <strong>Sin pacientes todavía</strong>
              <span>Cargá uno o dos para que la demo de Agenda tenga nombres reales y se vea más completa.</span>
            </div>
          ) : (
            <div className="list compact-list">
              {patients.map((patient) => (
                <div key={patient.id} className="directory-item">
                  <div>
                    <strong>
                      {patient.first_name} {patient.last_name}
                    </strong>
                    <small>doc {patient.document}</small>
                  </div>
                  <div className="appointment-meta">
                    <span className={`pill${patient.active ? '' : ' cancelled'}`}>{patient.active ? 'Activo' : 'Inactivo'}</span>
                    <span className="muted">{patient.phone}</span>
                    {patient.email ? <span className="muted">{patient.email}</span> : null}
                  </div>
                </div>
              ))}
            </div>
          )}
        </SectionCard>

        <SectionCard className="stack">
          <div className="section-header">
            <div>
              <h3>Profesionales</h3>
              <p>Bloque liviano para cargar especialidades y dejar lista la agenda con datos de verdad.</p>
            </div>
            <span className="badge neutral">{professionals.length} total</span>
          </div>

          <div className="form-grid">
            <div className="field">
              <label htmlFor="professional-first-name">Nombre</label>
              <input
                id="professional-first-name"
                value={professionalForm.first_name}
                onChange={(event) => setProfessionalForm((current) => ({ ...current, first_name: event.target.value }))}
              />
            </div>

            <div className="field">
              <label htmlFor="professional-last-name">Apellido</label>
              <input
                id="professional-last-name"
                value={professionalForm.last_name}
                onChange={(event) => setProfessionalForm((current) => ({ ...current, last_name: event.target.value }))}
              />
            </div>

            <div className="field form-grid-span-full">
              <label htmlFor="professional-specialty">Especialidad</label>
              <input
                id="professional-specialty"
                value={professionalForm.specialty}
                onChange={(event) => setProfessionalForm((current) => ({ ...current, specialty: event.target.value }))}
              />
            </div>
          </div>

          <div className="toolbar">
            <button
              className="button"
              type="button"
              onClick={() => void handleCreateProfessional()}
              disabled={isBootstrapping || isCreatingProfessional}
            >
              {isCreatingProfessional ? 'Guardando...' : 'Crear profesional'}
            </button>
            <span className="helper helper-inline">Después lo vas a poder elegir en Agenda.</span>
          </div>

          {isBootstrapping ? (
            <div className="empty-state empty-state-soft">Cargando profesionales...</div>
          ) : professionals.length === 0 ? (
            <div className="empty-state">
              <strong>Sin profesionales todavía</strong>
              <span>Creá al menos uno para poder generar slots y reservar turnos desde Agenda.</span>
            </div>
          ) : (
            <div className="list compact-list">
              {professionals.map((professional) => (
                <div key={professional.id} className="directory-item">
                  <div>
                    <strong>
                      {professional.first_name} {professional.last_name}
                    </strong>
                    <small>{professional.specialty}</small>
                  </div>
                  <div className="appointment-meta">
                    <span className={`pill${professional.active ? '' : ' cancelled'}`}>
                      {professional.active ? 'Activo' : 'Inactivo'}
                    </span>
                    <span className="muted">Especialidad: {professional.specialty}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </SectionCard>
      </div>
    </PageContainer>
  );
}
