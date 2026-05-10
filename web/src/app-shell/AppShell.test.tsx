import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AppShell } from './AppShell';
import type { AppShellProps } from './AppShell.types';

describe('AppShell', () => {
  it('renders the active surface, account summary, and feature body inside the shell stage', () => {
    render(<AppShell {...appShellProps()} />);

    expect(screen.getByRole('heading', { name: 'Pacientes' })).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Contenido de Pacientes' })).toContainElement(screen.getByText('clinical workspace'));
    expect(screen.getByLabelText('Sesión activa')).toHaveTextContent('doctor@example.com');
    expect(screen.getByLabelText('Sesión activa')).toHaveTextContent('doctor');
    expect(screen.getByLabelText('Sesión activa')).toHaveTextContent('Expira: Hoy 18:00');
    expect(screen.getByRole('searchbox', { name: 'Buscar paciente por nombre o DNI' })).toHaveAttribute('placeholder', 'Buscar paciente por nombre o DNI...');
  });

  it('preserves accessible navigation and action callbacks', () => {
    const onLogout = vi.fn();
    const onSelectSurface = vi.fn();

    render(<AppShell {...appShellProps({ onLogout, onSelectSurface })} />);

    const nav = screen.getByRole('navigation', { name: 'Áreas disponibles' });
    expect(within(nav).getByRole('button', { name: /Agenda/i })).toHaveAttribute('aria-pressed', 'false');
    expect(within(nav).getByRole('button', { name: /Pacientes/i })).toHaveAttribute('aria-pressed', 'true');

    fireEvent.click(within(nav).getByRole('button', { name: /Agenda/i }));
    fireEvent.click(screen.getByRole('button', { name: 'Cerrar sesión' }));

    expect(onSelectSurface).toHaveBeenCalledWith('agenda');
    expect(onLogout).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('button', { name: 'Crear nuevo turno' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Abrir notificaciones' })).toBeInTheDocument();
  });

  it('falls back to product initials when no logo is configured', () => {
    render(<AppShell {...appShellProps({ header: { ...baseHeader(), productName: 'Clinic Platform' } })} />);

    expect(screen.getByText('CP')).toBeInTheDocument();
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
  });
});

function appShellProps(overrides: Partial<AppShellProps> = {}): AppShellProps {
  return {
    header: baseHeader(),
    account: {
      email: 'doctor@example.com',
      role: 'doctor',
      sessionExpiryLabel: 'Hoy 18:00',
    },
    activeSurface: 'patients',
    sidebar: {
      eyebrow: 'Workspace',
      title: 'Operación clínica',
      description: 'Navigation shell',
    },
    pageIntro: {
      eyebrow: 'Seguimiento clínico',
      title: 'Pacientes',
      description: 'Panel de pacientes',
    },
    body: {
      children: <div>clinical workspace</div>,
    },
    navItems: [
      {
        id: 'agenda',
        label: 'Agenda',
        eyebrow: 'Práctica propia',
        description: 'Turnos de tu consultorio.',
      },
      {
        id: 'patients',
        label: 'Pacientes',
        eyebrow: 'Seguimiento clínico',
        description: 'Panel de tus pacientes.',
      },
    ],
    onLogout: vi.fn(),
    onSelectSurface: vi.fn(),
    ...overrides,
  };
}

function baseHeader(): AppShellProps['header'] {
  return {
    productName: 'Amicus',
    workspaceName: 'Centro operativo clínico',
    workspaceDescription: 'Gestión clínica diaria',
  };
}
