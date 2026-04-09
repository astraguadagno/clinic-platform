import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';

const { useAuthSessionMock } = vi.hoisted(() => ({
  useAuthSessionMock: vi.fn(),
}));

vi.mock('./auth/useAuthSession', () => ({
  useAuthSession: useAuthSessionMock,
}));

vi.mock('./features/auth/LoginScreen', () => ({
  LoginScreen: () => <div>login-screen</div>,
}));

vi.mock('./features/schedule/ScheduleDemo', () => ({
  ScheduleDemo: ({ agendaMode }: { agendaMode: { kind: string } }) => <div>schedule:{agendaMode.kind}</div>,
}));

vi.mock('./features/patients/PatientsWorkspace', () => ({
  PatientsWorkspace: ({ patientsMode }: { patientsMode: { kind: string } }) => <div>patients:{patientsMode.kind}</div>,
}));

vi.mock('./features/directory/DirectoryDemo', () => ({
  DirectoryDemo: ({ directoryMode }: { directoryMode: { kind: string } }) => <div>directory:{directoryMode.kind}</div>,
}));

describe('App actor-aware shell', () => {
  beforeEach(() => {
    useAuthSessionMock.mockReset();
  });

  it('shows agenda and patients for doctors, defaulting to agenda', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'doctor', professional_id: 'professional-1' }));

    render(<App />);

    expect(screen.getAllByRole('tab').map((tab) => tab.textContent)).toEqual(
      expect.arrayContaining(['Atención diariaMi agendaVista enfocada en tu agenda profesional.', 'Atención clínicaPacientesResumen clínico mínimo y encounters del paciente.']),
    );
    expect(screen.getAllByRole('tab')).toHaveLength(2);
    expect(screen.getByText('schedule:doctor-own')).toBeInTheDocument();
  });

  it('shows agenda and patients for secretaries without directory shell symmetry', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'secretary' }));

    render(<App />);

    expect(screen.getAllByRole('tab').map((tab) => tab.textContent)).toEqual(
      expect.arrayContaining(['Operación diariaAgendaTurnos, slots y gestión operativa.', 'Flujo operativoPacientesBúsqueda y selección para tareas administrativas y agenda.']),
    );
    expect(screen.getAllByRole('tab')).toHaveLength(2);
    expect(screen.getByText('schedule:operational-shared')).toBeInTheDocument();
  });

  it('keeps the unauthenticated login surface unchanged', () => {
    useAuthSessionMock.mockReturnValue({
      status: 'anonymous',
      accessToken: null,
      expiresAt: null,
      user: null,
      errorMessage: '',
      isSubmitting: false,
      login: vi.fn(),
      logout: vi.fn(),
    });

    render(<App />);

    expect(screen.getByText('login-screen')).toBeInTheDocument();
    expect(screen.queryByRole('tablist')).not.toBeInTheDocument();
  });

  it('shows directory as the only default surface for admins', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'admin' }));

    render(<App />);

    expect(screen.getAllByRole('tab')).toHaveLength(1);
    expect(screen.getAllByRole('tab')[0]?.textContent).toBe('Setup baseDirectorioAlta base de pacientes y profesionales.');
    expect(screen.getByText('directory:setup-shared')).toBeInTheDocument();
  });
});

function authSession(overrides: Partial<ReturnType<typeof baseUser>>) {
  return {
    status: 'authenticated',
    accessToken: 'token',
    expiresAt: '2026-01-01T00:00:00Z',
    user: {
      ...baseUser(),
      ...overrides,
    },
    errorMessage: '',
    isSubmitting: false,
    login: vi.fn(),
    logout: vi.fn(),
  };
}

function baseUser() {
  return {
    id: 'user-1',
    email: 'user@example.com',
    role: 'doctor',
    professional_id: 'professional-1',
    active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}
