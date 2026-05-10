import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/http';
import { PatientsWorkspace } from './PatientsWorkspace';

const {
  listPatientsMock,
  listPatientEncountersMock,
  createPatientEncounterMock,
  getClinicalHistoryMock,
  updateClinicalHistoryMock,
  listPatientClinicalNotesMock,
  createPatientClinicalNoteMock,
  updatePatientClinicalNoteMock,
  deletePatientClinicalNoteMock,
} = vi.hoisted(() => ({
  listPatientsMock: vi.fn(),
  listPatientEncountersMock: vi.fn(),
  createPatientEncounterMock: vi.fn(),
  getClinicalHistoryMock: vi.fn(),
  updateClinicalHistoryMock: vi.fn(),
  listPatientClinicalNotesMock: vi.fn(),
  createPatientClinicalNoteMock: vi.fn(),
  updatePatientClinicalNoteMock: vi.fn(),
  deletePatientClinicalNoteMock: vi.fn(),
}));

vi.mock('../../api/directory', () => ({
  listPatients: listPatientsMock,
}));

vi.mock('../../api/clinical', () => ({
  listPatientEncounters: listPatientEncountersMock,
  createPatientEncounter: createPatientEncounterMock,
  getClinicalHistory: getClinicalHistoryMock,
  updateClinicalHistory: updateClinicalHistoryMock,
  listPatientClinicalNotes: listPatientClinicalNotesMock,
  createPatientClinicalNote: createPatientClinicalNoteMock,
  updatePatientClinicalNote: updatePatientClinicalNoteMock,
  deletePatientClinicalNote: deletePatientClinicalNoteMock,
}));

