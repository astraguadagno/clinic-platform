import type { AgendaMode } from '../../auth/actorCapabilities';
import type { Professional } from '../../types/directory';
import type { ProfessionalScheduleOption, WeeklyScheduleActorContext } from './weeklyScheduleModels';

type BuildWeeklyScheduleFlowContextInput = {
  agendaMode: AgendaMode;
  professionals: Professional[];
  selectedProfessionalId: string;
};

type WeeklyScheduleFlowContext = {
  actor: WeeklyScheduleActorContext;
  professionalOptions: ProfessionalScheduleOption[];
  initialSelectedProfessionalId?: string;
};

export function buildWeeklyScheduleFlowContext({
  agendaMode,
  professionals,
  selectedProfessionalId,
}: BuildWeeklyScheduleFlowContextInput): WeeklyScheduleFlowContext {
  const professionalOptions = professionals.map((professional) => ({
    id: professional.id,
    label: formatProfessionalName(professional),
    subtitle: professional.specialty,
  }));

  if (agendaMode.kind === 'doctor-own') {
    const selectedProfessional = professionals.find((professional) => professional.id === agendaMode.professionalId);

    return {
      actor: {
        kind: 'doctor',
        actorLabel: selectedProfessional ? formatProfessionalName(selectedProfessional) : 'Agenda propia',
        professionalId: agendaMode.professionalId,
        professionalName: selectedProfessional ? formatProfessionalName(selectedProfessional) : 'Mi agenda profesional',
      },
      professionalOptions,
      initialSelectedProfessionalId: agendaMode.professionalId,
    };
  }

  return {
    actor: {
      kind: 'secretary',
      actorLabel: 'Secretaría central',
      workspaceLabel: 'Agenda operativa multi-profesional',
    },
    professionalOptions,
    initialSelectedProfessionalId: selectedProfessionalId || professionalOptions[0]?.id,
  };
}

function formatProfessionalName(professional: Professional) {
  return `${professional.first_name} ${professional.last_name}`;
}
