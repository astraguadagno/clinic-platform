import type { AuthUser } from '../types/auth';

export type SurfaceId = 'agenda' | 'weekly-schedule' | 'directory' | 'patients';

export type AgendaMode =
  | { kind: 'doctor-own'; professionalId: string }
  | { kind: 'operational-shared' }
  | { kind: 'forbidden'; message: string };

export type PatientsMode =
  | { kind: 'doctor-clinical'; professionalId: string }
  | { kind: 'secretary-operational' }
  | { kind: 'forbidden'; message: string };

export type DirectoryMode =
  | { kind: 'setup-admin' }
  | { kind: 'setup-secretary-support' }
  | { kind: 'forbidden'; message: string };

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
  supportSurfaces: SurfaceId[];
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
        visibleSurfaces: ['agenda', 'weekly-schedule', 'patients'],
        supportSurfaces: [],
        defaultSurface: 'agenda',
        agendaMode: { kind: 'forbidden', message: DOCTOR_ASSOCIATION_MESSAGE },
        patientsMode: { kind: 'forbidden', message: DOCTOR_ASSOCIATION_MESSAGE },
        directoryMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
      };
    }

    return {
      visibleSurfaces: ['agenda', 'weekly-schedule', 'patients'],
      supportSurfaces: [],
      defaultSurface: 'agenda',
      agendaMode: { kind: 'doctor-own', professionalId },
      patientsMode: { kind: 'doctor-clinical', professionalId },
      directoryMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
    };
  }

  if (user.role === 'secretary') {
    return {
      visibleSurfaces: ['agenda', 'weekly-schedule', 'patients'],
      supportSurfaces: ['directory'],
      defaultSurface: 'agenda',
      agendaMode: { kind: 'operational-shared' },
      patientsMode: { kind: 'secretary-operational' },
      directoryMode: { kind: 'setup-secretary-support' },
    };
  }

  if (user.role === 'admin') {
    return {
      visibleSurfaces: ['directory'],
      supportSurfaces: [],
      defaultSurface: 'directory',
      agendaMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
      patientsMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
      directoryMode: { kind: 'setup-admin' },
    };
  }

  return {
    visibleSurfaces: ['agenda'],
    supportSurfaces: [],
    defaultSurface: 'agenda',
    agendaMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
    patientsMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
    directoryMode: { kind: 'forbidden', message: ROLE_ACCESS_MESSAGE },
  };
}