describe('PatientsWorkspace', () => {
  beforeEach(() => {
    listPatientsMock.mockReset();
    listPatientEncountersMock.mockReset();
    createPatientEncounterMock.mockReset();
    getClinicalHistoryMock.mockReset();
    updateClinicalHistoryMock.mockReset();
    listPatientClinicalNotesMock.mockReset();
    createPatientClinicalNoteMock.mockReset();
    updatePatientClinicalNoteMock.mockReset();
    deletePatientClinicalNoteMock.mockReset();
    getClinicalHistoryMock.mockResolvedValue(clinicalHistory());
    listPatientClinicalNotesMock.mockResolvedValue({ items: [] });
  });

  it('promotes the selected patient search and note action into the contextual workspace header', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [encounter()] });
    listPatientClinicalNotesMock.mockResolvedValue({ items: [patientClinicalNote()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    const contextHeader = await screen.findByRole('region', { name: /contexto del paciente/i });
    const searchControl = within(contextHeader).getByRole('group', { name: /buscador de pacientes/i });

    expect(within(searchControl).getByRole('searchbox', { name: /buscar paciente/i })).toBeInTheDocument();
    expect(within(contextHeader).getByRole('button', { name: /^\+ Nueva nota$/i })).toHaveAttribute('title', 'Nueva nota');
    expect(within(contextHeader).getByRole('heading', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(within(contextHeader).getByText('Documento')).toBeInTheDocument();
    expect(within(contextHeader).getByText('12345678')).toBeInTheDocument();
    expect(within(contextHeader).getByText('Ficha')).toBeInTheDocument();
    expect(within(contextHeader).getByText('Editable')).toBeInTheDocument();
    const evolutionsFact = (await within(contextHeader).findByText('Evoluciones')).parentElement;
    const notesFact = within(contextHeader).getByText('Notas clínicas').parentElement;
    await waitFor(() => expect(evolutionsFact).toHaveTextContent('1'));
    expect(notesFact).toHaveTextContent('1');
  });

  it('keeps search with patient metadata visible', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [encounter()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByText(/pacientes activos/i)).toBeInTheDocument();
    expect(screen.getByText(/modo clínico habilitado/i)).toBeInTheDocument();
  });

  it('keeps the search dropdown anchored inside the contextual header action area', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient(), secondPatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [] });

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    const contextHeader = await screen.findByRole('region', { name: /contexto del paciente/i });
    const searchControl = within(contextHeader).getByRole('group', { name: /buscador de pacientes/i });
    const searchInput = within(searchControl).getByRole('searchbox', { name: /buscar paciente/i });

    expect(document.querySelector('.patient-search-surface')).not.toBeInTheDocument();
    expect(within(searchControl).queryByRole('listbox', { name: /resultados de búsqueda de pacientes/i })).not.toBeInTheDocument();

    fireEvent.focus(searchInput);

    const resultsList = await within(searchControl).findByRole('listbox', { name: /resultados de búsqueda de pacientes/i });
    expect(resultsList).toBeInTheDocument();
    expect(within(resultsList).getByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(within(resultsList).getByRole('option', { name: /María Gómez/i })).toBeInTheDocument();
  });

  it('renders patient context facts as compact metadata instead of loud status bubbles', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [encounter()] });
    listPatientClinicalNotesMock.mockResolvedValue({ items: [patientClinicalNote()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    const contextHeader = await screen.findByRole('region', { name: /contexto del paciente/i });

    expect(within(contextHeader).getByText('Documento')).toBeInTheDocument();
    expect(within(contextHeader).getByText('12345678')).toBeInTheDocument();
    expect(within(contextHeader).getByText('Ficha')).toBeInTheDocument();
    expect(within(contextHeader).getByText('Editable')).toBeInTheDocument();
    expect(within(contextHeader).queryByText(/ficha clínica editable/i)).not.toBeInTheDocument();
    expect(within(contextHeader).queryByText(/modo clínico habilitado/i)).not.toBeInTheDocument();
  });

  it('renders an empty contextual header without implying an active clinical record', async () => {
    listPatientsMock.mockResolvedValue({ items: [] });

    render(<PatientsWorkspace patientsMode={{ kind: 'secretary-operational' }} onSessionInvalid={vi.fn()} />);

    const contextHeader = await screen.findByRole('region', { name: /contexto del paciente/i });

    expect(within(contextHeader).getByRole('heading', { name: /Seleccioná un paciente/i })).toBeInTheDocument();
    expect(within(contextHeader).getByText(/sin paciente seleccionado/i)).toBeInTheDocument();
    expect(within(contextHeader).queryByText(/ficha clínica editable/i)).not.toBeInTheDocument();
  });

  it('separates ficha clínica, consultation-linked notes, and evolutions as distinct clinical concepts', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [encounter()] });
    listPatientClinicalNotesMock.mockResolvedValue({ items: [patientClinicalNote()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    const fichaSection = await screen.findByRole('region', { name: /ficha del paciente/i });
    const timelineSection = await screen.findByRole('region', { name: /historial clínico/i });

    expect(fichaSection.compareDocumentPosition(timelineSection) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(fichaSection).getByText(/ficha clínica persistente/i)).toBeInTheDocument();
    expect(within(timelineSection).getByText(/Vinculada a consulta/i)).toBeInTheDocument();
    expect(within(timelineSection).queryByText(/consultation-1|professional-1|patient-1/i)).not.toBeInTheDocument();
    expect(within(timelineSection).getByText('Control inicial')).toBeInTheDocument();
  });

  it('composes the selected patient workspace as ficha first and chronological history second', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [encounter()] });
    listPatientClinicalNotesMock.mockResolvedValue({ items: [patientClinicalNote()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    const fichaSection = await screen.findByRole('region', { name: /ficha del paciente/i });
    const timelineSection = await screen.findByRole('region', { name: /historial clínico/i });

    expect(fichaSection.compareDocumentPosition(timelineSection) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.queryByRole('region', { name: /resumen del paciente/i })).not.toBeInTheDocument();
    expect(within(fichaSection).getByText(/Juan Pérez/i)).toBeInTheDocument();
    expect(within(timelineSection).getByText(/vinculada a consulta/i)).toBeInTheDocument();

    const clinicalNote = within(timelineSection).getByText('Control de alergias actualizado').closest('article');
    const evolution = within(timelineSection).getByText('Control inicial').closest('article');

    expect(clinicalNote?.closest('.patients-timeline-list')).toBeInTheDocument();
    expect(evolution?.closest('.patients-timeline-list')).toBeInTheDocument();
  });

  it('shows doctor clinical data when mode allows encounters', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [encounter()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByText(/ficha clínica y evolución histórica/i)).toBeInTheDocument();
    expect(await screen.findByText('Control inicial')).toBeInTheDocument();
    expect(await screen.findByDisplayValue('Penicilina')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^guardar nota$/i })).toBeEnabled();
  });

  it('lets doctors edit the clinical history ficha without losing unsaved input on save errors', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [] });
    updateClinicalHistoryMock.mockRejectedValue(new ApiError('weight_kg debe ser positivo', 400));

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    const allergiesInput = await screen.findByDisplayValue('Penicilina');
    const weightInput = await screen.findByLabelText(/peso/i);

    fireEvent.change(allergiesInput, { target: { value: 'Penicilina y dipirona' } });
    fireEvent.change(weightInput, { target: { value: '-1' } });
    fireEvent.click(screen.getByRole('button', { name: /guardar ficha/i }));

    expect(await screen.findByText('weight_kg debe ser positivo')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Penicilina y dipirona')).toBeInTheDocument();
    expect(screen.getByDisplayValue('-1')).toBeInTheDocument();
    expect(updateClinicalHistoryMock).toHaveBeenCalledWith('patient-1', expect.objectContaining({
      allergies: 'Penicilina y dipirona',
      weight_kg: -1,
    }));
  });

  it('keeps standalone notes visible when only the clinical history section fails', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [] });
    getClinicalHistoryMock.mockRejectedValue(new ApiError('No se pudo cargar ficha', 500));
    listPatientClinicalNotesMock.mockResolvedValue({ items: [patientClinicalNote()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByText('No se pudo cargar ficha')).toBeInTheDocument();
    expect(await screen.findByText('Control de alergias actualizado')).toBeInTheDocument();
    expect(screen.queryByText(/no se pudieron cargar las notas clínicas/i)).not.toBeInTheDocument();
  });

  it('clears previous patient ficha values when the newly selected patient history load fails', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient(), secondPatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [] });
    getClinicalHistoryMock.mockImplementation((patientId: string) => {
      if (patientId === 'patient-2') {
        return Promise.reject(new ApiError('No se pudo cargar ficha de María', 500));
      }

      return Promise.resolve(clinicalHistory());
    });
    listPatientClinicalNotesMock.mockImplementation((patientId: string) => Promise.resolve({
      items: patientId === 'patient-2'
        ? [patientClinicalNote({ patient_id: 'patient-2', content: 'Nota propia de María' })]
        : [],
    }));

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByDisplayValue('Penicilina')).toBeInTheDocument();

    const searchInput = screen.getByLabelText('Buscar paciente');
    fireEvent.focus(searchInput);
    fireEvent.click(await screen.findByRole('option', { name: /María Gómez/i }));

    expect(await screen.findByText('No se pudo cargar ficha de María')).toBeInTheDocument();
    expect(await screen.findByText('Nota propia de María')).toBeInTheDocument();
    expect(screen.queryByDisplayValue('Penicilina')).not.toBeInTheDocument();
    expect(screen.queryByDisplayValue('Losartán')).not.toBeInTheDocument();
    expect(screen.queryByDisplayValue('Control anual')).not.toBeInTheDocument();
  });

  it('ignores late clinical history and notes responses after switching patients', async () => {
    const patientOneHistory = deferred<ReturnType<typeof clinicalHistory>>();
    const patientOneNotes = deferred<{ items: ReturnType<typeof patientClinicalNote>[] }>();

    listPatientsMock.mockResolvedValue({ items: [activePatient(), secondPatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [] });
    getClinicalHistoryMock.mockImplementation((patientId: string) => {
      if (patientId === 'patient-1') {
        return patientOneHistory.promise;
      }

      return Promise.resolve(clinicalHistory({
        id: 'history-2',
        patient_id: 'patient-2',
        allergies: 'Cefalexina',
        habitual_medication: 'Metformina',
        general_observations: 'Seguimiento María',
      }));
    });
    listPatientClinicalNotesMock.mockImplementation((patientId: string) => {
      if (patientId === 'patient-1') {
        return patientOneNotes.promise;
      }

      return Promise.resolve({ items: [patientClinicalNote({ patient_id: 'patient-2', content: 'Nota propia de María' })] });
    });

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    const searchInput = await screen.findByLabelText('Buscar paciente');
    fireEvent.focus(searchInput);
    fireEvent.click(await screen.findByRole('option', { name: /María Gómez/i }));

    expect(await screen.findByDisplayValue('Cefalexina')).toBeInTheDocument();
    expect(await screen.findByText('Nota propia de María')).toBeInTheDocument();

    await act(async () => {
      patientOneHistory.resolve(clinicalHistory({ allergies: 'Penicilina tardía', habitual_medication: 'Losartán tardío', general_observations: 'Control anual tardío' }));
      patientOneNotes.resolve({ items: [patientClinicalNote({ content: 'Nota tardía de Juan' })] });
    });

    expect(screen.getByDisplayValue('Cefalexina')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Metformina')).toBeInTheDocument();
    expect(screen.getByText('Nota propia de María')).toBeInTheDocument();
    expect(screen.queryByDisplayValue('Penicilina tardía')).not.toBeInTheDocument();
    expect(screen.queryByDisplayValue('Losartán tardío')).not.toBeInTheDocument();
    expect(screen.queryByDisplayValue('Control anual tardío')).not.toBeInTheDocument();
    expect(screen.queryByText('Nota tardía de Juan')).not.toBeInTheDocument();
  });

  it('opens note creation in a modal action and lets doctors create, edit, and delete standalone patient notes', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockResolvedValue({ items: [] });
    listPatientClinicalNotesMock.mockResolvedValue({ items: [patientClinicalNote()] });
    createPatientClinicalNoteMock.mockResolvedValue(patientClinicalNote({ id: 'note-2', content: 'Nueva nota suelta', consultation_id: null, created_at: '2026-01-04T10:00:00Z', updated_at: '2026-01-04T10:00:00Z' }));
    updatePatientClinicalNoteMock.mockResolvedValue(patientClinicalNote({ id: 'note-2', content: 'Nota corregida', consultation_id: null, created_at: '2026-01-04T10:00:00Z', updated_at: '2026-01-04T10:05:00Z' }));
    deletePatientClinicalNoteMock.mockResolvedValue({});

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByText('Control de alergias actualizado')).toBeInTheDocument();
    expect(screen.queryByRole('dialog', { name: /nueva nota clínica/i })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /^\+ nueva nota$/i }));

    const noteDialog = await screen.findByRole('dialog', { name: /nueva nota clínica/i });
    fireEvent.change(screen.getByLabelText(/contenido de la nota clínica/i), { target: { value: 'Nueva nota suelta' } });
    fireEvent.change(screen.getByLabelText(/identificador de consulta/i), { target: { value: '   ' } });
    fireEvent.click(within(noteDialog).getByRole('button', { name: /guardar nota clínica/i }));

    await waitFor(() => {
      expect(createPatientClinicalNoteMock).toHaveBeenCalledWith('patient-1', { content: 'Nueva nota suelta', consultation_id: null });
    });
    expect(await screen.findByText('Nueva nota suelta')).toBeInTheDocument();
    expect(screen.queryByRole('dialog', { name: /nueva nota clínica/i })).not.toBeInTheDocument();

    fireEvent.click(screen.getAllByRole('button', { name: /editar nota clínica/i })[0]);
    fireEvent.change(screen.getByLabelText(/editar contenido de nota clínica/i), { target: { value: 'Nota corregida' } });
    fireEvent.click(screen.getByRole('button', { name: /guardar cambios de nota clínica/i }));

    await waitFor(() => {
      expect(updatePatientClinicalNoteMock).toHaveBeenCalledWith('patient-1', 'note-2', { content: 'Nota corregida', consultation_id: null });
    });
    expect(await screen.findByText('Nota corregida')).toBeInTheDocument();

    fireEvent.click(screen.getAllByRole('button', { name: /eliminar nota clínica/i })[0]);

    await waitFor(() => {
      expect(screen.queryByText('Nota corregida')).not.toBeInTheDocument();
    });
    expect(deletePatientClinicalNoteMock).toHaveBeenCalledWith('patient-1', 'note-2');
  });

  it('keeps secretary in operational patient flow while denying encounters', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'secretary-operational' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByText(/datos operativos del paciente/i)).toBeInTheDocument();
    expect(screen.getByText(/evoluciones bloqueadas/i)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /guardar nota/i })).not.toBeInTheDocument();
    expect(screen.getByText(/trabajo clínico oculto para secretaría/i)).toBeInTheDocument();
    expect(listPatientEncountersMock).not.toHaveBeenCalled();
    expect(getClinicalHistoryMock).not.toHaveBeenCalled();
    expect(listPatientClinicalNotesMock).not.toHaveBeenCalled();
  });

  it('offers directory support to secretary when no active patients exist', async () => {
    listPatientsMock.mockResolvedValue({ items: [] });
    const onOpenDirectorySupport = vi.fn();

    render(
      <PatientsWorkspace
        patientsMode={{ kind: 'secretary-operational' }}
        onSessionInvalid={vi.fn()}
        onOpenDirectorySupport={onOpenDirectorySupport}
      />,
    );

    expect(await screen.findByText(/no hay pacientes activos/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /abrir soporte de directorio/i }));

    expect(onOpenDirectorySupport).toHaveBeenCalledTimes(1);
  });

  it('keeps malformed doctor sessions active while showing the forbidden patients state', () => {
    const onSessionInvalid = vi.fn();

    render(
      <PatientsWorkspace
        patientsMode={{ kind: 'forbidden', message: 'Tu usuario doctor no tiene professional_id asociado.' }}
        onSessionInvalid={onSessionInvalid}
      />,
    );

    expect(screen.getByText(/pacientes bloqueado/i)).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /acceso denegado/i })).toBeInTheDocument();
    expect(screen.getByText(/no tiene professional_id asociado/i)).toBeInTheDocument();
    expect(onSessionInvalid).not.toHaveBeenCalled();
    expect(listPatientsMock).not.toHaveBeenCalled();
    expect(listPatientEncountersMock).not.toHaveBeenCalled();
  });

  it('invalidates the session on 401 patient bootstrap failures', async () => {
    listPatientsMock.mockRejectedValue(new ApiError('Sesión vencida', 401));
    const onSessionInvalid = vi.fn();

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={onSessionInvalid} />);

    await waitFor(() => {
      expect(onSessionInvalid).toHaveBeenCalledTimes(1);
    });
  });

  it('keeps the session active on 403 encounter failures', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listPatientEncountersMock.mockRejectedValue(new ApiError('No podés ver encounters.', 403));
    const onSessionInvalid = vi.fn();

    render(<PatientsWorkspace patientsMode={{ kind: 'doctor-clinical', professionalId: 'professional-1' }} onSessionInvalid={onSessionInvalid} />);

    expect(await screen.findByText('No podés ver encounters.')).toBeInTheDocument();
    expect(onSessionInvalid).not.toHaveBeenCalled();
  });

  it('filters the sidebar while retaining the selected patient outside visible results and restores the list when cleared', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient(), secondPatient()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'secretary-operational' }} onSessionInvalid={vi.fn()} />);

    const searchInput = await screen.findByLabelText('Buscar paciente');

    fireEvent.focus(searchInput);

    expect(await screen.findByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: /María Gómez/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('option', { name: /María Gómez/i }));

    expect(screen.getAllByText('María Gómez').length).toBeGreaterThan(0);

    fireEvent.focus(searchInput);

    fireEvent.change(searchInput, { target: { value: 'juan' } });

    expect(screen.getByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: /María Gómez/i })).not.toBeInTheDocument();
    expect(screen.getByText(/Tenés seleccionado a María Gómez; el filtro actual lo ocultó de la lista./i)).toBeInTheDocument();
    expect(screen.getAllByText('María Gómez').length).toBeGreaterThan(0);

    fireEvent.change(searchInput, { target: { value: 'zzzz' } });

    expect(screen.getByText(/No hay pacientes que coincidan con “zzzz”./i)).toBeInTheDocument();
    expect(screen.getByText(/Limpiá el buscador para ver la lista completa./i)).toBeInTheDocument();

    fireEvent.change(searchInput, { target: { value: '' } });

    expect(screen.getByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: /María Gómez/i })).toBeInTheDocument();
  });

  it('renders the patient options inside the top search control instead of a separate results panel', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient(), secondPatient()] });

    render(<PatientsWorkspace patientsMode={{ kind: 'secretary-operational' }} onSessionInvalid={vi.fn()} />);

    const searchControl = await screen.findByRole('group', { name: /buscador de pacientes/i });
    const searchInput = within(searchControl).getByRole('searchbox', { name: /buscar paciente/i });

    expect(within(searchControl).queryByRole('listbox', { name: /resultados de búsqueda de pacientes/i })).not.toBeInTheDocument();

    fireEvent.focus(searchInput);

    const resultsList = await within(searchControl).findByRole('listbox', { name: /resultados de búsqueda de pacientes/i });

    expect(searchInput).toBeInTheDocument();
    expect(resultsList).toBeInTheDocument();
    expect(searchInput).toHaveAttribute('aria-expanded', 'true');
    expect(screen.queryByText(/lista corta para entrar rápido a una atención/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/este cuerpo mantiene foco operativo/i)).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /resultados/i })).not.toBeInTheDocument();
    expect(within(searchControl).getByText(/pacientes activos disponibles: 2/i)).toBeInTheDocument();
    expect(within(resultsList).getByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(within(resultsList).getByRole('option', { name: /María Gómez/i })).toBeInTheDocument();

    fireEvent.click(within(resultsList).getByRole('option', { name: /María Gómez/i }));

    expect(screen.getAllByText('María Gómez').length).toBeGreaterThan(0);
    expect(within(searchControl).queryByRole('listbox', { name: /resultados de búsqueda de pacientes/i })).not.toBeInTheDocument();
    expect(searchInput).toHaveAttribute('aria-expanded', 'false');
  });

  it('falls back to another active patient when the selected one disappears after refresh', async () => {
    listPatientsMock.mockResolvedValueOnce({ items: [activePatient(), secondPatient()] }).mockResolvedValueOnce({
      items: [activePatient(), inactiveSecondPatient()],
    });

    render(<PatientsWorkspace patientsMode={{ kind: 'secretary-operational' }} onSessionInvalid={vi.fn()} />);

    const searchInput = await screen.findByLabelText('Buscar paciente');

    fireEvent.focus(searchInput);
    fireEvent.click(await screen.findByRole('option', { name: /María Gómez/i }));

    expect(screen.getByRole('heading', { name: /María Gómez/i })).toBeInTheDocument();
    expect(screen.getAllByText('María Gómez').length).toBeGreaterThan(0);

    fireEvent.click(screen.getByRole('button', { name: /actualizar/i }));

    await waitFor(() => {
      expect(screen.getAllByText('Juan Pérez').length).toBeGreaterThan(0);
    });

    expect(screen.queryByText('María Gómez')).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /Juan Pérez/i })).toBeInTheDocument();

    fireEvent.focus(searchInput);

    expect(await screen.findByRole('option', { name: /Juan Pérez/i })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: /María Gómez/i })).not.toBeInTheDocument();
  });
});

