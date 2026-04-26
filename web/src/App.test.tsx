import { fireEvent, render, screen, within } from '@testing-library/react';
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
  ScheduleDemo: ({
    agendaMode,
    onOpenDirectorySupport,
  }: {
    agendaMode: { kind: string };
    onOpenDirectorySupport?: () => void;
  }) => (
    <div>
      <div>schedule:{agendaMode.kind}</div>
      {onOpenDirectorySupport ? <button onClick={onOpenDirectorySupport}>schedule-open-directory</button> : null}
    </div>
  ),
}));

vi.mock('./features/schedule/WeeklyScheduleWorkspace', () => ({
  WeeklyScheduleWorkspace: ({ agendaMode }: { agendaMode: { kind: string } }) => <div>weekly-schedule:{agendaMode.kind}</div>,
}));

vi.mock('./features/patients/PatientsWorkspace', () => ({
  PatientsWorkspace: ({
    patientsMode,
    onOpenDirectorySupport,
  }: {
    patientsMode: { kind: string };
    onOpenDirectorySupport?: () => void;
  }) => (
    <div>
      <div>patients:{patientsMode.kind}</div>
      {onOpenDirectorySupport ? <button onClick={onOpenDirectorySupport}>patients-open-directory</button> : null}
    </div>
  ),
}));

vi.mock('./features/directory/DirectoryDemo', () => ({
  DirectoryDemo: ({ directoryMode }: { directoryMode: { kind: string } }) => <div>directory:{directoryMode.kind}</div>,
}));

