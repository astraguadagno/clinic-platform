import { type FocusEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ContextualHeader, EmptyState, PageContainer, SectionCard } from '../../app-shell/AppShell.primitives';
import { type PatientsMode } from '../../auth/actorCapabilities';
import { resolveAuthenticatedViewError } from '../../auth/authenticatedViewPolicy';
import {
  createPatientClinicalNote,
  createPatientEncounter,
  deletePatientClinicalNote,
  getClinicalHistory,
  listPatientClinicalNotes,
  listPatientEncounters,
  updateClinicalHistory,
  updatePatientClinicalNote,
} from '../../api/clinical';
import { listPatients } from '../../api/directory';
import type { ClinicalHistory, CreateEncounterPayload, Encounter, Patient, PatientClinicalNote, UpdateClinicalHistoryPayload } from '../../types/clinical';
import { filterPatients, normalizePatientSearchValue } from '../patient-search/matching';

type PatientsWorkspaceProps = {
  patientsMode: PatientsMode;
  onSessionInvalid: () => void;
  onOpenDirectorySupport?: () => void;
};

type EncounterFormState = {
  note: string;
  occurred_at: string;
};

type ClinicalHistoryFormState = {
  weight_kg: string;
  height_cm: string;
  antecedentes: string;
  allergies: string;
  habitual_medication: string;
  chronic_conditions: string;
  habits: string;
  general_observations: string;
};

type PatientClinicalNoteFormState = {
  content: string;
  consultation_id: string;
};

const EMPTY_ENCOUNTER_FORM: EncounterFormState = {
  note: '',
  occurred_at: '',
};

const EMPTY_HISTORY_FORM: ClinicalHistoryFormState = {
  weight_kg: '',
  height_cm: '',
  antecedentes: '',
  allergies: '',
  habitual_medication: '',
  chronic_conditions: '',
  habits: '',
  general_observations: '',
};

const EMPTY_NOTE_FORM: PatientClinicalNoteFormState = {
  content: '',
  consultation_id: '',
};

