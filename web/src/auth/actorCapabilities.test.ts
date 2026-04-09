import { describe, expect, it } from 'vitest';
import { deriveActorCapabilities } from './actorCapabilities';

describe('deriveActorCapabilities', () => {
  it('keeps doctor focused on own agenda and patients', () => {
    const capabilities = deriveActorCapabilities(user({ role: 'doctor', professional_id: 'professional-1' }));

    expect(capabilities.visibleSurfaces).toEqual(['agenda', 'patients']);
    expect(capabilities.defaultSurface).toBe('agenda');
    expect(capabilities.agendaMode).toEqual({ kind: 'doctor-own', professionalId: 'professional-1' });
    expect(capabilities.patientsMode).toEqual({ kind: 'doctor-clinical', professionalId: 'professional-1' });
    expect(capabilities.directoryMode.kind).toBe('forbidden');
  });

  it('blocks malformed doctor without professional association', () => {
    const capabilities = deriveActorCapabilities(user({ role: 'doctor', professional_id: undefined }));

    expect(capabilities.visibleSurfaces).toEqual(['agenda', 'patients']);
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

    expect(capabilities.visibleSurfaces).toEqual(['agenda', 'patients']);
    expect(capabilities.defaultSurface).toBe('agenda');
    expect(capabilities.agendaMode).toEqual({ kind: 'operational-shared' });
    expect(capabilities.patientsMode).toEqual({ kind: 'secretary-operational' });
    expect(capabilities.directoryMode).toEqual({ kind: 'setup-shared' });
  });

  it('keeps admin on setup-oriented surfaces only', () => {
    const capabilities = deriveActorCapabilities(user({ role: 'admin' }));

    expect(capabilities.visibleSurfaces).toEqual(['directory']);
    expect(capabilities.defaultSurface).toBe('directory');
    expect(capabilities.directoryMode).toEqual({ kind: 'setup-shared' });
    expect(capabilities.agendaMode.kind).toBe('forbidden');
    expect(capabilities.patientsMode.kind).toBe('forbidden');
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
