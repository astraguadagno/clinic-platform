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

export type ShellNavItem = {
  id: SurfaceId;
  label: string;
  eyebrow: string;
  description: string;
};

export type ShellSurfaceIntro = {
  eyebrow: string;
  title: string;
  description: string;
};

export type ShellSurfaceMetadata = {
  navItem: ShellNavItem;
  intro: ShellSurfaceIntro;
};

export type ActorCapabilities = {
  visibleSurfaces: SurfaceId[];
  defaultSurface: SurfaceId;
  agendaMode: AgendaMode;
  patientsMode: PatientsMode;
  directoryMode: DirectoryMode;
};

const DOCTOR_ASSOCIATION_MESSAGE = 'Tu usuario doctor no tiene professional_id asociado.';
const ROLE_ACCESS_MESSAGE = 'Tu rol no tiene acceso a esta superficie.';

const SHELL_SURFACE_COPY: Record<SurfaceId, ShellSurfaceMetadata> = {
  agenda: {
    navItem: {
      id: 'agenda',
      label: 'Agenda',
      eyebrow: 'Operación clínica',
      description: 'Turnos del día.',
    },
    intro: {
      eyebrow: 'Operación clínica',
      title: 'Agenda',
      description: 'Turnos y disponibilidad en una sola vista.',
    },
  },
  patients: {
    navItem: {
      id: 'patients',
      label: 'Pacientes',
      eyebrow: 'Relación asistencial',
      description: 'Seguimiento activo.',
    },
    intro: {
      eyebrow: 'Relación asistencial',
      title: 'Pacientes',
      description: 'Seguimiento clínico del panel activo.',
    },
  },
  directory: {
    navItem: {
      id: 'directory',
      label: 'Directorio',
      eyebrow: 'Base operativa',
      description: 'Base operativa.',
    },
    intro: {
      eyebrow: 'Base operativa',
      title: 'Directorio',
      description: 'Personas y equipos disponibles.',
    },
  },
};

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

export function resolveShellSurfaceMetadata(
  surfaceId: SurfaceId,
  _capabilities: ActorCapabilities,
): ShellSurfaceMetadata {
  return SHELL_SURFACE_COPY[surfaceId];
}
