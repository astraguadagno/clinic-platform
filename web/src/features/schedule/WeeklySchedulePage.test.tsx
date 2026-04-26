import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { ScheduleTemplateVersion } from '../../types/appointments';
import { WeeklySchedulePage } from './WeeklySchedulePage';

describe('WeeklySchedulePage', () => {
  it('lets secretariat choose the target professional and submit a weekly draft once effective_from is set', () => {
    const onSave = vi.fn();

    render(
      <WeeklySchedulePage
        actor={{ kind: 'secretary', actorLabel: 'Secretaría central', workspaceLabel: 'Agenda operativa multi-profesional' }}
        professionalOptions={[
          { id: 'professional-1', label: 'Dra. Julia Núñez', subtitle: 'Clínica médica' },
          { id: 'professional-2', label: 'Dr. Tomás Suárez', subtitle: 'Cardiología' },
        ]}
        initialSelectedProfessionalId="professional-2"
        currentVersion={activeVersion()}
        futureVersion={futureVersion()}
        knownConflicts={[
          {
            id: 'conflict-1',
            date: '2026-05-06',
            startTime: '10:00',
            endTime: '10:30',
            patientLabel: 'María Gómez',
            summary: 'Hay una consulta futura que quedaría fuera del nuevo template.',
            severity: 'critical',
          },
        ]}
        onSave={onSave}
      />,
    );

    expect(screen.getByRole('combobox', { name: 'Profesional' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /editor semanal/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /preview semanal/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /conflictos futuros/i })).toBeInTheDocument();

    const saveButton = screen.getByRole('button', { name: /guardar versión semanal/i });
    expect(saveButton).toBeDisabled();

    fireEvent.change(screen.getByLabelText('Vigencia desde'), { target: { value: '2026-05-01' } });

    expect(saveButton).toBeEnabled();

    fireEvent.click(saveButton);

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        professionalId: 'professional-2',
        effectiveFrom: '2026-05-01',
      }),
    );
    expect(onSave.mock.calls[0]?.[0].previewDays).toEqual(
      expect.arrayContaining([expect.objectContaining({ dayKey: 'monday', slotCount: 6 })]),
    );
  });

  it('keeps doctors scoped to their own professional agenda without a cross-professional selector', () => {
    render(
      <WeeklySchedulePage
        actor={{
          kind: 'doctor',
          actorLabel: 'Dr. Gregory House',
          professionalId: 'professional-1',
          professionalName: 'Dr. Gregory House',
        }}
        currentVersion={activeVersion()}
      />,
    );

    expect(screen.queryByRole('combobox', { name: 'Profesional' })).not.toBeInTheDocument();
    expect(screen.getByText('Dr. Gregory House')).toBeInTheDocument();
    expect(screen.getByText(/agenda propia/i)).toBeInTheDocument();
    expect(screen.getByText(/solo podés preparar versiones sobre tu agenda profesional/i)).toBeInTheDocument();
  });

  it('re-seeds the draft when the loaded future version changes after a save or reload', () => {
    const onSave = vi.fn();
    const { rerender } = render(
      <WeeklySchedulePage
        actor={{ kind: 'secretary', actorLabel: 'Secretaría central', workspaceLabel: 'Agenda operativa multi-profesional' }}
        professionalOptions={[{ id: 'professional-1', label: 'Dra. Julia Núñez', subtitle: 'Clínica médica' }]}
        initialSelectedProfessionalId="professional-1"
        currentVersion={activeVersion()}
        futureVersion={futureVersion()}
        onSave={onSave}
      />,
    );

    rerender(
      <WeeklySchedulePage
        actor={{ kind: 'secretary', actorLabel: 'Secretaría central', workspaceLabel: 'Agenda operativa multi-profesional' }}
        professionalOptions={[{ id: 'professional-1', label: 'Dra. Julia Núñez', subtitle: 'Clínica médica' }]}
        initialSelectedProfessionalId="professional-1"
        currentVersion={activeVersion()}
        futureVersion={reloadedFutureVersion()}
        onSave={onSave}
      />,
    );

    fireEvent.change(screen.getByLabelText('Vigencia desde'), { target: { value: '2026-06-01' } });
    fireEvent.click(screen.getByRole('button', { name: /guardar versión semanal/i }));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        professionalId: 'professional-1',
        effectiveFrom: '2026-06-01',
        recurrence: {
          tuesday: {
            start_time: '08:00',
            end_time: '11:00',
            slot_duration_minutes: 30,
          },
        },
      }),
    );
  });
});

function activeVersion(): ScheduleTemplateVersion {
  return {
    id: 'version-1',
    template_id: 'template-1',
    version_number: 3,
    effective_from: '2026-04-01',
    recurrence: {
      monday: {
        start_time: '09:00',
        end_time: '12:00',
        slot_duration_minutes: 30,
      },
      thursday: {
        start_time: '15:00',
        end_time: '18:00',
        slot_duration_minutes: 30,
      },
    },
    created_at: '2026-03-25T10:00:00Z',
    created_by: 'secretary-1',
    reason: 'Base operativa vigente',
  };
}

function futureVersion(): ScheduleTemplateVersion {
  return {
    id: 'version-2',
    template_id: 'template-1',
    version_number: 4,
    effective_from: '2026-05-15',
    recurrence: {
      monday: {
        start_time: '10:00',
        end_time: '13:00',
        slot_duration_minutes: 30,
      },
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

function reloadedFutureVersion(): ScheduleTemplateVersion {
  return {
    id: 'version-3',
    template_id: 'template-1',
    version_number: 5,
    effective_from: '2026-06-15',
    recurrence: {
      tuesday: {
        start_time: '08:00',
        end_time: '11:00',
        slot_duration_minutes: 30,
      },
    },
    created_at: '2026-05-20T09:00:00Z',
    created_by: 'secretary-1',
    reason: 'Cambio re-cargado desde backend',
  };
}
