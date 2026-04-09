import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { cancelAppointment, createAppointment, createSlotsBulk, listOperationalWeek, type OperationalWeekDay } from '../../api/appointments';
import { listPatients, listProfessionals } from '../../api/directory';
import { type AgendaMode } from '../../auth/actorCapabilities';
import { resolveAuthenticatedViewError } from '../../auth/authenticatedViewPolicy';
import type { Appointment, BulkCreateSlotsPayload, Slot } from '../../types/appointments';
import type { Patient, Professional } from '../../types/directory';
import {
  formatDateInputValue,
  formatDateTimeRange,
  formatLongDate,
  formatTimeBand,
  getUtcWeekStart,
  mapDateToOperationalWeek,
} from './helpers';

type ScheduleDemoProps = {
  agendaMode: AgendaMode;
  onSessionInvalid: () => void;
};

export function ScheduleDemo({ agendaMode, onSessionInvalid }: ScheduleDemoProps) {
  const initialSelectedDate = mapDateToOperationalWeek(formatDateInputValue());
  const [professionals, setProfessionals] = useState<Professional[]>([]);
  const [patients, setPatients] = useState<Patient[]>([]);
  const [selectedProfessionalId, setSelectedProfessionalId] = useState('');
  const [selectedPatientId, setSelectedPatientId] = useState('');
  const [selectedDate, setSelectedDate] = useState(initialSelectedDate);
  const [weekStart, setWeekStart] = useState(getUtcWeekStart(initialSelectedDate));
  const [days, setDays] = useState<OperationalWeekDay[]>([]);
  const [timeBands, setTimeBands] = useState<string[]>([]);
  const [focusedBand, setFocusedBand] = useState('');
  const [selectedSlotId, setSelectedSlotId] = useState('');
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

  const isDoctorAgenda = agendaMode.kind === 'doctor-own';

  const clearReleasedSlotFeedback = useCallback(() => {
    releasedSlotIdRef.current = '';
    setReleasedSlotId('');
    setReleasedSlotLabel('');
  }, []);

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
  const selectedPatient = useMemo(() => patients.find((patient) => patient.id === selectedPatientId) ?? null, [patients, selectedPatientId]);
  const patientNameById = useMemo(
    () => new Map(patients.map((patient) => [patient.id, `${patient.first_name} ${patient.last_name}`])),
    [patients],
  );
  const allSlots = useMemo(() => days.flatMap((day) => day.slots), [days]);
  const allAppointments = useMemo(() => days.flatMap((day) => day.appointments), [days]);
  const slotById = useMemo(() => new Map(allSlots.map((slot) => [slot.id, slot])), [allSlots]);
  const appointmentBySlotId = useMemo(
    () => new Map(allAppointments.filter((appointment) => appointment.status === 'booked').map((appointment) => [appointment.slot_id, appointment])),
    [allAppointments],
  );
  const selectedSlot = useMemo(() => slotById.get(selectedSlotId) ?? null, [selectedSlotId, slotById]);
  const selectedWeekDay = useMemo(() => days.find((day) => day.date === selectedDate) ?? null, [days, selectedDate]);
  const weeklySummary = useMemo(
    () => ({
      available: days.reduce((total, day) => total + day.summary.available, 0),
      booked: days.reduce((total, day) => total + day.summary.booked, 0),
      cancelled: days.reduce((total, day) => total + day.summary.cancelled, 0),
    }),
    [days],
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
      setSelectedPatientId((current) => (nextPatients.some((patient) => patient.id === current) ? current : nextPatients[0]?.id || ''));
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
      setDays([]);
      setTimeBands([]);
      setFocusedBand('');
      setSelectedSlotId('');
      return;
    }

    if (!selectedProfessionalId || !weekStart) {
      setDays([]);
      setTimeBands([]);
      setFocusedBand('');
      setSelectedSlotId('');
      return;
    }

    try {
      setIsRefreshingAgenda(true);
      setErrorMessage('');
      setAccessDeniedMessage('');

      const nextWeek = await listOperationalWeek({ professional_id: selectedProfessionalId, date: weekStart });
      const nextSlots = nextWeek.days.flatMap((day) => day.slots);

      setWeekStart(nextWeek.weekStart);
      setDays(nextWeek.days);
      setTimeBands(nextWeek.timeBands);
      setSelectedDate((current) => {
        if (nextWeek.days.some((day) => day.date === current)) {
          return current;
        }

        return nextWeek.days[0]?.date ?? current;
      });
      setSelectedSlotId((current) =>
        nextSlots.find((slot) => slot.id === releasedSlotIdRef.current && slot.status === 'available')?.id ||
        nextSlots.find((slot) => slot.id === current && slot.status === 'available')?.id ||
        nextSlots.find((slot) => slot.status === 'available')?.id ||
        '',
      );
      setFocusedBand((current) => current || nextWeek.timeBands[0] || '');
    } catch (error) {
      const nextError = handleApiFailure(error, 'No se pudo cargar la agenda seleccionada.');
      setErrorMessage(nextError.errorMessage);
      setAccessDeniedMessage(nextError.accessDeniedMessage);
      setDays([]);
      setTimeBands([]);
      setFocusedBand('');
      setSelectedSlotId('');
    } finally {
      setIsRefreshingAgenda(false);
    }
  }, [agendaMode.kind, handleApiFailure, selectedProfessionalId, weekStart]);

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  useEffect(() => {
    if (isBootstrapping) {
      return;
    }

    void refreshAgenda();
  }, [isBootstrapping, refreshAgenda]);

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
    const appointment = allAppointments.find((item) => item.id === appointmentId);
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

  function shiftWeek(daysOffset: number) {
    const currentWeekStart = new Date(`${weekStart}T00:00:00Z`);
    const currentSelectedDay = new Date(`${selectedDate}T00:00:00Z`);
    const selectedDayOffset = Math.round((currentSelectedDay.getTime() - currentWeekStart.getTime()) / 86400000);
    const nextWeekStart = new Date(currentWeekStart);

    nextWeekStart.setUTCDate(nextWeekStart.getUTCDate() + daysOffset);

    const nextSelectedDay = new Date(nextWeekStart);
    nextSelectedDay.setUTCDate(nextSelectedDay.getUTCDate() + selectedDayOffset);

    clearReleasedSlotFeedback();
    setWeekStart(formatDateInputValue(nextWeekStart));
    setSelectedDate(formatDateInputValue(nextSelectedDay));
    setSelectedSlotId('');
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
      <section className="card schedule-overview stack">
        <div className="stack">
          <div className="schedule-hero-badges status-bar">
            <span className="badge neutral">Profesionales activos: {professionals.length}</span>
            <span className="badge neutral">Pacientes activos: {patients.length}</span>
            <span className="badge neutral">Disponibles semana: {weeklySummary.available}</span>
          </div>

          {(successMessage || errorMessage || accessDeniedMessage) && (
            <div className="status-bar" aria-live="polite">
              {successMessage ? <span className="badge success">{successMessage}</span> : null}
              {errorMessage ? <span className="badge error">{errorMessage}</span> : null}
              {accessDeniedMessage ? <span className="badge error">Acceso denegado: {accessDeniedMessage}</span> : null}
            </div>
          )}
        </div>

        <div className="schedule-hero-panel">
          <div className="schedule-hero-panel-head">
            <span className="summary-label">Contexto de agenda</span>
            <strong>{selectedProfessional ? `${selectedProfessional.first_name} ${selectedProfessional.last_name}` : 'Sin profesional'}</strong>
            <small>{selectedProfessional?.specialty ?? 'Elegí un profesional para revisar la agenda.'}</small>
          </div>

          <div className="schedule-stat-grid">
            <article className="schedule-stat-card">
              <span className="summary-label">Semana UTC</span>
              <strong>{formatLongDate(weekStart)}</strong>
              <small>Tablero operativo de lunes a viernes.</small>
            </article>
            <article className="schedule-stat-card">
              <span className="summary-label">Disponibles</span>
              <strong>{weeklySummary.available}</strong>
              <small>{weeklySummary.available > 0 ? 'Listos para reservar.' : 'Sin disponibilidad reservable.'}</small>
            </article>
            <article className="schedule-stat-card">
              <span className="summary-label">Reservados</span>
              <strong>{weeklySummary.booked}</strong>
              <small>Turnos activos visibles.</small>
            </article>
            <article className="schedule-stat-card">
              <span className="summary-label">Cancelados</span>
              <strong>{weeklySummary.cancelled}</strong>
              <small>Visibles para seguimiento.</small>
            </article>
          </div>
        </div>
      </section>

      <section className="card card-accent schedule-controls-card stack">
        <div className="schedule-controls-head">
          <div>
            <span className="summary-label">Contexto</span>
            <h2>Filtros de agenda</h2>
            <p className="helper">Elegí profesional y semana para cargar la operación visible sin salir del espacio actual.</p>
          </div>
          <div className="schedule-controls-toolbar">
            <span className="badge info">UTC fijo para evitar corrimientos</span>
            <button className="button secondary" type="button" onClick={() => void refreshAgenda()} disabled={isBootstrapping || isRefreshingAgenda}>
              {isRefreshingAgenda ? 'Actualizando...' : 'Actualizar agenda'}
            </button>
          </div>
        </div>

        <div className="schedule-filters-grid">
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
                const nextSelectedDate = mapDateToOperationalWeek(event.target.value);

                clearReleasedSlotFeedback();
                setSelectedDate(nextSelectedDate);
                setWeekStart(getUtcWeekStart(nextSelectedDate));
                setSelectedSlotId('');
              }}
            />
            <small className="helper">La fecha define el día operativo seleccionado; la semana visible se acomoda en UTC alrededor de ese día.</small>
          </div>

          <div className="schedule-controls-toolbar">
            <button className="button secondary" type="button" onClick={() => shiftWeek(-7)} disabled={isBootstrapping || isRefreshingAgenda}>
              Semana anterior
            </button>
            <button className="button secondary" type="button" onClick={() => shiftWeek(7)} disabled={isBootstrapping || isRefreshingAgenda}>
              Semana siguiente
            </button>
          </div>
        </div>
      </section>

      <div className="layout schedule-layout">
        <section className="stack schedule-main">
          <section className="card stack">
            <div className="section-header">
              <div>
                <span className="summary-label">Tablero</span>
                <h2>Semana operativa</h2>
                <p>{selectedProfessional ? `${selectedProfessional.first_name} ${selectedProfessional.last_name} · foco actual ${formatLongDate(selectedDate)}` : 'Seleccioná un profesional para ver disponibilidad.'}</p>
              </div>
            </div>

            {releasedSlotId && releasedSlotLabel ? (
              <div className="slot-grid-feedback" role="status">
                <strong>Slot liberado</strong>
                <span>{releasedSlotLabel} ya está otra vez disponible y quedó seleccionado para que se note el cambio.</span>
              </div>
            ) : null}

            <div className="schedule-week-board stack">
              <div className="schedule-week-header" role="row">
                <div className="summary-label schedule-week-axis">UTC</div>
                {days.map((day) => (
                  <button
                    key={day.date}
                    type="button"
                    aria-pressed={day.date === selectedDate}
                    className={`schedule-day-card${day.date === selectedDate ? ' selected' : ''}`}
                    onClick={() => {
                      setSelectedDate(day.date);
                      setSelectedSlotId('');
                    }}
                  >
                    <strong>{day.weekdayLabel}</strong>
                    <small>{day.date === selectedDate ? 'Día seleccionado para operar' : 'Hacé click para operar este día'}</small>
                    {day.slots.length === 0 && day.appointments.length === 0 ? <small>Sin actividad para este día</small> : null}
                    <small>
                      {day.summary.available} disp. · {day.summary.booked} reserv. · {day.summary.cancelled} canc.
                    </small>
                  </button>
                ))}
              </div>

              {timeBands.length === 0 ? (
                <div className="schedule-week-empty-grid">
                  <div className="summary-label schedule-week-axis">UTC</div>
                  {days.map((day) => (
                    <div key={day.date} className="empty-state">
                      <strong>{day.weekdayLabel}</strong>
                      <span>Sin actividad para este día</span>
                    </div>
                  ))}
                </div>
              ) : (
                timeBands.map((band) => (
                  <div key={band} className={`schedule-week-row${focusedBand === band ? ' selected' : ''}`} role="row">
                    <div className="summary-label">{band}</div>
                    {days.map((day) => {
                      const slot = day.slots.find((item) => formatTimeBand(item.start_time) === band) ?? null;
                      const appointment = slot ? appointmentBySlotId.get(slot.id) ?? day.appointments.find((item) => item.slot_id === slot.id) ?? null : null;
                      const patientName = appointment ? patientNameById.get(appointment.patient_id) ?? appointment.patient_id : '';
                      const isSelectedSlot = slot?.id === selectedSlotId;
                      const isReleasedSlot = slot?.id === releasedSlotId;
                      const cellLabel = slot ? `${day.weekdayLabel} · ${formatDateTimeRange(slot.start_time, slot.end_time)}` : `${day.weekdayLabel} · ${band}`;

                      return (
                        <div key={`${day.date}-${band}`} data-testid={`board-cell-${day.date}-${band}`} className={`card schedule-board-cell${day.date === selectedDate ? ' selected-day' : ''}`}>
                          {slot?.status === 'available' ? (
                            <div className="stack-tight">
                              {appointment?.status === 'cancelled' ? (
                                <>
                                  <strong>{patientName}</strong>
                                  <span className="pill cancelled">Cancelado</span>
                                </>
                              ) : null}
                              <button
                                type="button"
                                aria-pressed={isSelectedSlot}
                                className={`slot-button${isSelectedSlot ? ' selected' : ''}${isReleasedSlot ? ' released' : ''}`}
                                onClick={() => {
                                  setSelectedDate(day.date);
                                  setSelectedSlotId(slot.id);
                                  setFocusedBand(band);
                                  if (slot.id !== releasedSlotId) {
                                    setReleasedSlotId('');
                                    setReleasedSlotLabel('');
                                  }
                                }}
                              >
                                <span className="slot-card-kicker">Horario disponible</span>
                                <strong>{cellLabel}</strong>
                                <span className="slot-note">{isReleasedSlot ? 'Se liberó recién' : 'Seleccioná este horario'}</span>
                              </button>
                            </div>
                          ) : appointment ? (
                            <div className="stack-tight">
                              <strong>{patientName}</strong>
                              <small>{slot ? formatDateTimeRange(slot.start_time, slot.end_time) : 'Horario no disponible'}</small>
                              {selectedProfessional ? (
                                <small>
                                  {selectedProfessional.first_name} {selectedProfessional.last_name} · {formatLongDate(day.date)}
                                </small>
                              ) : null}
                              <span className={`pill${appointment.status === 'cancelled' ? ' cancelled' : ''}`}>{appointment.status === 'cancelled' ? 'Cancelado' : 'Reservado'}</span>
                              <button
                                className="button ghost"
                                type="button"
                                onClick={() => {
                                  setSelectedDate(day.date);
                                  setFocusedBand(band);
                                  void handleCancelAppointment(appointment.id);
                                }}
                                disabled={appointment.status === 'cancelled' || cancellingAppointmentId === appointment.id}
                              >
                                {cancellingAppointmentId === appointment.id ? 'Cancelando...' : 'Cancelar'}
                              </button>
                            </div>
                          ) : (
                            <span className="helper">—</span>
                          )}
                        </div>
                      );
                    })}
                  </div>
                ))
              )}
            </div>
          </section>

          <section className="card schedule-booking-card stack">
            <div>
              <span className="summary-label">Reserva</span>
              <h2>Reservar turno</h2>
              <p className="helper">La reserva sigue arrancando desde la grilla principal: primero elegí un horario y después asignalo a un paciente.</p>
            </div>

            <div className="field">
              <label htmlFor="patient">Paciente</label>
              <select id="patient" value={selectedPatientId} onChange={(event) => setSelectedPatientId(event.target.value)} disabled={isBootstrapping || patients.length === 0}>
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

            <button className="button" type="button" onClick={() => void handleBookAppointment()} disabled={isBootstrapping || isRefreshingAgenda || isBooking || !selectedSlotId || !selectedPatientId}>
              {isBooking ? 'Reservando...' : 'Reservar turno'}
            </button>
          </section>
        </section>

        <aside className="schedule-sidebar stack">
          <section className="card schedule-generator-card stack" aria-labelledby="generate-slots-title">
            <div className="stack-tight">
              <span className="summary-label schedule-dark-eyebrow">Soporte</span>
              <h2 id="generate-slots-title">Generar slots</h2>
              <p className="helper">Acción secundaria: cargá disponibilidad para el día seleccionado sin salir de esta vista.</p>
            </div>

            <div className="inline-note">
              <strong>{selectedWeekDay?.weekdayLabel ?? 'Sin día seleccionado'}</strong>
              <span>{selectedWeekDay ? formatLongDate(selectedWeekDay.date) : 'Elegí una fecha para cargar disponibilidad.'}</span>
            </div>

            <div className="time-grid">
              <div className="field">
                <label htmlFor="slot-start-time">Desde</label>
                <input id="slot-start-time" type="time" value={slotStartTime} onChange={(event) => setSlotStartTime(event.target.value)} />
              </div>

              <div className="field">
                <label htmlFor="slot-end-time">Hasta</label>
                <input id="slot-end-time" type="time" value={slotEndTime} onChange={(event) => setSlotEndTime(event.target.value)} />
              </div>
            </div>

            <div className="field">
              <label htmlFor="slot-duration">Duración del slot</label>
              <select id="slot-duration" value={slotDurationMinutes} onChange={(event) => setSlotDurationMinutes(Number(event.target.value) as BulkCreateSlotsPayload['slot_duration_minutes'])}>
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

            <button className="button schedule-generator-button" type="button" onClick={() => void handleCreateSlots()} disabled={isBootstrapping || isRefreshingAgenda || isCreatingSlots || !selectedProfessionalId || !selectedDate}>
              {isCreatingSlots ? 'Generando...' : 'Generar slots'}
            </button>
          </section>
        </aside>
      </div>
    </div>
  );
}
