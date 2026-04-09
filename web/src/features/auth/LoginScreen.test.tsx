import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { LoginScreen } from './LoginScreen';

describe('LoginScreen', () => {
  it('renders clinic-product sign-in copy', () => {
    render(<LoginScreen errorMessage="" isSubmitting={false} onLogin={vi.fn()} />);

    expect(screen.getByText('Acceso seguro')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Ingresá a tu espacio de trabajo' })).toBeInTheDocument();
    expect(screen.getByText('Usá tu usuario para entrar a la operación diaria de la clínica.')).toBeInTheDocument();
    expect(screen.getByText('Credenciales disponibles para esta instancia: admin/admin123 · secretary/secretary123 · doctor/doctor123')).toBeInTheDocument();
    expect(screen.getByText('La sesión queda abierta en este navegador')).toBeInTheDocument();
    expect(screen.getByText('Tu acceso define qué espacios vas a ver')).toBeInTheDocument();
  });

  it('keeps the existing submit flow while using the polished labels', async () => {
    const onLogin = vi.fn().mockResolvedValue(undefined);

    render(<LoginScreen errorMessage="" isSubmitting={false} onLogin={onLogin} />);

    fireEvent.click(screen.getByRole('button', { name: 'Ingresar' }));

    expect(onLogin).toHaveBeenCalledWith('admin@clinic.local', 'admin123');
  });

  it('keeps helper guidance secondary to the sign-in action', () => {
    render(<LoginScreen errorMessage="" isSubmitting={false} onLogin={vi.fn()} />);

    expect(screen.getAllByRole('button').map((button) => button.textContent)).toEqual(['Ingresar']);
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
    expect(screen.getByText('Credenciales disponibles para esta instancia: admin/admin123 · secretary/secretary123 · doctor/doctor123')).toBeInTheDocument();
    expect(screen.getByText('La sesión queda abierta en este navegador')).toBeInTheDocument();
  });
});
