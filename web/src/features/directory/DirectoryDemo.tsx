import { useCallback, useEffect, useState } from 'react';
import { createPatient, createProfessional, listPatients, listProfessionals } from '../../api/directory';
import { ApiError } from '../../api/http';
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

export function DirectoryDemo() {
  const [patients, setPatients] = useState<Patient[]>([]);
  const [professionals, setProfessionals] = useState<Professional[]>([]);
  const [patientForm, setPatientForm] = useState<CreatePatientPayload>(EMPTY_PATIENT_FORM);
  const [professionalForm, setProfessionalForm] = useState<CreateProfessionalPayload>(EMPTY_PROFESSIONAL_FORM);
  const [isBootstrapping, setIsBootstrapping] = useState(true);
  const [isCreatingPatient, setIsCreatingPatient] = useState(false);
  const [isCreatingProfessional, setIsCreatingProfessional] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [successMessage, setSuccessMessage] = useState('');

  const bootstrap = useCallback(async () => {
    try {
      setIsBootstrapping(true);
      setErrorMessage('');

      const [patientsResponse, professionalsResponse] = await Promise.all([listPatients(), listProfessionals()]);

      setPatients(patientsResponse.items);
      setProfessionals(professionalsResponse.items);
    } catch (error) {
      setErrorMessage(getErrorMessage(error, 'No se pudo cargar el directorio demo.'));
    } finally {
      setIsBootstrapping(false);
    }
  }, []);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  async function handleCreatePatient() {
    try {
      setIsCreatingPatient(true);
      setErrorMessage('');
      setSuccessMessage('');

      const patient = await createPatient(patientForm);

      setPatients((current) => [patient, ...current]);
      setPatientForm(EMPTY_PATIENT_FORM);
      setSuccessMessage('Paciente creado correctamente.');
    } catch (error) {
      setErrorMessage(getErrorMessage(error, 'No se pudo crear el paciente.'));
    } finally {
      setIsCreatingPatient(false);
    }
  }

  async function handleCreateProfessional() {
    try {
      setIsCreatingProfessional(true);
      setErrorMessage('');
      setSuccessMessage('');

      const professional = await createProfessional(professionalForm);

      setProfessionals((current) => [professional, ...current]);
      setProfessionalForm(EMPTY_PROFESSIONAL_FORM);
      setSuccessMessage('Profesional creado correctamente.');
    } catch (error) {
      setErrorMessage(getErrorMessage(error, 'No se pudo crear el profesional.'));
    } finally {
      setIsCreatingProfessional(false);
    }
  }

  return (
    <div className="stack">
      <header className="hero section-hero section-hero-card card">
        <div className="hero-kicker">Base de datos demo</div>
        <h2>Directorio demo</h2>
        <p>
          Alta rápida y listado claro para poblar la demo sin mezclar esta superficie con la operación diaria de la
          agenda.
        </p>
        <div className="status-bar">
          <span className="badge neutral">Pacientes: {patients.length}</span>
          <span className="badge neutral">Profesionales: {professionals.length}</span>
          <span className="badge info">Agenda ve altas nuevas al volver al tab</span>
          {successMessage ? <span className="badge success">{successMessage}</span> : null}
          {errorMessage ? <span className="badge error">{errorMessage}</span> : null}
        </div>
      </header>

      <div className="directory-grid">
        <section className="card stack">
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
        </section>

        <section className="card stack">
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
        </section>
      </div>
    </div>
  );
}

function getErrorMessage(error: unknown, fallbackMessage: string) {
  if (error instanceof ApiError) {
    return error.message;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return fallbackMessage;
}