export function resolveShellSurfaceMetadata(
  surfaceId: SurfaceId,
  capabilities: ActorCapabilities,
): ShellSurfaceMetadata {
  if (surfaceId === 'agenda') {
    if (capabilities.agendaMode.kind === 'doctor-own') {
      return {
        navItem: {
          id: 'agenda',
          label: 'Agenda',
          eyebrow: 'Práctica propia',
          description: 'Turnos de tu consultorio.',
        },
        intro: {
          eyebrow: 'Práctica propia',
          title: 'Agenda',
          description: 'Turnos y disponibilidad vinculados a tu professional_id.',
        },
      };
    }

    if (capabilities.agendaMode.kind === 'operational-shared') {
      return {
        navItem: {
          id: 'agenda',
          label: 'Agenda',
          eyebrow: 'Operación diaria',
          description: 'Turnos y coordinación.',
        },
        intro: {
          eyebrow: 'Operación diaria',
          title: 'Agenda',
          description: 'Coordinación de turnos con foco operativo y sin trabajo clínico.',
        },
      };
    }

    return {
      navItem: {
        id: 'agenda',
        label: 'Agenda',
        eyebrow: 'Acceso restringido',
        description: 'Superficie bloqueada.',
      },
      intro: {
        eyebrow: 'Acceso restringido',
        title: 'Agenda',
        description: 'Esta superficie queda visible solo para comunicar el bloqueo actual.',
      },
    };
  }

  if (surfaceId === 'patients') {
    if (capabilities.patientsMode.kind === 'doctor-clinical') {
      return {
        navItem: {
          id: 'patients',
          label: 'Pacientes',
          eyebrow: 'Seguimiento clínico',
          description: 'Panel de tus pacientes.',
        },
        intro: {
          eyebrow: 'Seguimiento clínico',
          title: 'Pacientes',
          description: 'Resumen y encounters del panel clínico asociado a tu práctica.',
        },
      };
    }

    if (capabilities.patientsMode.kind === 'secretary-operational') {
      return {
        navItem: {
          id: 'patients',
          label: 'Pacientes',
          eyebrow: 'Atención operativa',
          description: 'Búsqueda y resumen.',
        },
        intro: {
          eyebrow: 'Atención operativa',
          title: 'Pacientes',
          description: 'Selección y verificación de datos sin exponer trabajo clínico.',
        },
      };
    }

    return {
      navItem: {
        id: 'patients',
        label: 'Pacientes',
        eyebrow: 'Acceso restringido',
        description: 'Superficie bloqueada.',
      },
      intro: {
        eyebrow: 'Acceso restringido',
        title: 'Pacientes',
        description: 'Esta superficie queda visible solo para comunicar el bloqueo actual.',
      },
    };
  }

  if (surfaceId === 'weekly-schedule') {
    if (capabilities.agendaMode.kind === 'doctor-own') {
      return {
        navItem: {
          id: 'weekly-schedule',
          label: 'Agenda semanal',
          eyebrow: 'Plantilla semanal',
          description: 'Template, vigencia y preview futuro.',
        },
        intro: {
          eyebrow: 'Plantilla semanal',
          title: 'Agenda semanal',
          description: 'Definí tu esquema semanal y su vigencia sin mezclarlo con la operación diaria.',
        },
      };
    }

    if (capabilities.agendaMode.kind === 'operational-shared') {
      return {
        navItem: {
          id: 'weekly-schedule',
          label: 'Agenda semanal',
          eyebrow: 'Configuración visible',
          description: 'Template, vigencia y preview futuro.',
        },
        intro: {
          eyebrow: 'Configuración visible',
          title: 'Agenda semanal',
          description: 'Editá la plantilla semanal por profesional en una superficie dedicada y explícita.',
        },
      };
    }

    return {
      navItem: {
        id: 'weekly-schedule',
        label: 'Agenda semanal',
        eyebrow: 'Acceso restringido',
        description: 'Superficie bloqueada.',
      },
      intro: {
        eyebrow: 'Acceso restringido',
        title: 'Agenda semanal',
        description: 'Esta superficie queda visible solo para comunicar el bloqueo actual.',
      },
    };
  }

  if (capabilities.directoryMode.kind === 'setup-admin') {
    return {
      navItem: {
        id: 'directory',
        label: 'Directorio',
        eyebrow: 'Puesta a punto',
        description: 'Pacientes y profesionales.',
      },
      intro: {
        eyebrow: 'Puesta a punto',
        title: 'Directorio',
        description: 'Base de preparación para dejar agenda y pacientes listos antes de operar.',
      },
    };
  }

  if (capabilities.directoryMode.kind === 'setup-secretary-support') {
    return {
      navItem: {
        id: 'directory',
        label: 'Directorio',
        eyebrow: 'Soporte operativo',
        description: 'Pacientes y consulta de profesionales.',
      },
      intro: {
        eyebrow: 'Soporte operativo',
        title: 'Directorio',
        description: 'Soporte puntual para destrabar agenda y pacientes sin abrir configuración completa.',
      },
    };
  }

  return {
    navItem: {
      id: 'directory',
      label: 'Directorio',
      eyebrow: 'Acceso restringido',
      description: 'Superficie bloqueada.',
    },
    intro: {
      eyebrow: 'Acceso restringido',
      title: 'Directorio',
      description: 'Esta superficie no forma parte del alcance disponible para tu rol.',
    },
  };
}
