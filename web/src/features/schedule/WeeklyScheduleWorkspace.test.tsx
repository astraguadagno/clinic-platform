import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/http';
import { WeeklyScheduleWorkspace } from './WeeklyScheduleWorkspace';

const {
  listProfessionalsMock,
  getScheduleTemplateMock,
  listScheduleTemplateVersionsMock,
  createScheduleTemplateVersionMock,
} = vi.hoisted(() => ({
  listProfessionalsMock: vi.fn(),
  getScheduleTemplateMock: vi.fn(),
  listScheduleTemplateVersionsMock: vi.fn(),
  createScheduleTemplateVersionMock: vi.fn(),
}));

vi.mock('../../api/directory', () => ({
  listProfessionals: listProfessionalsMock,
}));

vi.mock('../../api/appointments', () => ({
  getScheduleTemplate: getScheduleTemplateMock,
  listScheduleTemplateVersions: listScheduleTemplateVersionsMock,
  createScheduleTemplateVersion: createScheduleTemplateVersionMock,
}));

describe('WeeklyScheduleWorkspace', () => {
  beforeEach(() => {
    listProfessionalsMock.mockReset();
    getScheduleTemplateMock.mockReset();
    listScheduleTemplateVersionsMock.mockReset();
    createScheduleTemplateVersionMock.mockReset();
  });

  it('shows a dedicated weekly schedule surface for secretaries with professional selection', async () => {
    listProfessionalsMock.mockResolvedValue({
      items: [
        activeProfessional(),
        secondProfessional(),
      ],
    });
    getScheduleTemplateMock.mockResolvedValue({ id: 'template-1', professional_id: 'professional-1', created_at: '2026-04-01T00:00:00Z', updated_at: '2026-04-01T00:00:00Z', versions: [] });
    listScheduleTemplateVersionsMock.mockResolvedValue({ items: [] });

    render(<WeeklyScheduleWorkspace agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

    await waitFor(() => {
      expect(listProfessionalsMock).toHaveBeenCalledTimes(1);
      expect(getScheduleTemplateMock).toHaveBeenCalledWith({
        professional_id: 'professional-1',
        effective_date: '2099-12-31',
      });
    });

    expect(screen.getByText(/guardá una nueva versión que impacte desde la fecha elegida/i)).toBeInTheDocument();
  });

  it('keeps doctors scoped to their own weekly surface without cross-professional selection', async () => {
    listProfessionalsMock.mockResolvedValue({
      items: [
        activeProfessional(),
        secondProfessional(),
      ],
    });
    getScheduleTemplateMock.mockResolvedValue({ id: 'template-1', professional_id: 'professional-1', created_at: '2026-04-01T00:00:00Z', updated_at: '2026-04-01T00:00:00Z', versions: [] });
    listScheduleTemplateVersionsMock.mockResolvedValue({ items: [] });

    render(
      <WeeklyScheduleWorkspace
        agendaMode={{ kind: 'doctor-own', professionalId: 'professional-1' }}
        onSessionInvalid={vi.fn()}
      />,
    );

    expect(await screen.findByText('Ana Médica')).toBeInTheDocument();
    expect(screen.queryByRole('combobox', { name: 'Profesional' })).not.toBeInTheDocument();
    expect(screen.getByText(/solo podés preparar versiones sobre tu agenda profesional/i)).toBeInTheDocument();
  });

  it('invalidates the session on 401 failures while bootstrapping the weekly surface', async () => {
    listProfessionalsMock.mockRejectedValue(new ApiError('Sesión vencida', 401));
    const onSessionInvalid = vi.fn();

    render(<WeeklyScheduleWorkspace agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={onSessionInvalid} />);

    await waitFor(() => {
      expect(onSessionInvalid).toHaveBeenCalledTimes(1);
    });

    expect(screen.queryByText(/Acceso denegado:/i)).not.toBeInTheDocument();
  });

  it('loads current and future versions from backend contracts and refreshes them after save', async () => {
    listProfessionalsMock.mockResolvedValue({
      items: [activeProfessional()],
    });
    getScheduleTemplateMock.mockResolvedValue({
      id: 'template-1',
      professional_id: 'professional-1',
      created_at: '2026-04-01T00:00:00Z',
      updated_at: '2026-04-01T00:00:00Z',
      versions: [],
    });
    listScheduleTemplateVersionsMock
      .mockResolvedValueOnce({
        items: [futureVersion(), activeVersion()],
      })
      .mockResolvedValueOnce({
        items: [savedFutureVersion(), futureVersion(), activeVersion()],
      });
    createScheduleTemplateVersionMock.mockResolvedValue({
      id: 'template-1',
      professional_id: 'professional-1',
      created_at: '2026-04-01T00:00:00Z',
      updated_at: '2026-05-21T00:00:00Z',
      versions: [savedFutureVersion()],
    });

    render(<WeeklyScheduleWorkspace agendaMode={{ kind: 'operational-shared' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByText('Versión 3')).toBeInTheDocument();
    expect(screen.getByText('Versión 4')).toBeInTheDocument();

    const saveButton = screen.getByRole('button', { name: /guardar versión semanal/i });
    fireEvent.change(screen.getByLabelText('Vigencia desde'), { target: { value: '2026-06-01' } });
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(createScheduleTemplateVersionMock).toHaveBeenCalledWith(
        expect.objectContaining({
          professional_id: 'professional-1',
          effective_from: '2026-06-01',
        }),
      );
    });

    expect(await screen.findByText('Versión 5')).toBeInTheDocument();
    expect(screen.getByText(/versión semanal guardada y recargada desde backend/i)).toBeInTheDocument();
  });
});

function activeProfessional() {
  return {
    id: 'professional-1',
    first_name: 'Ana',
    last_name: 'Médica',
    specialty: 'Clínica médica',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function secondProfessional() {
  return {
    id: 'professional-2',
    first_name: 'Tomás',
    last_name: 'Suárez',
    specialty: 'Cardiología',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

function activeVersion() {
  return {
    id: 'version-1',
    template_id: 'template-1',
    version_number: 3,
    effective_from: '2026-04-01T00:00:00Z',
    recurrence: {
      monday: {
        start_time: '09:00',
        end_time: '12:00',
        slot_duration_minutes: 30,
      },
    },
    created_at: '2026-03-25T10:00:00Z',
    created_by: 'secretary-1',
    reason: 'Base operativa vigente',
  };
}

function futureVersion() {
  return {
    id: 'version-2',
    template_id: 'template-1',
    version_number: 4,
    effective_from: '2026-05-15T00:00:00Z',
    recurrence: {
      friday: {
        start_time: '09:00',
        end_time: '12:00',
        slot_duration_minutes: 30,
      },
    },
    created_at: '2026-04-20T09:00:00Z',
    created_by: 'secretary-1',
    reason: 'Cambio por consultorio nuevo',
  };
}

function savedFutureVersion() {
  return {
    id: 'version-3',
    template_id: 'template-1',
    version_number: 5,
    effective_from: '2026-06-01T00:00:00Z',
    recurrence: {
      friday: {
        start_time: '09:00',
        end_time: '12:00',
        slot_duration_minutes: 30,
      },
    },
    created_at: '2026-05-20T09:00:00Z',
    created_by: 'secretary-1',
    reason: 'Versión recargada desde backend',
  };
}
