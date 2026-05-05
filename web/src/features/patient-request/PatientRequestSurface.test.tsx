import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PatientRequestSurface } from './PatientRequestSurface';

const { createPatientRequestMock, listPublicAvailabilityMock, listPublicProfessionalsMock } = vi.hoisted(() => ({
  createPatientRequestMock: vi.fn(),
  listPublicAvailabilityMock: vi.fn(),
  listPublicProfessionalsMock: vi.fn(),
}));

vi.mock('../../api/appointments', () => ({
  createPatientRequest: createPatientRequestMock,
  listPublicAvailability: listPublicAvailabilityMock,
}));

vi.mock('../../api/directory', () => ({
  listPublicProfessionals: listPublicProfessionalsMock,
}));

describe('PatientRequestSurface', () => {
  beforeEach(() => {
    createPatientRequestMock.mockReset();
    listPublicAvailabilityMock.mockReset();
    listPublicProfessionalsMock.mockReset();
    listPublicProfessionalsMock.mockResolvedValue({
      items: [
        {
          id: 'professional-1',
          first_name: 'Ana',
          last_name: 'Lopez',
          specialty: 'Cardiología',
          active: true,
          created_at: '2026-04-16T09:00:00Z',
          updated_at: '2026-04-16T09:00:00Z',
        },
      ],
    });
    listPublicAvailabilityMock.mockResolvedValue({
      items: [
        {
          professional_id: 'professional-1',
          start_time: '2026-04-27T14:00:00Z',
          end_time: '2026-04-27T14:30:00Z',
        },
        {
          professional_id: 'professional-1',
          start_time: '2026-04-28T15:00:00Z',
          end_time: '2026-04-28T15:30:00Z',
        },
      ],
    });
  });

  it('renders the pre-auth DNI request form without exposing patient profile data', async () => {
    render(<PatientRequestSurface />);

    expect(screen.getByRole('heading', { name: 'Pedí un turno con tu DNI' })).toBeInTheDocument();
    expect(screen.getByLabelText('DNI / documento')).toBeInTheDocument();
    expect(await screen.findByRole('option', { name: 'Ana Lopez — Cardiología' })).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: /lun 27\/04, 14:00/ })).toBeInTheDocument();
    expect(screen.queryByText(/historia clínica/i)).not.toBeInTheDocument();
  });

  it('loads availability, selects a slot and submits a scheduled patient booking', async () => {
    createPatientRequestMock.mockResolvedValue({ id: 'request-1', status: 'scheduled', source: 'patient' });
    render(<PatientRequestSurface />);

    await screen.findByRole('option', { name: 'Ana Lopez — Cardiología' });
    await waitFor(() => {
      expect(listPublicAvailabilityMock).toHaveBeenCalledWith({
        professional_id: 'professional-1',
        week_start: '2026-04-27',
      });
    });
    fireEvent.click(await screen.findByRole('button', { name: /lun 27\/04, 14:00/ }));
    fireEvent.change(screen.getByLabelText('DNI / documento'), { target: { value: '12345678' } });
    fireEvent.change(screen.getByLabelText('Nota opcional'), { target: { value: 'Prefiero tarde' } });
    fireEvent.change(screen.getByLabelText('Contacto opcional'), { target: { value: '11-5555' } });
    fireEvent.click(screen.getByRole('button', { name: 'Reservar horario' }));

    await waitFor(() => {
      expect(createPatientRequestMock).toHaveBeenCalledWith({
        document: '12345678',
        professional_id: 'professional-1',
        notes: 'Prefiero tarde',
        contact: '11-5555',
        scheduled_start: '2026-04-27T14:00:00Z',
        scheduled_end: '2026-04-27T14:30:00Z',
      });
    });
    expect(await screen.findByRole('status')).toHaveTextContent('Turno reservado');
  });

  it('submits a fallback request without selected slot', async () => {
    createPatientRequestMock.mockResolvedValue({ id: 'request-1', status: 'requested', source: 'patient' });
    render(<PatientRequestSurface />);

    await screen.findByRole('button', { name: /lun 27\/04, 14:00/ });
    fireEvent.change(screen.getByLabelText('DNI / documento'), { target: { value: '12345678' } });
    fireEvent.click(screen.getByRole('button', { name: 'Solicitar sin horario' }));

    await waitFor(() => {
      expect(createPatientRequestMock).toHaveBeenCalledWith({
        document: '12345678',
        professional_id: 'professional-1',
        notes: undefined,
        contact: undefined,
      });
    });
    expect(await screen.findByRole('status')).toHaveTextContent('Solicitud recibida');
  });

  it('groups available slots by day without exposing patient data', async () => {
    render(<PatientRequestSurface />);

    const monday = await screen.findByRole('group', { name: 'lunes 27/04' });
    expect(within(monday).getByRole('button', { name: /14:00/ })).toBeInTheDocument();
    const tuesday = screen.getByRole('group', { name: 'martes 28/04' });
    expect(within(tuesday).getByRole('button', { name: /15:00/ })).toBeInTheDocument();
    expect(screen.queryByText(/paciente/i)).not.toBeInTheDocument();
  });

  it('shows an error when the request cannot be created', async () => {
    createPatientRequestMock.mockRejectedValue(new Error('not found'));
    render(<PatientRequestSurface />);

    await screen.findByRole('option', { name: 'Ana Lopez — Cardiología' });
    fireEvent.change(screen.getByLabelText('DNI / documento'), { target: { value: '99999999' } });
    fireEvent.click(screen.getByRole('button', { name: 'Solicitar sin horario' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('No encontramos el DNI');
  });
});
