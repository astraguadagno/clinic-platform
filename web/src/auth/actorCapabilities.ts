import type { AuthUser } from '../types/auth';

export type SurfaceId = 'agenda' | 'directory' | 'patients';

export type AgendaMode =
  | { kind: 'doctor-own'; professionalId: string }
  | { kind: 'operational-shared' }
  | { kind: 'forbidden'; message: string };

export type PatientsMode =
  | { kind: 'doctor-clinical'; professionalId: string }
  | { kind: 'secretary-operational' }
  | { kind: 'forbidden'; message: string };

export type DirectoryMode = { kind: 'setup-shared' } | { kind: 'forbidden'; message: string };

export type ActorCapabilities = {
  visibleSurfaces: SurfaceId[];
  defaultSurface: SurfaceId;
  agendaMode: AgendaMode;
  patientsMode: PatientsMode;
  directoryMode: DirectoryMode;
};

const DOCTOR_ASSOCIATION_MESSAGE = 'Tu usuario doctor no tiene professional_id asociado.';
const ROLE_ACCESS_MESSAGE = 'Tu rol no tiene acceso a esta superficie.';

export function deriveActorCapabilities(user: AuthUser): ActorCapabilities {
  const professionalId = user.professional_id?.trim() ?? '';

  if (user.role === 'doctor') {
    if (!professionalId) {
      return {
        visibleSurfaces: ['agenda', 'patients'],
        defaultSurface: 'agenda',
        agendaMode: { kind: 'forbidden', message: DOCTOR_ASSOCIATION_MESSAGE },
        patientsMode: { kind: 'forbidden', message: DOCTOR_ASSOCIATION_MESSAGE },
        directoryMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
      };
    }

    return {
      visibleSurfaces: ['agenda', 'patients'],
      defaultSurface: 'agenda',
      agendaMode: { kind: 'doctor-own', professionalId },
      patientsMode: { kind: 'doctor-clinical', professionalId },
      directoryMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
    };
  }

  if (user.role === 'secretary') {
    return {
      visibleSurfaces: ['agenda', 'patients'],
      defaultSurface: 'agenda',
      agendaMode: { kind: 'operational-shared' },
      patientsMode: { kind: 'secretary-operational' },
      directoryMode: { kind: 'setup-shared' },
    };
  }

  if (user.role === 'admin') {
    return {
      visibleSurfaces: ['directory'],
      defaultSurface: 'directory',
      agendaMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
      patientsMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
      directoryMode: { kind: 'setup-shared' },
    };
  }

  return {
    visibleSurfaces: ['agenda'],
    defaultSurface: 'agenda',
    agendaMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
    patientsMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
    directoryMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
  };
}
