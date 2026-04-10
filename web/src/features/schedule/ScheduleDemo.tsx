import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { cancelAppointment, createAppointment, createSlotsBulk, listOperationalWeek, type OperationalWeekDay } from '../../api/appointments';
import { listPatients, listProfessionals } from '../../api/directory';
import { type AgendaMode } from '../../auth/actorCapabilities';
import { resolveAuthenticatedViewError } from '../../auth/authenticatedViewPolicy';
import { EmptyState, PageContainer, SectionCard } from '../../app-shell/AppShell.primitives';
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
  const [activeModal, setActiveModal] = useState<'book' | 'generate' | null>(null);
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

      setActiveModal(null);
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

      setActiveModal(null);
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

  const selectedSlotLabel = selectedSlot ? formatDateTimeRange(selectedSlot.start_time, selectedSlot.end_time) : 'Todavía sin horario seleccionado';
  const selectedDayLabel = selectedWeekDay?.weekdayLabel ?? 'Sin día seleccionado';
  const selectedDayLongLabel = selectedWeekDay ? formatLongDate(selectedWeekDay.date) : 'Elegí un día del tablero para operar.';
  const isBookActionDisabled = isBootstrapping || isRefreshingAgenda || isBooking || !selectedSlotId;
  const isGenerateActionDisabled = isBootstrapping || isRefreshingAgenda || isCreatingSlots || !selectedProfessionalId || !selectedDate;

  if (agendaMode.kind === 'forbidden') {
    return (
      <PageContainer className="schedule-demo">
        <EmptyState eyebrow="Agenda bloqueada" title="Acceso denegado" description={agendaMode.message} className="schedule-shell-empty" />
      </PageContainer>
    );
  }

  return (
    <PageContainer className="stack schedule-demo">
      {(successMessage || errorMessage || accessDeniedMessage) && (
        <div className="status-bar" aria-live="polite">
          {successMessage ? <span className="badge success">{successMessage}</span> : null}
          {errorMessage ? <span className="badge error">{errorMessage}</span> : null}
          {accessDeniedMessage ? <span className="badge error">Acceso denegado: {accessDeniedMessage}</span> : null}
        </div>
      )}

      <SectionCard className="card-accent schedule-context-bar">
        <div className="schedule-context-bar-main">
          <div>
            <span className="summary-label">Agenda</span>
            <h2>{selectedProfessional ? `${selectedProfessional.first_name} ${selectedProfessional.last_name}` : 'Sin profesional seleccionado'}</h2>
            <p className="helper">{selectedProfessional?.specialty ?? 'Elegí un profesional para revisar la agenda visible.'}</p>
          </div>
          <div className="schedule-context-bar-week">
            <span className="badge info">UTC fijo para evitar corrimientos</span>
            <span className="badge neutral">Semana UTC {formatLongDate(weekStart)}</span>
          </div>
        </div>

        <div className="schedule-context-bar-controls">
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

        </div>

        <div className="schedule-context-bar-actions">
          <button className="button secondary" type="button" onClick={() => shiftWeek(-7)} disabled={isBootstrapping || isRefreshingAgenda}>
            Semana anterior
          </button>
          <button className="button secondary" type="button" onClick={() => shiftWeek(7)} disabled={isBootstrapping || isRefreshingAgenda}>
            Semana siguiente
          </button>
          <button className="button secondary" type="button" onClick={() => void refreshAgenda()} disabled={isBootstrapping || isRefreshingAgenda}>
            {isRefreshingAgenda ? 'Actualizando...' : 'Actualizar agenda'}
          </button>
          <button className="button" type="button" onClick={() => setActiveModal('book')} disabled={isBookActionDisabled}>
            {isBooking ? 'Reservando...' : 'Reservar turno'}
          </button>
          <button className="button schedule-generator-button" type="button" onClick={() => setActiveModal('generate')} disabled={isGenerateActionDisabled}>
            {isCreatingSlots ? 'Generando...' : 'Generar slots'}
          </button>
        </div>
      </SectionCard>

      <section className="card stack schedule-board-shell">
        <div className="schedule-board-header">
          <div>
            <span className="summary-label">Tablero</span>
            <h2>Agenda semanal operativa</h2>
            <p>{selectedProfessional ? `${selectedProfessional.first_name} ${selectedProfessional.last_name} · tablero operativo de lunes a viernes.` : 'Seleccioná un profesional para ver disponibilidad.'}</p>
          </div>
          <div className="schedule-board-toolbar">
            <div className="inline-note">
              <strong>Día seleccionado: {selectedDayLabel}</strong>
              <span>{selectedDayLongLabel}</span>
            </div>
            {selectedSlot ? (
              <div className="inline-note">
                <strong>Slot listo para reservar</strong>
                <span>{selectedSlotLabel}</span>
              </div>
            ) : null}
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

      {activeModal ? (
        <div className="schedule-modal-backdrop" role="presentation">
          <section className="card stack schedule-modal" role="dialog" aria-modal="true" aria-labelledby={`schedule-modal-title-${activeModal}`}>
            <div className="schedule-modal-head">
              <div>
                <span className="summary-label">{activeModal === 'book' ? 'Reserva' : 'Disponibilidad'}</span>
                <h2 id={`schedule-modal-title-${activeModal}`}>{activeModal === 'book' ? 'Reservar turno' : 'Generar slots'}</h2>
                <p className="helper">{selectedDayLabel} · {selectedDayLongLabel}</p>
              </div>
              <button className="button ghost" type="button" onClick={() => setActiveModal(null)}>
                Cerrar
              </button>
            </div>

            {activeModal === 'book' ? (
              <>
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
                  <strong>{selectedSlotLabel}</strong>
                  <span>
                    {selectedSlot
                      ? `Paciente listo para reservar: ${selectedPatient ? `${selectedPatient.first_name} ${selectedPatient.last_name}` : 'seleccioná un paciente'}.`
                      : 'Seleccioná un horario desde la grilla de disponibilidad para continuar.'}
                  </span>
                </div>

                <div className="schedule-modal-actions">
                  <button className="button ghost" type="button" onClick={() => setActiveModal(null)}>
                    Cancelar
                  </button>
                  <button className="button" type="button" onClick={() => void handleBookAppointment()} disabled={isBootstrapping || isRefreshingAgenda || isBooking || !selectedSlotId || !selectedPatientId}>
                    {isBooking ? 'Reservando...' : 'Confirmar reserva'}
                  </button>
                </div>
              </>
            ) : (
              <>
                <div className="inline-note">
                  <strong>{selectedDayLabel}</strong>
                  <span>{selectedDayLongLabel}</span>
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

                <div className="schedule-modal-actions">
                  <button className="button ghost" type="button" onClick={() => setActiveModal(null)}>
                    Cancelar
                  </button>
                  <button className="button schedule-generator-button" type="button" onClick={() => void handleCreateSlots()} disabled={isGenerateActionDisabled}>
                    {isCreatingSlots ? 'Generando...' : 'Confirmar generación'}
                  </button>
                </div>
              </>
            )}
          </section>
        </div>
      ) : null}
    </PageContainer>
  );
}
