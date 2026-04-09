import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { cancelAppointment, createAppointment, createSlotsBulk, listAppointments, listSlots } from '../../api/appointments';
import { listPatients, listProfessionals } from '../../api/directory';
import { type AgendaMode } from '../../auth/actorCapabilities';
import { resolveAuthenticatedViewError } from '../../auth/authenticatedViewPolicy';
import type { Appointment, BulkCreateSlotsPayload, Slot } from '../../types/appointments';
import type { Patient, Professional } from '../../types/directory';
import { formatDateInputValue, formatDateTimeRange, formatLongDate } from './helpers';

type ScheduleDemoProps = {
  agendaMode: AgendaMode;
  onSessionInvalid: () => void;
};

export function ScheduleDemo({ agendaMode, onSessionInvalid }: ScheduleDemoProps) {
  const [professionals, setProfessionals] = useState<Professional[]>([]);
  const [patients, setPatients] = useState<Patient[]>([]);
  const [selectedProfessionalId, setSelectedProfessionalId] = useState('');
  const [selectedPatientId, setSelectedPatientId] = useState('');
  const [selectedDate, setSelectedDate] = useState(formatDateInputValue());
  const [selectedSlotId, setSelectedSlotId] = useState('');
  const [daySlots, setDaySlots] = useState<Slot[]>([]);
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [isBootstrapping, setIsBootstrapping] = useState(true);
  const [isRefreshingAgenda, setIsRefreshingAgenda] = useState(false);
  const [isCreatingSlots, setIsCreatingSlots] = useState(false);
  const [isBooking, setIsBooking] = useState(false);
  const [cancellingAppointmentId, setCancellingAppointmentId] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const [accessDeniedMessage, setAccessDeniedMessage] = useState('');
  const [successMessage, setSuccessMessage] = useState('');
  const [releasedSlotId, setReleasedSlotId] = useState('');
  const [releasedSlotLabel, setReleasedSlotLabel] = useState('');
  const [slotStartTime, setSlotStartTime] = useState('09:00');
  const [slotEndTime, setSlotEndTime] = useState('12:00');
  const [slotDurationMinutes, setSlotDurationMinutes] = useState<BulkCreateSlotsPayload['slot_duration_minutes']>(30);
  const releasedSlotIdRef = useRef('');

  const clearReleasedSlotFeedback = useCallback(() => {
    releasedSlotIdRef.current = '';
    setReleasedSlotId('');
    setReleasedSlotLabel('');
  }, []);

  const isDoctorAgenda = agendaMode.kind === 'doctor-own';

  const handleApiFailure = useCallback(
    (error: unknown, fallbackMessage: string) => {
      const resolution = resolveAuthenticatedViewError(error, onSessionInvalid, fallbackMessage, 'No podés operar esta agenda.');

      if (resolution.kind === 'session-invalid') {
        return { errorMessage: '', accessDeniedMessage: '' };
      }

      if (resolution.kind === 'forbidden') {
        return { errorMessage: '', accessDeniedMessage: resolution.message };
      }

      return { errorMessage: resolution.message, accessDeniedMessage: '' };
    },
    [onSessionInvalid],
  );

  const selectedProfessional = useMemo(
    () => professionals.find((professional) => professional.id === selectedProfessionalId) ?? null,
    [professionals, selectedProfessionalId],
  );

  const selectedPatient = useMemo(
    () => patients.find((patient) => patient.id === selectedPatientId) ?? null,
    [patients, selectedPatientId],
  );

  const availableSlots = useMemo(
    () => daySlots.filter((slot) => slot.status === 'available').sort(compareByStartTime),
    [daySlots],
  );

  const slotById = useMemo(
    () => new Map(daySlots.map((slot) => [slot.id, slot])),
    [daySlots],
  );

  const selectedSlot = useMemo(() => slotById.get(selectedSlotId) ?? null, [selectedSlotId, slotById]);

  const patientNameById = useMemo(
    () => new Map(patients.map((patient) => [patient.id, `${patient.first_name} ${patient.last_name}`])),
    [patients],
  );

  const bookedAppointmentsCount = useMemo(
    () => appointments.filter((appointment) => appointment.status === 'booked').length,
    [appointments],
  );

  const cancelledAppointmentsCount = useMemo(
    () => appointments.filter((appointment) => appointment.status === 'cancelled').length,
    [appointments],
  );

  const bootstrap = useCallback(async () => {
    if (agendaMode.kind === 'forbidden') {
      setProfessionals([]);
      setPatients([]);
      setSelectedProfessionalId('');
      setSelectedPatientId('');
      setAccessDeniedMessage(agendaMode.message);
      setIsBootstrapping(false);
      return;
    }

    try {
      setIsBootstrapping(true);
      setErrorMessage('');
      setAccessDeniedMessage('');

      const [professionalsResponse, patientsResponse] = await Promise.all([listProfessionals(), listPatients()]);

      const nextProfessionals = professionalsResponse.items.filter((professional) => professional.active);
      const nextPatients = patientsResponse.items.filter((patient) => patient.active);

      setProfessionals(nextProfessionals);
      setPatients(nextPatients);
      setSelectedProfessionalId((current) => {
        if (agendaMode.kind === 'doctor-own') {
          return agendaMode.professionalId;
        }

	      return nextProfessionals.some((professional) => professional.id === current) ? current : nextProfessionals[0]?.id || '';
	    });
      setSelectedPatientId((current) =>
        nextPatients.some((patient) => patient.id === current) ? current : nextPatients[0]?.id || '',
      );
    } catch (error) {
		const nextError = handleApiFailure(error, 'No se pudieron cargar profesionales y pacientes.');
		setErrorMessage(nextError.errorMessage);
		setAccessDeniedMessage(nextError.accessDeniedMessage);
    } finally {
      setIsBootstrapping(false);
    }
  }, [agendaMode, handleApiFailure]);

  const refreshAgenda = useCallback(async () => {
    if (agendaMode.kind === 'forbidden') {
      setDaySlots([]);
      setAppointments([]);
      setSelectedSlotId('');
      return;
    }

    if (!selectedProfessionalId || !selectedDate) {
      setDaySlots([]);
      setAppointments([]);
      setSelectedSlotId('');
      return;
    }

    try {
      setIsRefreshingAgenda(true);
      setErrorMessage('');
      setAccessDeniedMessage('');

      const [slotsResponse, appointmentsResponse] = await Promise.all([
        listSlots({ professional_id: selectedProfessionalId, date: selectedDate }),
        listAppointments({ professional_id: selectedProfessionalId, date: selectedDate }),
      ]);

      const nextSlots = [...slotsResponse.items].sort(compareByStartTime);
      const nextAppointments = [...appointmentsResponse.items].sort(compareAppointmentsBySlot(nextSlots));

      setDaySlots(nextSlots);
      setAppointments(nextAppointments);
      setSelectedSlotId((current) =>
        nextSlots.find((slot) => slot.id === releasedSlotIdRef.current && slot.status === 'available')?.id ||
        nextSlots.find((slot) => slot.id === current && slot.status === 'available')?.id ||
        nextSlots.find((slot) => slot.status === 'available')?.id ||
        '',
      );
    } catch (error) {
		const nextError = handleApiFailure(error, 'No se pudo cargar la agenda seleccionada.');
		setErrorMessage(nextError.errorMessage);
		setAccessDeniedMessage(nextError.accessDeniedMessage);
      setDaySlots([]);
      setAppointments([]);
      setSelectedSlotId('');
    } finally {
      setIsRefreshingAgenda(false);
    }
  }, [agendaMode.kind, handleApiFailure, selectedDate, selectedProfessionalId]);

  useEffect(() => {
    if (isBootstrapping) {
      return;
    }

    void refreshAgenda();
  }, [isBootstrapping, refreshAgenda]);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  async function handleBookAppointment() {
    if (!selectedProfessionalId || !selectedPatientId || !selectedSlotId) {
      setErrorMessage('Seleccioná profesional, paciente y slot antes de reservar.');
      return;
    }

    try {
      setIsBooking(true);
      setErrorMessage('');
      setAccessDeniedMessage('');
      setSuccessMessage('');
      clearReleasedSlotFeedback();

      await createAppointment({
        slot_id: selectedSlotId,
        patient_id: selectedPatientId,
        professional_id: selectedProfessionalId,
      });

      setSuccessMessage('Turno reservado correctamente.');
      await refreshAgenda();
    } catch (error) {
		const nextError = handleApiFailure(error, 'No se pudo reservar el turno.');
		setErrorMessage(nextError.errorMessage);
		setAccessDeniedMessage(nextError.accessDeniedMessage);
    } finally {
      setIsBooking(false);
    }
  }

  async function handleCancelAppointment(appointmentId: string) {
    const appointment = appointments.find((item) => item.id === appointmentId);
    const slot = appointment ? slotById.get(appointment.slot_id) : null;

    try {
      setCancellingAppointmentId(appointmentId);
      setErrorMessage('');
      setAccessDeniedMessage('');
      setSuccessMessage('');

      await cancelAppointment(appointmentId);

      releasedSlotIdRef.current = appointment?.slot_id ?? '';
      setReleasedSlotId(appointment?.slot_id ?? '');
      setReleasedSlotLabel(slot ? formatDateTimeRange(slot.start_time, slot.end_time) : 'el horario cancelado');
      setSuccessMessage(
        slot
          ? `Turno cancelado. El slot ${formatDateTimeRange(slot.start_time, slot.end_time)} volvió a quedar disponible.`
          : 'Turno cancelado. El slot volvió a quedar disponible.',
      );
      await refreshAgenda();
    } catch (error) {
		const nextError = handleApiFailure(error, 'No se pudo cancelar el turno.');
		setErrorMessage(nextError.errorMessage);
		setAccessDeniedMessage(nextError.accessDeniedMessage);
    } finally {
      setCancellingAppointmentId('');
    }
  }

  async function handleCreateSlots() {
    if (!selectedProfessionalId || !selectedDate || !slotStartTime || !slotEndTime) {
      setErrorMessage('Seleccioná profesional, fecha y rango horario antes de generar slots.');
      return;
    }

    if (slotStartTime >= slotEndTime) {
      setErrorMessage('El horario de inicio tiene que ser anterior al de fin.');
      return;
    }

    try {
      setIsCreatingSlots(true);
      setErrorMessage('');
      setAccessDeniedMessage('');
      setSuccessMessage('');
      clearReleasedSlotFeedback();

      const response = await createSlotsBulk({
        professional_id: selectedProfessionalId,
        date: selectedDate,
        start_time: slotStartTime,
        end_time: slotEndTime,
        slot_duration_minutes: slotDurationMinutes,
      });

      setSuccessMessage(
        response.items.length > 0
          ? `Se generaron ${response.items.length} slot${response.items.length === 1 ? '' : 's'} correctamente.`
          : 'No se generaron slots nuevos.',
      );
      await refreshAgenda();
    } catch (error) {
		const nextError = handleApiFailure(error, 'No se pudieron generar los slots.');
		setErrorMessage(nextError.errorMessage);
		setAccessDeniedMessage(nextError.accessDeniedMessage);
    } finally {
      setIsCreatingSlots(false);
    }
  }

  if (agendaMode.kind === 'forbidden') {
    return (
      <section className="card stack schedule-demo" aria-live="polite">
        <div className="hero-kicker">Agenda bloqueada</div>
        <h1>Acceso denegado</h1>
        <p>{agendaMode.message}</p>
      </section>
    );
  }

  return (
    <div className="stack schedule-demo">
      <header className="card schedule-hero">
        <div className="schedule-hero-copy stack">
          <div className="hero-kicker">Agenda diaria</div>
          <div className="stack-tight">
            <h1>Agenda de turnos</h1>
            <p>
              Elegí la fecha y el profesional, revisá la disponibilidad generada y reservá o cancelá con el flujo que
              ya existe hoy.
            </p>
          </div>

          <div className="schedule-hero-badges status-bar">
            <span className="badge neutral">Profesionales activos: {professionals.length}</span>
            <span className="badge neutral">Pacientes activos: {patients.length}</span>
            <span className="badge neutral">Slots del día: {daySlots.length}</span>
            <span className="badge info">UTC fijo para evitar corrimientos</span>
          </div>

          {(successMessage || errorMessage || accessDeniedMessage) && (
            <div className="status-bar">
              {successMessage ? <span className="badge success">{successMessage}</span> : null}
              {errorMessage ? <span className="badge error">{errorMessage}</span> : null}
              {accessDeniedMessage ? <span className="badge error">Acceso denegado: {accessDeniedMessage}</span> : null}
            </div>
          )}
        </div>

        <div className="schedule-hero-panel">
            <div className="schedule-hero-panel-head">
            <span className="summary-label">Agenda seleccionada</span>
            <strong>{selectedProfessional ? `${selectedProfessional.first_name} ${selectedProfessional.last_name}` : 'Sin profesional'}</strong>
            <small>{selectedProfessional?.specialty ?? 'Elegí un profesional para revisar la agenda.'}</small>
          </div>

          <div className="schedule-stat-grid">
            <article className="schedule-stat-card">
              <span className="summary-label">Fecha</span>
              <strong>{formatLongDate(selectedDate)}</strong>
              <small>Vista filtrada por día.</small>
            </article>
            <article className="schedule-stat-card">
              <span className="summary-label">Disponibles</span>
              <strong>{availableSlots.length}</strong>
              <small>{availableSlots.length > 0 ? 'Listos para seleccionar.' : 'Todavía sin disponibilidad.'}</small>
            </article>
            <article className="schedule-stat-card">
              <span className="summary-label">Reservados</span>
              <strong>{bookedAppointmentsCount}</strong>
              <small>Turnos activos del día.</small>
            </article>
            <article className="schedule-stat-card">
              <span className="summary-label">Cancelados</span>
              <strong>{cancelledAppointmentsCount}</strong>
              <small>Historial visible en agenda.</small>
            </article>
          </div>
        </div>
      </header>

      <div className="layout schedule-layout">
        <aside className="schedule-sidebar stack">
          <section className="card card-accent schedule-controls-card stack">
            <div>
              <span className="summary-label">Contexto</span>
              <h2>Filtros de agenda</h2>
              <p className="helper">Paso 1: elegí profesional y fecha para cargar la disponibilidad del día.</p>
            </div>

            <div className="field">
              <label htmlFor="professional">Profesional</label>
              {isDoctorAgenda ? (
                <div className="inline-note">
                  <strong>{selectedProfessional ? `${selectedProfessional.first_name} ${selectedProfessional.last_name}` : 'Mi agenda'}</strong>
                  <span>{selectedProfessional?.specialty ?? 'Agenda fija por ownership del doctor.'}</span>
                </div>
              ) : (
                <select
                  id="professional"
                  value={selectedProfessionalId}
                  onChange={(event) => {
                    clearReleasedSlotFeedback();
                    setSelectedProfessionalId(event.target.value);
                  }}
                  disabled={isBootstrapping || professionals.length === 0}
                >
                  {professionals.length === 0 ? <option value="">No hay profesionales</option> : null}
                  {professionals.map((professional) => (
                    <option key={professional.id} value={professional.id}>
                      {professional.first_name} {professional.last_name} · {professional.specialty}
                    </option>
                  ))}
                </select>
              )}
            </div>

            <div className="field">
              <label htmlFor="date">Fecha</label>
              <input
                id="date"
                type="date"
                value={selectedDate}
                onChange={(event) => {
                  clearReleasedSlotFeedback();
                  setSelectedDate(event.target.value);
                }}
              />
              <small className="helper">La fecha se interpreta en UTC, igual que la disponibilidad del backend.</small>
            </div>

            <button
              className="button secondary"
              type="button"
              onClick={() => {
                clearReleasedSlotFeedback();
                void refreshAgenda();
              }}
              disabled={isBootstrapping || isRefreshingAgenda}
            >
              {isRefreshingAgenda ? 'Actualizando...' : 'Actualizar agenda'}
            </button>
          </section>

          <section className="card schedule-generator-card stack" aria-labelledby="generate-slots-title">
            <div className="stack-tight">
              <span className="summary-label schedule-dark-eyebrow">Automatización</span>
              <h2 id="generate-slots-title">Generar slots</h2>
              <p className="helper">Paso 2 opcional: cargá disponibilidad para esa agenda sin salir de esta vista.</p>
            </div>

            <div className="time-grid">
              <div className="field">
                <label htmlFor="slot-start-time">Desde</label>
                <input
                  id="slot-start-time"
                  type="time"
                  value={slotStartTime}
                  onChange={(event) => setSlotStartTime(event.target.value)}
                />
              </div>

              <div className="field">
                <label htmlFor="slot-end-time">Hasta</label>
                <input id="slot-end-time" type="time" value={slotEndTime} onChange={(event) => setSlotEndTime(event.target.value)} />
              </div>
            </div>

            <div className="field">
              <label htmlFor="slot-duration">Duración del slot</label>
              <select
                id="slot-duration"
                value={slotDurationMinutes}
                onChange={(event) => setSlotDurationMinutes(Number(event.target.value) as BulkCreateSlotsPayload['slot_duration_minutes'])}
              >
                {[15, 20, 30, 60].map((duration) => (
                  <option key={duration} value={duration}>
                    {duration} minutos
                  </option>
                ))}
              </select>
            </div>

            <div className="schedule-generator-summary">
              <span>Ventana</span>
              <strong>
                {slotStartTime} → {slotEndTime}
              </strong>
              <small>{slotDurationMinutes} min por slot</small>
            </div>

            <button
              className="button schedule-generator-button"
              type="button"
              onClick={() => void handleCreateSlots()}
              disabled={isBootstrapping || isRefreshingAgenda || isCreatingSlots || !selectedProfessionalId || !selectedDate}
            >
              {isCreatingSlots ? 'Generando...' : 'Generar slots'}
            </button>
          </section>

          <section className="card schedule-booking-card stack">
            <div>
              <span className="summary-label">Reserva</span>
              <h2>Reservar turno</h2>
              <p className="helper">Paso 3: elegí un horario desde la grilla principal y asignalo a un paciente.</p>
            </div>

            <div className="field">
              <label htmlFor="patient">Paciente</label>
              <select
                id="patient"
                value={selectedPatientId}
                onChange={(event) => setSelectedPatientId(event.target.value)}
                disabled={isBootstrapping || patients.length === 0}
              >
                {patients.length === 0 ? <option value="">No hay pacientes</option> : null}
                {patients.map((patient) => (
                  <option key={patient.id} value={patient.id}>
                    {patient.first_name} {patient.last_name} · doc {patient.document}
                  </option>
                ))}
              </select>
            </div>

            <div className="inline-note" aria-live="polite">
              <strong>{selectedSlot ? formatDateTimeRange(selectedSlot.start_time, selectedSlot.end_time) : 'Todavía sin horario seleccionado'}</strong>
              <span>
                {selectedSlot
                  ? `Paciente listo para reservar: ${selectedPatient ? `${selectedPatient.first_name} ${selectedPatient.last_name}` : 'seleccioná un paciente'}.`
                  : 'Seleccioná un horario desde la grilla de disponibilidad para continuar.'}
              </span>
            </div>

            <button
              className="button"
              type="button"
              onClick={() => void handleBookAppointment()}
              disabled={isBootstrapping || isRefreshingAgenda || isBooking || !selectedSlotId || !selectedPatientId}
            >
              {isBooking ? 'Reservando...' : 'Reservar turno'}
            </button>

            <div className="inline-note">
              <strong>Nota:</strong> si cargás un paciente o profesional en Directorio, al volver a Agenda se refresca la base
              disponible sin recargar toda la app.
            </div>
          </section>
        </aside>

        <section className="stack schedule-main">
          <article className="card">
            <div className="section-header">
              <div>
                <h2>Slots disponibles</h2>
                <p>
                  {selectedProfessional
                    ? `${selectedProfessional.first_name} ${selectedProfessional.last_name} · ${formatLongDate(selectedDate)}`
                    : 'Seleccioná un profesional para ver disponibilidad.'}
                </p>
              </div>
              <span className="badge neutral">{availableSlots.length} disponibles</span>
            </div>

            {isBootstrapping || isRefreshingAgenda ? (
              <div className="empty-state empty-state-soft">Cargando disponibilidad...</div>
            ) : availableSlots.length === 0 ? (
              <div className="empty-state">
                <strong>Sin slots disponibles</strong>
                <span>Generá disponibilidad o cambiá la fecha para mostrar horarios reservables.</span>
              </div>
            ) : (
                <>
                  {releasedSlotId && releasedSlotLabel ? (
                    <div className="slot-grid-feedback" role="status">
                      <strong>Slot liberado</strong>
                      <span>{releasedSlotLabel} ya está otra vez disponible y quedó seleccionado para que se note el cambio.</span>
                    </div>
                  ) : null}

                  <div className="slot-grid schedule-slot-grid">
                    {availableSlots.map((slot) => {
                      const isSelected = slot.id === selectedSlotId;
                      const isRecentlyReleased = slot.id === releasedSlotId;

                      return (
                        <button
                          key={slot.id}
                          type="button"
                          aria-pressed={isSelected}
                          className={`slot-button${isSelected ? ' selected' : ''}${isRecentlyReleased ? ' released' : ''}`}
                          onClick={() => {
                            setSelectedSlotId(slot.id);

                            if (slot.id !== releasedSlotId) {
                              setReleasedSlotId('');
                              setReleasedSlotLabel('');
                            }
                          }}
                        >
                          <span className="slot-card-kicker">Horario disponible</span>
                          <strong>{formatDateTimeRange(slot.start_time, slot.end_time)}</strong>
                          <span className="slot-note">{isRecentlyReleased ? 'Se liberó recién' : 'Seleccioná este horario'}</span>
                        </button>
                      );
                    })}
                  </div>
                </>
            )}
          </article>

          <article className="card">
            <div className="section-header">
              <div>
                <h2>Turnos del día</h2>
                <p>Listado simple con paciente, horario y estado listo para operar.</p>
              </div>
              <span className="badge neutral">{appointments.length} total</span>
            </div>

            {isBootstrapping || isRefreshingAgenda ? (
              <div className="empty-state empty-state-soft">Cargando turnos...</div>
            ) : appointments.length === 0 ? (
              <div className="empty-state">
                <strong>Sin turnos cargados</strong>
                <span>Reservá uno desde un slot disponible para que la demo muestre el flujo completo.</span>
              </div>
            ) : (
                <div className="list schedule-appointments-list">
                  {appointments.map((appointment) => {
                    const slot = slotById.get(appointment.slot_id);
                    const patientName = patientNameById.get(appointment.patient_id) ?? appointment.patient_id;
                    const isCancelled = appointment.status === 'cancelled';

                    return (
                      <div key={appointment.id} className="appointment-item">
                        <div className="appointment-main">
                          <div className="appointment-avatar" aria-hidden="true">
                            {getInitials(patientName)}
                          </div>

                          <div className="stack-tight">
                            <strong>{patientName}</strong>
                            <small>{slot ? formatDateTimeRange(slot.start_time, slot.end_time) : 'Horario no disponible'}</small>
                            {selectedProfessional ? (
                              <small>
                                {selectedProfessional.first_name} {selectedProfessional.last_name} · {formatLongDate(selectedDate)}
                              </small>
                            ) : null}
                          </div>
                        </div>

                        <div className="appointment-meta">
                          <span className={`pill${isCancelled ? ' cancelled' : ''}`}>{isCancelled ? 'Cancelado' : 'Reservado'}</span>
                        </div>

                        <div className="appointment-actions">
                          <button
                            className="button ghost"
                            type="button"
                            onClick={() => void handleCancelAppointment(appointment.id)}
                            disabled={isCancelled || cancellingAppointmentId === appointment.id}
                          >
                            {cancellingAppointmentId === appointment.id ? 'Cancelando...' : 'Cancelar'}
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>
            )}
          </article>
        </section>
      </div>
    </div>
  );
}

function getInitials(fullName: string) {
  return fullName
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((chunk) => chunk[0]?.toUpperCase() ?? '')
    .join('');
}

function compareByStartTime(left: Slot, right: Slot) {
  return new Date(left.start_time).getTime() - new Date(right.start_time).getTime();
}

function compareAppointmentsBySlot(slots: Slot[]) {
  const slotOrder = new Map(slots.map((slot, index) => [slot.id, index]));

  return (left: Appointment, right: Appointment) => {
    const leftOrder = slotOrder.get(left.slot_id) ?? Number.MAX_SAFE_INTEGER;
    const rightOrder = slotOrder.get(right.slot_id) ?? Number.MAX_SAFE_INTEGER;

    return leftOrder - rightOrder;
  };
}