export function PatientsWorkspace({ patientsMode, onSessionInvalid, onOpenDirectorySupport }: PatientsWorkspaceProps) {
  const [patients, setPatients] = useState<Patient[]>([]);
  const [selectedPatientId, setSelectedPatientId] = useState('');
  const [encounters, setEncounters] = useState<Encounter[]>([]);
  const [clinicalHistory, setClinicalHistory] = useState<ClinicalHistory | null>(null);
  const [historyForm, setHistoryForm] = useState<ClinicalHistoryFormState>(EMPTY_HISTORY_FORM);
  const [patientClinicalNotes, setPatientClinicalNotes] = useState<PatientClinicalNote[]>([]);
  const [clinicalNoteForm, setClinicalNoteForm] = useState<PatientClinicalNoteFormState>(EMPTY_NOTE_FORM);
  const [editingClinicalNoteId, setEditingClinicalNoteId] = useState('');
  const [editingClinicalNoteForm, setEditingClinicalNoteForm] = useState<PatientClinicalNoteFormState>(EMPTY_NOTE_FORM);
  const [sidebarQuery, setSidebarQuery] = useState('');
  const [form, setForm] = useState<EncounterFormState>(EMPTY_ENCOUNTER_FORM);
  const [isBootstrapping, setIsBootstrapping] = useState(true);
  const [isLoadingEncounters, setIsLoadingEncounters] = useState(false);
  const [isLoadingHistory, setIsLoadingHistory] = useState(false);
  const [isSavingHistory, setIsSavingHistory] = useState(false);
  const [isLoadingClinicalNotes, setIsLoadingClinicalNotes] = useState(false);
  const [isSavingClinicalNote, setIsSavingClinicalNote] = useState(false);
  const [isCreatingEncounter, setIsCreatingEncounter] = useState(false);
  const [patientsError, setPatientsError] = useState('');
  const [encountersError, setEncountersError] = useState('');
  const [historyError, setHistoryError] = useState('');
  const [clinicalNotesError, setClinicalNotesError] = useState('');
  const [formError, setFormError] = useState('');
  const [clinicalNoteFormError, setClinicalNoteFormError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const searchControlRef = useRef<HTMLDivElement | null>(null);
  const clinicalHistoryRequestRef = useRef(0);
  const clinicalNotesRequestRef = useRef(0);

  const activePatients = useMemo(() => patients.filter((patient) => patient.active), [patients]);
  const filteredPatients = useMemo(() => filterPatients(activePatients, sidebarQuery), [activePatients, sidebarQuery]);
  const selectedPatient = activePatients.find((patient) => patient.id === selectedPatientId) ?? null;
  const isSelectedPatientHiddenByFilter = useMemo(
    () => Boolean(selectedPatient && normalizePatientSearchValue(sidebarQuery) && !filteredPatients.some((patient) => patient.id === selectedPatient.id)),
    [filteredPatients, selectedPatient, sidebarQuery],
  );
  const hasActiveSearchQuery = Boolean(normalizePatientSearchValue(sidebarQuery));
  const shouldShowSearchDropdown =
    !isBootstrapping &&
    activePatients.length > 0 &&
    isSearchOpen &&
    (hasActiveSearchQuery || filteredPatients.length > 0 || isSelectedPatientHiddenByFilter);
  const canAccessClinical = patientsMode.kind === 'doctor-clinical';
  const isSecretaryOperational = patientsMode.kind === 'secretary-operational';
  const clinicalDeniedMessage =
    patientsMode.kind === 'secretary-operational'
      ? 'Este perfil secretaría puede buscar y seleccionar pacientes, pero no ver ni registrar encounters clínicos.'
      : patientsMode.kind === 'forbidden'
        ? patientsMode.message
        : '';
  const patientContextTitle = selectedPatient
    ? `${selectedPatient.first_name} ${selectedPatient.last_name}`
    : 'Seleccioná un paciente';
  const patientContextDescription = selectedPatient
    ? canAccessClinical
      ? 'Contexto clínico del paciente seleccionado; la ficha, notas y evoluciones se mantienen separadas.'
      : 'Contexto operativo del paciente seleccionado; el trabajo clínico permanece bloqueado para este actor.'
    : 'Usá el buscador para elegir un paciente antes de abrir una ficha clínica activa.';

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
      setClinicalHistory(null);
      setHistoryForm(EMPTY_HISTORY_FORM);
      setPatientClinicalNotes([]);
      setPatientsError('');
      setEncountersError(patientsMode.message);
      setHistoryError(patientsMode.message);
      setClinicalNotesError(patientsMode.message);
      setIsBootstrapping(false);
      return;
    }

    try {
      setIsBootstrapping(true);
      setPatientsError('');
      setEncountersError(patientsMode.kind === 'secretary-operational' ? clinicalDeniedMessage : '');
      setHistoryError(patientsMode.kind === 'secretary-operational' ? clinicalDeniedMessage : '');
      setClinicalNotesError(patientsMode.kind === 'secretary-operational' ? clinicalDeniedMessage : '');

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
      setHistoryError(nextError.deniedMessage);
      setClinicalNotesError(nextError.deniedMessage);
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

  const loadClinicalHistory = useCallback(
    async (patientId: string) => {
      const requestId = clinicalHistoryRequestRef.current + 1;
      clinicalHistoryRequestRef.current = requestId;

      if (!canAccessClinical) {
        setClinicalHistory(null);
        setHistoryForm(EMPTY_HISTORY_FORM);
        setHistoryError(clinicalDeniedMessage);
        return;
      }

      try {
        setIsLoadingHistory(true);
        setHistoryError('');
        setClinicalHistory(null);
        setHistoryForm(EMPTY_HISTORY_FORM);

        const history = await getClinicalHistory(patientId);
        if (clinicalHistoryRequestRef.current !== requestId) {
          return;
        }

        setClinicalHistory(history);
        setHistoryForm(clinicalHistoryToForm(history));
      } catch (error) {
        if (clinicalHistoryRequestRef.current !== requestId) {
          return;
        }

        setClinicalHistory(null);
        setHistoryForm(EMPTY_HISTORY_FORM);
        const nextError = resolveViewError(
          error,
          'No se pudo cargar la ficha clínica del paciente.',
          'No tenés permiso para consultar la ficha clínica.',
        );
        setHistoryError(nextError.deniedMessage || nextError.errorMessage);
      } finally {
        if (clinicalHistoryRequestRef.current === requestId) {
          setIsLoadingHistory(false);
        }
      }
    },
    [canAccessClinical, clinicalDeniedMessage, resolveViewError],
  );

  const loadClinicalNotes = useCallback(
    async (patientId: string) => {
      const requestId = clinicalNotesRequestRef.current + 1;
      clinicalNotesRequestRef.current = requestId;

      if (!canAccessClinical) {
        setPatientClinicalNotes([]);
        setClinicalNotesError(clinicalDeniedMessage);
        return;
      }

      try {
        setIsLoadingClinicalNotes(true);
        setClinicalNotesError('');
        setPatientClinicalNotes([]);

        const response = await listPatientClinicalNotes(patientId);
        if (clinicalNotesRequestRef.current !== requestId) {
          return;
        }

        setPatientClinicalNotes(sortPatientClinicalNotes(response.items));
      } catch (error) {
        if (clinicalNotesRequestRef.current !== requestId) {
          return;
        }

        setPatientClinicalNotes([]);
        const nextError = resolveViewError(
          error,
          'No se pudieron cargar las notas clínicas del paciente.',
          'No tenés permiso para consultar notas clínicas.',
        );
        setClinicalNotesError(nextError.deniedMessage || nextError.errorMessage);
      } finally {
        if (clinicalNotesRequestRef.current === requestId) {
          setIsLoadingClinicalNotes(false);
        }
      }
    },
    [canAccessClinical, clinicalDeniedMessage, resolveViewError],
  );

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  useEffect(() => {
    if (isBootstrapping || activePatients.length === 0) {
      setIsSearchOpen(false);
    }
  }, [activePatients.length, isBootstrapping]);

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
      setClinicalHistory(null);
      setHistoryForm(EMPTY_HISTORY_FORM);
      setHistoryError(canAccessClinical ? '' : clinicalDeniedMessage);
      setPatientClinicalNotes([]);
      setClinicalNotesError(canAccessClinical ? '' : clinicalDeniedMessage);
      return;
    }

    if (!canAccessClinical) {
      setEncounters([]);
      setEncountersError(clinicalDeniedMessage);
      setClinicalHistory(null);
      setHistoryForm(EMPTY_HISTORY_FORM);
      setHistoryError(clinicalDeniedMessage);
      setPatientClinicalNotes([]);
      setClinicalNotesError(clinicalDeniedMessage);
      return;
    }

    void loadEncounters(selectedPatientId);
    void loadClinicalHistory(selectedPatientId);
    void loadClinicalNotes(selectedPatientId);
  }, [canAccessClinical, clinicalDeniedMessage, loadClinicalHistory, loadClinicalNotes, loadEncounters, selectedPatientId]);

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

  async function handleSaveClinicalHistory() {
    if (!canAccessClinical) {
      setHistoryError(clinicalDeniedMessage);
      return;
    }

    if (!selectedPatientId) {
      setHistoryError('Elegí un paciente antes de guardar la ficha.');
      return;
    }

    const payload = historyFormToPayload(historyForm);
    if (!payload) {
      setHistoryError('Peso y altura deben ser números válidos.');
      return;
    }

    try {
      setIsSavingHistory(true);
      setHistoryError('');
      setSuccessMessage('');

      const history = await updateClinicalHistory(selectedPatientId, payload);
      setClinicalHistory(history);
      setHistoryForm(clinicalHistoryToForm(history));
      setSuccessMessage('Ficha clínica guardada correctamente.');
    } catch (error) {
      const nextError = resolveViewError(
        error,
        'No se pudo guardar la ficha clínica.',
        'No tenés permiso para guardar la ficha clínica.',
      );
      setHistoryError(nextError.deniedMessage || nextError.errorMessage);
    } finally {
      setIsSavingHistory(false);
    }
  }

  async function handleCreateClinicalNote() {
    if (!canAccessClinical) {
      setClinicalNoteFormError(clinicalDeniedMessage);
      return;
    }

    if (!selectedPatientId) {
      setClinicalNoteFormError('Elegí un paciente antes de guardar una nota clínica.');
      return;
    }

    const content = clinicalNoteForm.content.trim();
    if (!content) {
      setClinicalNoteFormError('Escribí una nota clínica antes de guardar.');
      return;
    }

    try {
      setIsSavingClinicalNote(true);
      setClinicalNoteFormError('');
      setClinicalNotesError('');
      setSuccessMessage('');

      const note = await createPatientClinicalNote(selectedPatientId, {
        content,
        consultation_id: normalizeNullableText(clinicalNoteForm.consultation_id),
      });
      setPatientClinicalNotes((current) => sortPatientClinicalNotes([note, ...current]));
      setClinicalNoteForm(EMPTY_NOTE_FORM);
      setSuccessMessage('Nota clínica guardada correctamente.');
    } catch (error) {
      const nextError = resolveViewError(
        error,
        'No se pudo guardar la nota clínica.',
        'No tenés permiso para guardar notas clínicas.',
      );
      setClinicalNoteFormError(nextError.deniedMessage || nextError.errorMessage);
    } finally {
      setIsSavingClinicalNote(false);
    }
  }

  async function handleUpdateClinicalNote(noteId: string) {
    if (!selectedPatientId) {
      return;
    }

    const content = editingClinicalNoteForm.content.trim();
    if (!content) {
      setClinicalNotesError('La nota clínica editada no puede quedar vacía.');
      return;
    }

    try {
      setClinicalNotesError('');
      const note = await updatePatientClinicalNote(selectedPatientId, noteId, {
        content,
        consultation_id: normalizeNullableText(editingClinicalNoteForm.consultation_id),
      });
      setPatientClinicalNotes((current) => sortPatientClinicalNotes(current.map((item) => (item.id === noteId ? note : item))));
      setEditingClinicalNoteId('');
      setEditingClinicalNoteForm(EMPTY_NOTE_FORM);
    } catch (error) {
      const nextError = resolveViewError(
        error,
        'No se pudo editar la nota clínica.',
        'No tenés permiso para editar notas clínicas.',
      );
      setClinicalNotesError(nextError.deniedMessage || nextError.errorMessage);
    }
  }

  async function handleDeleteClinicalNote(noteId: string) {
    if (!selectedPatientId) {
      return;
    }

    try {
      setClinicalNotesError('');
      await deletePatientClinicalNote(selectedPatientId, noteId);
      setPatientClinicalNotes((current) => current.filter((note) => note.id !== noteId));
    } catch (error) {
      const nextError = resolveViewError(
        error,
        'No se pudo eliminar la nota clínica.',
        'No tenés permiso para eliminar notas clínicas.',
      );
      setClinicalNotesError(nextError.deniedMessage || nextError.errorMessage);
    }
  }

  function startEditingClinicalNote(note: PatientClinicalNote) {
    setEditingClinicalNoteId(note.id);
    setEditingClinicalNoteForm({
      content: note.content,
      consultation_id: note.consultation_id ?? '',
    });
  }

  function handleSearchBlur(event: FocusEvent<HTMLDivElement>) {
    if (searchControlRef.current?.contains(event.relatedTarget as Node | null)) {
      return;
    }

    setIsSearchOpen(false);
  }

  function handlePatientSelection(patientId: string) {
    setSelectedPatientId(patientId);
    setIsSearchOpen(false);
  }

  if (patientsMode.kind === 'forbidden') {
    return (
      <PageContainer>
        <EmptyState eyebrow="Pacientes bloqueado" title="Acceso denegado" description={patientsMode.message} />
      </PageContainer>
    );
  }

  return (
    <PageContainer className="stack">
      <SectionCard className="stack patient-search-surface">
        <div className="status-bar">
          <span className="badge neutral">Pacientes activos: {activePatients.length}</span>
          <span className="badge neutral">Encounters visibles: {encounters.length}</span>
          <span className={`badge ${canAccessClinical ? 'info' : 'neutral'}`}>
            {canAccessClinical ? 'Modo clínico habilitado' : 'Modo operativo sin clinical encounters'}
          </span>
          {successMessage ? <span className="badge success">{successMessage}</span> : null}
          {patientsError ? <span className="badge error">{patientsError}</span> : null}
        </div>
        <div className="stack" role="group" aria-label="Buscador de pacientes">
          <div className="section-header">
            <div
              ref={searchControlRef}
              className="field patient-search-combobox"
              style={{ flex: 1 }}
              onFocus={() => setIsSearchOpen(true)}
              onBlur={handleSearchBlur}
            >
              <label htmlFor="patients-sidebar-search">Buscar paciente</label>
              <input
                id="patients-sidebar-search"
                type="search"
                value={sidebarQuery}
                onChange={(event) => {
                  setSidebarQuery(event.target.value);
                  setIsSearchOpen(true);
                }}
                placeholder="Nombre o documento"
                disabled={isBootstrapping || activePatients.length === 0}
                autoComplete="off"
                aria-controls="patients-search-results"
                aria-autocomplete="list"
                aria-expanded={shouldShowSearchDropdown}
                aria-haspopup="listbox"
              />

              {shouldShowSearchDropdown ? (
                <div className="patient-search-dropdown" aria-live="polite">
                  {filteredPatients.length === 0 ? (
                    <div id="patients-search-results" className="empty-state empty-state-soft patient-search-feedback" role="status">
                      <strong>No hay pacientes que coincidan con “{sidebarQuery.trim()}”.</strong>
                      <span>Limpiá el buscador para ver la lista completa.</span>
                    </div>
                  ) : (
                    <>
                      <div className="patient-search-feedback inline-note">
                        <strong>
                          {hasActiveSearchQuery
                            ? `${filteredPatients.length} coincidencia${filteredPatients.length === 1 ? '' : 's'} visible${filteredPatients.length === 1 ? '' : 's'}`
                            : `Pacientes activos disponibles: ${filteredPatients.length}`}
                        </strong>
                      </div>

                      <ul id="patients-search-results" className="list compact-list patient-selector-list patient-search-results" role="listbox" aria-label="Resultados de búsqueda de pacientes">
                        {filteredPatients.map((patient) => {
                          const isSelected = patient.id === selectedPatientId;

                          return (
                            <li key={patient.id}>
                              <button
                                className={`patient-selector-card${isSelected ? ' selected' : ''}`}
                                type="button"
                                role="option"
                                aria-selected={isSelected}
                                onMouseDown={(event) => event.preventDefault()}
                                onClick={() => handlePatientSelection(patient.id)}
                              >
                                <strong>
                                  {patient.first_name} {patient.last_name}
                                </strong>
                                <small>Documento {patient.document}</small>
                              </button>
                            </li>
                          );
                        })}
                      </ul>
                    </>
                  )}
                </div>
              ) : null}
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
              {isSecretaryOperational && onOpenDirectorySupport ? (
                <button className="button secondary" type="button" onClick={onOpenDirectorySupport}>
                  Abrir soporte de directorio
                </button>
              ) : null}
            </div>
          ) : (
            <>
              {isSelectedPatientHiddenByFilter ? (
                <div className="inline-note" aria-live="polite">
                  <strong>Tenés seleccionado a {selectedPatient?.first_name} {selectedPatient?.last_name}; el filtro actual lo ocultó de la lista.</strong>
                </div>
              ) : null}

              {!shouldShowSearchDropdown && hasActiveSearchQuery && filteredPatients.length === 0 ? (
                <div className="inline-note" aria-live="polite">
                  <strong>No hay resultados visibles para el filtro actual.</strong>
                </div>
              ) : null}
            </>
          )}
        </div>
      </SectionCard>

      <ContextualHeader
        eyebrow="Contexto del paciente"
        tone={canAccessClinical ? 'clinical' : 'operational'}
        title={patientContextTitle}
        description={patientContextDescription}
        meta={(
          <>
            <span className="badge neutral">{selectedPatient ? `Documento ${selectedPatient.document}` : 'Sin ficha clínica activa'}</span>
            <span className={`badge ${canAccessClinical && selectedPatient ? 'info' : 'neutral'}`}>
              {canAccessClinical && selectedPatient ? 'Ficha clínica editable' : 'Ficha clínica no abierta'}
            </span>
            <span className="badge neutral">Evoluciones: {selectedPatient && canAccessClinical ? encounters.length : 0}</span>
            <span className="badge neutral">Notas clínicas: {selectedPatient && canAccessClinical ? patientClinicalNotes.length : 0}</span>
          </>
        )}
        feedback={(successMessage || patientsError || !selectedPatient) ? (
          <>
            {successMessage ? <span className="badge success">{successMessage}</span> : null}
            {patientsError ? <span className="badge error">{patientsError}</span> : null}
            {!selectedPatient ? <span className="helper">El header no implica una ficha clínica activa.</span> : null}
          </>
        ) : undefined}
      />

      <div className="stack patients-main">
          <SectionCard className="stack patients-section patients-section-summary" aria-label="Resumen del paciente">
            <div className="section-header">
              <div>
                <h3>Resumen del paciente</h3>
                <p>{canAccessClinical ? 'Ficha básica para contexto rápido antes de una tarea clínica.' : 'Ficha básica para verificar datos sin abrir trabajo clínico.'}</p>
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
          </SectionCard>

          <SectionCard className="stack patients-section patients-section-ficha" aria-label="Ficha clínica editable">
            <div className="section-header">
              <div>
                <h3>Ficha clínica</h3>
                <p>Estado editable del paciente. Separada de notas, consultas, evoluciones, recetas y estudios.</p>
              </div>
              {canAccessClinical && selectedPatientId ? (
                <button className="button secondary" type="button" onClick={() => void loadClinicalHistory(selectedPatientId)} disabled={isLoadingHistory}>
                  {isLoadingHistory ? 'Actualizando...' : 'Recargar ficha'}
                </button>
              ) : null}
            </div>

            {historyError ? <div className="inline-note inline-note-error">{historyError}</div> : null}

            {!selectedPatient ? (
              <div className="empty-state empty-state-soft">Elegí un paciente para ver su ficha clínica.</div>
            ) : !canAccessClinical ? (
              <div className="empty-state">
                <strong>Ficha clínica bloqueada</strong>
                <span>{clinicalDeniedMessage}</span>
              </div>
            ) : isLoadingHistory && !clinicalHistory ? (
              <div className="empty-state empty-state-soft">Cargando ficha clínica...</div>
            ) : (
              <>
                <div className="form-grid">
                  <div className="field">
                    <label htmlFor="history-weight">Peso (kg)</label>
                    <input id="history-weight" inputMode="decimal" value={historyForm.weight_kg} onChange={(event) => setHistoryForm((current) => ({ ...current, weight_kg: event.target.value }))} />
                  </div>
                  <div className="field">
                    <label htmlFor="history-height">Altura (cm)</label>
                    <input id="history-height" inputMode="decimal" value={historyForm.height_cm} onChange={(event) => setHistoryForm((current) => ({ ...current, height_cm: event.target.value }))} />
                  </div>
                  <div className="field form-grid-span-full">
                    <label htmlFor="history-antecedentes">Antecedentes</label>
                    <textarea id="history-antecedentes" value={historyForm.antecedentes} onChange={(event) => setHistoryForm((current) => ({ ...current, antecedentes: event.target.value }))} />
                  </div>
                  <div className="field form-grid-span-full">
                    <label htmlFor="history-allergies">Alergias</label>
                    <textarea id="history-allergies" value={historyForm.allergies} onChange={(event) => setHistoryForm((current) => ({ ...current, allergies: event.target.value }))} />
                  </div>
                  <div className="field form-grid-span-full">
                    <label htmlFor="history-medication">Medicación habitual</label>
                    <textarea id="history-medication" value={historyForm.habitual_medication} onChange={(event) => setHistoryForm((current) => ({ ...current, habitual_medication: event.target.value }))} />
                  </div>
                  <div className="field form-grid-span-full">
                    <label htmlFor="history-conditions">Condiciones crónicas</label>
                    <textarea id="history-conditions" value={historyForm.chronic_conditions} onChange={(event) => setHistoryForm((current) => ({ ...current, chronic_conditions: event.target.value }))} />
                  </div>
                  <div className="field form-grid-span-full">
                    <label htmlFor="history-habits">Hábitos</label>
                    <textarea id="history-habits" value={historyForm.habits} onChange={(event) => setHistoryForm((current) => ({ ...current, habits: event.target.value }))} />
                  </div>
                  <div className="field form-grid-span-full">
                    <label htmlFor="history-observations">Observaciones generales</label>
                    <textarea id="history-observations" value={historyForm.general_observations} onChange={(event) => setHistoryForm((current) => ({ ...current, general_observations: event.target.value }))} />
                  </div>
                </div>
                <div className="toolbar">
                  <button className="button" type="button" onClick={() => void handleSaveClinicalHistory()} disabled={!selectedPatientId || isSavingHistory}>
                    {isSavingHistory ? 'Guardando...' : 'Guardar ficha'}
                  </button>
                  <span className="helper helper-inline">No actualiza notas ni evoluciones automáticamente.</span>
                </div>
              </>
            )}
          </SectionCard>

          <SectionCard className="stack patients-section patients-section-notes" aria-label="Notas clínicas por paciente">
            <div className="section-header">
              <div>
                <h3>Notas clínicas por paciente</h3>
                <p>Artefactos de nota separados de la ficha; pueden estar vinculados a una consulta o quedar sin vínculo.</p>
              </div>
              {canAccessClinical && selectedPatientId ? (
                <button className="button secondary" type="button" onClick={() => void loadClinicalNotes(selectedPatientId)} disabled={isLoadingClinicalNotes}>
                  {isLoadingClinicalNotes ? 'Actualizando...' : 'Recargar notas'}
                </button>
              ) : null}
            </div>

            {clinicalNotesError ? <div className="inline-note inline-note-error">{clinicalNotesError}</div> : null}

            {!selectedPatient ? (
              <div className="empty-state empty-state-soft">Elegí un paciente para ver sus notas clínicas.</div>
            ) : !canAccessClinical ? (
              <div className="empty-state">
                <strong>Notas clínicas bloqueadas</strong>
                <span>{clinicalDeniedMessage}</span>
              </div>
            ) : (
              <>
                <div className="form-grid">
                  <div className="field form-grid-span-full">
                    <label htmlFor="patient-clinical-note-content">Nueva nota clínica</label>
                    <textarea id="patient-clinical-note-content" value={clinicalNoteForm.content} onChange={(event) => setClinicalNoteForm((current) => ({ ...current, content: event.target.value }))} />
                  </div>
                  <div className="field form-grid-span-full">
                    <label htmlFor="patient-clinical-note-consultation">Consulta asociada (UUID opcional)</label>
                    <input id="patient-clinical-note-consultation" value={clinicalNoteForm.consultation_id} onChange={(event) => setClinicalNoteForm((current) => ({ ...current, consultation_id: event.target.value }))} />
                  </div>
                </div>

                {clinicalNoteFormError ? <div className="inline-note inline-note-error">{clinicalNoteFormError}</div> : null}

                <div className="toolbar">
                  <button className="button" type="button" onClick={() => void handleCreateClinicalNote()} disabled={!selectedPatientId || isSavingClinicalNote}>
                    {isSavingClinicalNote ? 'Guardando...' : 'Guardar nota clínica'}
                  </button>
                </div>

                {isLoadingClinicalNotes ? (
                  <div className="empty-state empty-state-soft">Cargando notas clínicas...</div>
                ) : patientClinicalNotes.length === 0 ? (
                  <div className="empty-state empty-state-soft">Sin notas clínicas por paciente todavía.</div>
                ) : (
                  <div className="list">
                    {patientClinicalNotes.map((note) => (
                      <article key={note.id} className="encounter-card">
                        {editingClinicalNoteId === note.id ? (
                          <>
                            <div className="field">
                              <label htmlFor={`edit-clinical-note-${note.id}`}>Editar contenido de nota clínica</label>
                              <textarea id={`edit-clinical-note-${note.id}`} value={editingClinicalNoteForm.content} onChange={(event) => setEditingClinicalNoteForm((current) => ({ ...current, content: event.target.value }))} />
                            </div>
                            <div className="field">
                              <label htmlFor={`edit-clinical-note-consultation-${note.id}`}>Editar consulta asociada</label>
                              <input id={`edit-clinical-note-consultation-${note.id}`} value={editingClinicalNoteForm.consultation_id} onChange={(event) => setEditingClinicalNoteForm((current) => ({ ...current, consultation_id: event.target.value }))} />
                            </div>
                            <div className="toolbar">
                              <button className="button" type="button" onClick={() => void handleUpdateClinicalNote(note.id)}>Guardar cambios de nota clínica</button>
                              <button className="button secondary" type="button" onClick={() => setEditingClinicalNoteId('')}>Cancelar edición</button>
                            </div>
                          </>
                        ) : (
                          <>
                            <div className="section-header">
                              <div>
                                <span className="surface-tab-eyebrow">Nota clínica</span>
                                <h4>{formatDateTime(note.created_at)}</h4>
                              </div>
                              <span className="badge neutral">{note.consultation_id ? 'Vinculada a consulta' : 'Independiente'}</span>
                            </div>
                            <p className="encounter-note">{note.content}</p>
                            <div className="appointment-meta">
                              <span className="muted">Profesional: {note.professional_id}</span>
                              <span className="muted">Consulta: {note.consultation_id ?? 'sin vínculo'}</span>
                            </div>
                            <div className="toolbar">
                              <button className="button secondary" type="button" onClick={() => startEditingClinicalNote(note)}>Editar nota clínica</button>
                              <button className="button secondary" type="button" onClick={() => void handleDeleteClinicalNote(note.id)}>Eliminar nota clínica</button>
                            </div>
                          </>
                        )}
                      </article>
                    ))}
                  </div>
                )}
              </>
            )}
          </SectionCard>

          <SectionCard className="stack patients-section patients-section-evolutions" aria-label="Consultas y evoluciones">
            <div className="section-header">
              <div>
                <h3>Consultas y evoluciones</h3>
                <p>Eventos clínicos ordenados por fecha; no forman parte de la ficha editable.</p>
              </div>
              {canAccessClinical ? (
                <button
                  className="button secondary"
                  type="button"
                  onClick={() => (selectedPatientId ? void loadEncounters(selectedPatientId) : undefined)}
                  disabled={!selectedPatientId || isLoadingEncounters}
                >
                  {isLoadingEncounters ? 'Actualizando...' : 'Recargar'}
                </button>
              ) : null}
            </div>

            {encountersError ? <div className="inline-note inline-note-error">{encountersError}</div> : null}

            {!selectedPatient ? (
              <div className="empty-state empty-state-soft">Elegí un paciente para ver su alcance disponible.</div>
            ) : !canAccessClinical ? (
              <div className="empty-state">
                <strong>Encounters clínicos bloqueados</strong>
                <span>{clinicalDeniedMessage}</span>
                {isSecretaryOperational && activePatients.length === 0 && onOpenDirectorySupport ? (
                  <button className="button secondary" type="button" onClick={onOpenDirectorySupport}>
                    Abrir soporte de directorio
                  </button>
                ) : null}
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
          </SectionCard>

          <SectionCard className="stack patients-section patients-section-new-evolution" aria-label="Nueva nota o evolución">
            <div className="section-header">
              <div>
                <h3>Nueva nota / evolución</h3>
                <p>
                  {canAccessClinical
                    ? 'Formulario corto para registrar una observación inicial solo cuando el actor tiene alcance clínico.'
                    : 'Secretaría no recibe formulario clínico en esta línea base.'}
                </p>
              </div>
            </div>

            {canAccessClinical ? (
              <>
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
                    disabled={!selectedPatientId || isCreatingEncounter}
                  >
                    {isCreatingEncounter ? 'Guardando...' : 'Guardar nota'}
                  </button>
                  <span className="helper helper-inline">Sin router, sin wizard, sin historia clínica completa.</span>
                </div>
              </>
            ) : (
              <div className="empty-state">
                <strong>Trabajo clínico oculto para secretaría</strong>
                <span>Esta línea base deja solo lista y resumen de pacientes. La escritura clínica sigue reservada al actor médico.</span>
              </div>
            )}
          </SectionCard>
      </div>
    </PageContainer>
  );
}

