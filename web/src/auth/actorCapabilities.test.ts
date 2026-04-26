import { describe, expect, it } from 'vitest';
import { deriveActorCapabilities, resolveShellSurfaceMetadata } from './actorCapabilities';

describe('deriveActorCapabilities', () => {
  it('keeps doctor focused on own agenda and patients', () => {
    const capabilities = deriveActorCapabilities(user({ role: 'doctor', professional_id: 'professional-1' }));
    const agendaShell = resolveShellSurfaceMetadata('agenda', capabilities);
    const weeklyScheduleShell = resolveShellSurfaceMetadata('weekly-schedule', capabilities);
    const patientsShell = resolveShellSurfaceMetadata('patients', capabilities);

    expect(capabilities.visibleSurfaces).toEqual(['agenda', 'weekly-schedule', 'patients']);
    expect(capabilities.supportSurfaces).toEqual([]);
    expect(capabilities.defaultSurface).toBe('agenda');
    expect(capabilities.agendaMode).toEqual({ kind: 'doctor-own', professionalId: 'professional-1' });
    expect(capabilities.patientsMode).toEqual({ kind: 'doctor-clinical', professionalId: 'professional-1' });
    expect(capabilities.directoryMode.kind).toBe('forbidden');
    expect(agendaShell.navItem).toEqual({
      id: 'agenda',
      label: 'Agenda',
      eyebrow: 'Práctica propia',
      description: 'Turnos de tu consultorio.',
    });
    expect(weeklyScheduleShell.intro).toEqual({
      eyebrow: 'Plantilla semanal',
      title: 'Agenda semanal',
      description: 'Definí tu esquema semanal y su vigencia sin mezclarlo con la operación diaria.',
    });
    expect(patientsShell.intro).toEqual({
      eyebrow: 'Seguimiento clínico',
      title: 'Pacientes',
      description: 'Resumen y encounters del panel clínico asociado a tu práctica.',
    });
  });

  it('blocks malformed doctor without professional association', () => {
    const capabilities = deriveActorCapabilities(user({ role: 'doctor', professional_id: undefined }));

    expect(capabilities.visibleSurfaces).toEqual(['agenda', 'weekly-schedule', 'patients']);
    expect(capabilities.agendaMode).toEqual({
      kind: 'forbidden',
      message: 'Tu usuario doctor no tiene professional_id asociado.',
    });
    expect(capabilities.patientsMode).toEqual({
      kind: 'forbidden',
      message: 'Tu usuario doctor no tiene professional_id asociado.',
    });
  });

  it('gives secretary operational agenda and patient access without admin symmetry', () => {
    const capabilities = deriveActorCapabilities(user({ role: 'secretary' }));
    const agendaShell = resolveShellSurfaceMetadata('agenda', capabilities);
    const weeklyScheduleShell = resolveShellSurfaceMetadata('weekly-schedule', capabilities);
    const patientsShell = resolveShellSurfaceMetadata('patients', capabilities);
    const directoryShell = resolveShellSurfaceMetadata('directory', capabilities);

    expect(capabilities.visibleSurfaces).toEqual(['agenda', 'weekly-schedule', 'patients']);
    expect(capabilities.supportSurfaces).toEqual(['directory']);
    expect(capabilities.defaultSurface).toBe('agenda');
    expect(capabilities.agendaMode).toEqual({ kind: 'operational-shared' });
    expect(capabilities.patientsMode).toEqual({ kind: 'secretary-operational' });
    expect(capabilities.directoryMode).toEqual({ kind: 'setup-secretary-support' });
    expect(agendaShell.navItem.label).toBe('Agenda');
    expect(weeklyScheduleShell.navItem).toEqual({
      id: 'weekly-schedule',
      label: 'Agenda semanal',
      eyebrow: 'Configuración visible',
      description: 'Template, vigencia y preview futuro.',
    });
    expect(patientsShell.intro).toEqual({
      eyebrow: 'Atención operativa',
      title: 'Pacientes',
      description: 'Selección y verificación de datos sin exponer trabajo clínico.',
    });
    expect(directoryShell.navItem).toEqual({
      id: 'directory',
      label: 'Directorio',
      eyebrow: 'Soporte operativo',
      description: 'Pacientes y consulta de profesionales.',
    });
  });

  it('keeps admin on setup-oriented surfaces only', () => {
    const capabilities = deriveActorCapabilities(user({ role: 'admin' }));
    const directoryShell = resolveShellSurfaceMetadata('directory', capabilities);

    expect(capabilities.visibleSurfaces).toEqual(['directory']);
    expect(capabilities.supportSurfaces).toEqual([]);
    expect(capabilities.defaultSurface).toBe('directory');
    expect(capabilities.directoryMode).toEqual({ kind: 'setup-admin' });
    expect(capabilities.agendaMode.kind).toBe('forbidden');
    expect(capabilities.patientsMode.kind).toBe('forbidden');
    expect(directoryShell.intro).toEqual({
      eyebrow: 'Puesta a punto',
      title: 'Directorio',
      description: 'Base de preparación para dejar agenda y pacientes listos antes de operar.',
    });
  });

  it('falls back to a blocked agenda-only shell for unknown roles', () => {
    const capabilities = deriveActorCapabilities(user({ role: 'auditor' }));

    expect(capabilities.visibleSurfaces).toEqual(['agenda']);
    expect(capabilities.defaultSurface).toBe('agenda');
    expect(capabilities.agendaMode).toEqual({ kind: 'forbidden', message: 'Tu rol no tiene acceso a esta superficie.' });
  });
});

function user(overrides: Partial<ReturnType<typeof baseUser>>) {
  return {
    ...baseUser(),
    ...overrides,
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