describe('App actor-aware shell', () => {
  beforeEach(() => {
    useAuthSessionMock.mockReset();
  });

  it('shows agenda, weekly schedule and patients for doctors, defaulting to agenda', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'doctor', professional_id: 'professional-1' }));

    const { container } = render(<App />);

    expect(screen.getAllByText('Centro operativo clínico').length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('Amicus')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Agenda' })).toBeInTheDocument();
    const doctorNavButtons = screen.getAllByRole('button', { name: /Agenda|Agenda semanal|Pacientes/ });
    expect(doctorNavButtons).toHaveLength(3);
    expect(doctorNavButtons[0]).toHaveTextContent('Práctica propia');
    expect(doctorNavButtons[0]).toHaveTextContent('Agenda');
    expect(doctorNavButtons[1]).toHaveTextContent('Plantilla semanal');
    expect(doctorNavButtons[1]).toHaveTextContent('Agenda semanal');
    expect(doctorNavButtons[2]).toHaveTextContent('Seguimiento clínico');
    expect(doctorNavButtons[2]).toHaveTextContent('Pacientes');
    expect(doctorNavButtons[0]).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('schedule:doctor-own')).toBeInTheDocument();

    const shellPage = container.querySelector('.page-app-shell');
    expect(shellPage?.firstElementChild).toHaveClass('app-shell-layout');
    expect(container.querySelector('.app-shell-frame')).not.toBeInTheDocument();
    expect(container.querySelector('.app-shell-column > .app-shell-topbar')).toBeInTheDocument();
    expect(container.querySelector('.app-shell-column > .app-shell-page-intro')).toBeInTheDocument();
    expect(container.querySelector('.app-shell-column > .app-shell-stage')).toBeInTheDocument();
    expect(container.querySelector('.app-shell-brand-image')).not.toBeInTheDocument();
    expect(container.querySelector('.app-shell-brand-mark-fallback')).toHaveTextContent('A');
  });

  it('shows agenda, weekly schedule and patients for secretaries without directory shell symmetry', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'secretary' }));

    render(<App />);

    const secretaryNavButtons = screen.getAllByRole('button', { name: /Agenda|Agenda semanal|Pacientes/ });
    expect(secretaryNavButtons).toHaveLength(3);
    expect(secretaryNavButtons[0]).toHaveTextContent('Operación diaria');
    expect(secretaryNavButtons[0]).toHaveTextContent('Agenda');
    expect(secretaryNavButtons[1]).toHaveTextContent('Configuración visible');
    expect(secretaryNavButtons[1]).toHaveTextContent('Agenda semanal');
    expect(secretaryNavButtons[2]).toHaveTextContent('Atención operativa');
    expect(secretaryNavButtons[2]).toHaveTextContent('Pacientes');
    expect(screen.getByRole('heading', { name: 'Agenda' })).toBeInTheDocument();
    expect(screen.getByText('schedule:operational-shared')).toBeInTheDocument();
    expect(screen.queryByText('directory:setup-secretary-support')).not.toBeInTheDocument();
  });

  it('lets doctors reach the weekly schedule from the shell navigation', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'doctor', professional_id: 'professional-1' }));

    render(<App />);

    fireEvent.click(screen.getByRole('button', { name: /Agenda semanal/i }));

    expect(screen.getByRole('heading', { name: 'Agenda semanal' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Agenda semanal/i })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('weekly-schedule:doctor-own')).toBeInTheDocument();
    expect(screen.queryByText('schedule:doctor-own')).not.toBeInTheDocument();
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
    expect(screen.queryByRole('navigation', { name: 'Áreas disponibles' })).not.toBeInTheDocument();
  });

  it('keeps the loading surface outside the authenticated shell', () => {
    useAuthSessionMock.mockReturnValue({
      status: 'loading',
      accessToken: null,
      expiresAt: null,
      user: null,
      errorMessage: '',
      isSubmitting: false,
      login: vi.fn(),
      logout: vi.fn(),
    });

    render(<App />);

    expect(screen.getByRole('heading', { name: 'Recuperando tu sesión...' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Agenda' })).not.toBeInTheDocument();
    expect(screen.queryByRole('navigation', { name: 'Áreas disponibles' })).not.toBeInTheDocument();
  });

  it('shows directory as the only default surface for admins', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'admin' }));

    render(<App />);

    expect(screen.getByRole('navigation', { name: 'Áreas disponibles' })).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: /Directorio/ })).toHaveLength(1);
    expect(screen.getAllByRole('button', { name: /Directorio/ })[0]).toHaveTextContent('Puesta a punto');
    expect(screen.getAllByRole('button', { name: /Directorio/ })[0]).toHaveTextContent('Directorio');
    expect(screen.getByRole('heading', { name: 'Directorio' })).toBeInTheDocument();
    expect(screen.getByText('directory:setup-admin')).toBeInTheDocument();
  });

  it('lets secretary reach hidden directory support without adding sidebar symmetry', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'secretary' }));

    render(<App />);

    const nav = screen.getByRole('navigation', { name: 'Áreas disponibles' });

    expect(within(nav).getAllByRole('button')).toHaveLength(3);
    expect(screen.queryByText('directory:setup-secretary-support')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'schedule-open-directory' }));

    expect(screen.getByText('directory:setup-secretary-support')).toBeInTheDocument();
    expect(within(nav).getAllByRole('button')).toHaveLength(3);
    expect(screen.queryAllByRole('button', { name: /Directorio/ })).toHaveLength(0);
  });

  it('keeps authenticated shell navigation unchanged under the polished copy', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'secretary' }));

    render(<App />);

    expect(screen.getByText('schedule:operational-shared')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Pacientes/i }));

    expect(screen.getByRole('button', { name: /Pacientes/i })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('heading', { name: 'Pacientes' })).toBeInTheDocument();
    expect(screen.getByText('patients:secretary-operational')).toBeInTheDocument();
    expect(screen.queryByText('schedule:operational-shared')).not.toBeInTheDocument();
  });

  it('keeps the feature body inside the shell-owned stage when switching surfaces', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'doctor', professional_id: 'professional-1' }));

    render(<App />);

    const agendaStage = screen.getByRole('region', { name: 'Contenido de Agenda' });
    expect(agendaStage).toContainElement(screen.getByText('schedule:doctor-own'));
    expect(agendaStage).not.toContainElement(screen.queryByText('patients:doctor-clinical'));

    fireEvent.click(screen.getByRole('button', { name: /Pacientes/i }));

    const patientsStage = screen.getByRole('region', { name: 'Contenido de Pacientes' });
    expect(patientsStage).toContainElement(screen.getByText('patients:doctor-clinical'));
    expect(patientsStage).not.toContainElement(screen.queryByText('schedule:doctor-own'));
  });

  it('preserves blocked fallback shell behavior for unknown roles', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'auditor' }));

    render(<App />);

    expect(screen.getAllByRole('button', { name: /Agenda/ })).toHaveLength(1);
    expect(screen.getByRole('button', { name: /Agenda/i })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('heading', { name: 'Agenda' })).toBeInTheDocument();
    expect(screen.getByText('schedule:forbidden')).toBeInTheDocument();
  });

  it('keeps the rendered surface corrected to the allowed default when the actor loses the current one', () => {
    useAuthSessionMock.mockReturnValue(authSession({ role: 'doctor', professional_id: 'professional-1' }));

    const { rerender } = render(<App />);

    fireEvent.click(screen.getByRole('button', { name: /Pacientes/i }));
    expect(screen.getByText('patients:doctor-clinical')).toBeInTheDocument();

    useAuthSessionMock.mockReturnValue(authSession({ role: 'admin' }));
    rerender(<App />);

    expect(screen.getAllByRole('button', { name: /Directorio/ })).toHaveLength(1);
    expect(screen.getByRole('button', { name: /Directorio/i })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('directory:setup-admin')).toBeInTheDocument();
    expect(screen.queryByText('patients:doctor-clinical')).not.toBeInTheDocument();
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