function activePatient() {
  return {
    id: 'patient-1',
    first_name: 'Juan',
    last_name: 'Pérez',
    document: '12345678',
    birth_date: '1990-01-01',
    phone: '555-1234',
    email: 'juan@example.com',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function secondPatient() {
  return {
    id: 'patient-2',
    first_name: 'María',
    last_name: 'Gómez',
    document: '20999111',
    birth_date: '1991-02-02',
    phone: '555-9876',
    email: 'maria@example.com',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function inactiveSecondPatient() {
  return {
    ...secondPatient(),
    active: false,
  };
}

function encounter() {
  return {
    id: 'encounter-1',
    chart_id: 'chart-1',
    patient_id: 'patient-1',
    professional_id: 'professional-1',
    occurred_at: '2026-01-02T10:00:00Z',
    created_at: '2026-01-02T10:00:00Z',
    updated_at: '2026-01-02T10:00:00Z',
    initial_note: {
      id: 'note-1',
      encounter_id: 'encounter-1',
      chart_id: 'chart-1',
      patient_id: 'patient-1',
      professional_id: 'professional-1',
      kind: 'summary',
      content: 'Control inicial',
      created_at: '2026-01-02T10:00:00Z',
      updated_at: '2026-01-02T10:00:00Z',
    },
  };
}

function clinicalHistory(overrides: Partial<ReturnType<typeof clinicalHistoryBase>> = {}) {
  return {
    ...clinicalHistoryBase(),
    ...overrides,
  };
}

function clinicalHistoryBase() {
  return {
    id: 'history-1',
    patient_id: 'patient-1',
    weight_kg: 72.5,
    height_cm: 178,
    antecedentes: 'HTA familiar',
    allergies: 'Penicilina',
    habitual_medication: 'Losartán',
    chronic_conditions: null,
    habits: 'Camina 3 veces por semana',
    general_observations: 'Control anual',
    created_at: '2026-01-02T10:00:00Z',
    updated_at: '2026-01-02T10:00:00Z',
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });

  return { promise, resolve, reject };
}

function patientClinicalNote(overrides: Partial<ReturnType<typeof patientClinicalNoteBase>> = {}) {
  return {
    ...patientClinicalNoteBase(),
    ...overrides,
  };
}

function patientClinicalNoteBase() {
  return {
    id: 'note-1',
    patient_id: 'patient-1',
    professional_id: 'professional-1',
    consultation_id: 'consultation-1' as string | null,
    kind: 'standalone',
    content: 'Control de alergias actualizado',
    created_at: '2026-01-03T10:00:00Z',
    updated_at: '2026-01-03T10:00:00Z',
  };
}