function sortEncounters(encounters: Encounter[]) {
  return [...encounters].sort((left, right) => {
    const leftTime = Date.parse(left.occurred_at) || Date.parse(left.created_at) || 0;
    const rightTime = Date.parse(right.occurred_at) || Date.parse(right.created_at) || 0;
    return rightTime - leftTime;
  });
}

function sortPatientClinicalNotes(notes: PatientClinicalNote[]) {
  return [...notes].sort((left, right) => {
    const leftTime = Date.parse(left.created_at) || Date.parse(left.updated_at) || 0;
    const rightTime = Date.parse(right.created_at) || Date.parse(right.updated_at) || 0;
    return rightTime - leftTime;
  });
}

function clinicalHistoryToForm(history: ClinicalHistory): ClinicalHistoryFormState {
  return {
    weight_kg: numberToFormValue(history.weight_kg),
    height_cm: numberToFormValue(history.height_cm),
    antecedentes: history.antecedentes ?? '',
    allergies: history.allergies ?? '',
    habitual_medication: history.habitual_medication ?? '',
    chronic_conditions: history.chronic_conditions ?? '',
    habits: history.habits ?? '',
    general_observations: history.general_observations ?? '',
  };
}

function historyFormToPayload(form: ClinicalHistoryFormState): UpdateClinicalHistoryPayload | null {
  const weight = numberFieldToPayload(form.weight_kg);
  const height = numberFieldToPayload(form.height_cm);

  if (weight === undefined || height === undefined) {
    return null;
  }

  return {
    weight_kg: weight,
    height_cm: height,
    antecedentes: normalizeNullableText(form.antecedentes),
    allergies: normalizeNullableText(form.allergies),
    habitual_medication: normalizeNullableText(form.habitual_medication),
    chronic_conditions: normalizeNullableText(form.chronic_conditions),
    habits: normalizeNullableText(form.habits),
    general_observations: normalizeNullableText(form.general_observations),
  };
}

function numberToFormValue(value: number | null) {
  return value === null ? '' : String(value);
}

function numberFieldToPayload(value: string): number | null | undefined {
  const trimmed = value.trim();

  if (!trimmed) {
    return null;
  }

  const parsed = Number(trimmed);
  return Number.isNaN(parsed) ? undefined : parsed;
}

function normalizeNullableText(value: string) {
  const trimmed = value.trim();
  return trimmed ? trimmed : null;
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
