import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/http';
import { DirectoryDemo } from './DirectoryDemo';

const { listPatientsMock, listProfessionalsMock, createPatientMock, createProfessionalMock } = vi.hoisted(() => ({
  listPatientsMock: vi.fn(),
  listProfessionalsMock: vi.fn(),
  createPatientMock: vi.fn(),
  createProfessionalMock: vi.fn(),
}));

vi.mock('../../api/directory', () => ({
  listPatients: listPatientsMock,
  listProfessionals: listProfessionalsMock,
  createPatient: createPatientMock,
  createProfessional: createProfessionalMock,
}));

describe('DirectoryDemo', () => {
  beforeEach(() => {
    listPatientsMock.mockReset();
    listProfessionalsMock.mockReset();
    createPatientMock.mockReset();
    createProfessionalMock.mockReset();
  });

  it('renders setup data for allowed actors', async () => {
    listPatientsMock.mockResolvedValue({ items: [activePatient()] });
    listProfessionalsMock.mockResolvedValue({ items: [activeProfessional()] });

    render(<DirectoryDemo directoryMode={{ kind: 'setup-shared' }} onSessionInvalid={vi.fn()} />);

    expect(await screen.findByText(/superficie de configuración liviana/i)).toBeInTheDocument();
    expect(screen.getByText('Juan Pérez')).toBeInTheDocument();
    expect(screen.getByText('Ana Médica')).toBeInTheDocument();
  });

  it('invalidates the session on 401 directory bootstrap failures', async () => {
    listPatientsMock.mockRejectedValue(new ApiError('Sesión vencida', 401));
    listProfessionalsMock.mockResolvedValue({ items: [] });
    const onSessionInvalid = vi.fn();

    render(<DirectoryDemo directoryMode={{ kind: 'setup-shared' }} onSessionInvalid={onSessionInvalid} />);

    await waitFor(() => {
      expect(onSessionInvalid).toHaveBeenCalledTimes(1);
    });
  });

  it('keeps the session active on 403 directory bootstrap failures', async () => {
    listPatientsMock.mockRejectedValue(new ApiError('No podés abrir el directorio.', 403));
    listProfessionalsMock.mockResolvedValue({ items: [] });
    const onSessionInvalid = vi.fn();

    render(<DirectoryDemo directoryMode={{ kind: 'setup-shared' }} onSessionInvalid={onSessionInvalid} />);

    expect(await screen.findByText('Acceso denegado: No podés abrir el directorio.')).toBeInTheDocument();
    expect(onSessionInvalid).not.toHaveBeenCalled();
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

function activeProfessional() {
  return {
    id: 'professional-1',
    first_name: 'Ana',
    last_name: 'Médica',
    specialty: 'Clínica',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}
