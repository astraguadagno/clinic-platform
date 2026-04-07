import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { cancelAppointment, createAppointment, createSlotsBulk, listAppointments, listSlots } from '../../api/appointments';
import { listPatients, listProfessionals } from '../../api/directory';
import { ApiError } from '../../api/http';
import type { Appointment, BulkCreateSlotsPayload, Slot } from '../../types/appointments';
import type { Patient, Professional } from '../../types/directory';
import { formatDateInputValue, formatDateTimeRange, formatLongDate } from './helpers';

export function ScheduleDemo() {
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

  const selectedProfessional = useMemo(
    () => professionals.find((professional) => professional.id === selectedProfessionalId) ?? null,
    [professionals, selectedProfessionalId],
  );

  const availableSlots = useMemo(
    () => daySlots.filter((slot) => slot.status === 'available').sort(compareByStartTime),
    [daySlots],
  );

  const slotById = useMemo(
    () => new Map(daySlots.map((slot) => [slot.id, slot])),
    [daySlots],
  );

  const patientNameById = useMemo(
    () => new Map(patients.map((patient) => [patient.id, `${patient.first_name} ${patient.last_name}`])),
    [patients],
  );

  const bookedAppointmentsCount = useMemo(
    () => appointments.filter((appointment) => appointment.status === 'booked').length,
    [appointments],
  );

  const bootstrap = useCallback(async () => {
    try {
      setIsBootstrapping(true);
      setErrorMessage('');

      const [professionalsResponse, patientsResponse] = await Promise.all([listProfessionals(), listPatients()]);

      const nextProfessionals = professionalsResponse.items.filter((professional) => professional.active);
      const nextPatients = patientsResponse.items.filter((patient) => patient.active);

      setProfessionals(nextProfessionals);
      setPatients(nextPatients);
      setSelectedProfessionalId((current) =>
        nextProfessionals.some((professional) => professional.id === current) ? current : nextProfessionals[0]?.id || '',
      );
      setSelectedPatientId((current) =>
        nextPatients.some((patient) => patient.id === current) ? current : nextPatients[0]?.id || '',
      );
    } catch (error) {
      setErrorMessage(getErrorMessage(error, 'No se pudieron cargar profesionales y pacientes.'));
    } finally {
      setIsBootstrapping(false);
    }
  }, []);

  const refreshAgenda = useCallback(async () => {
    if (!selectedProfessionalId || !selectedDate) {
      setDaySlots([]);
      setAppointments([]);
      setSelectedSlotId('');
      return;
    }

    try {
      setIsRefreshingAgenda(true);
      setErrorMessage('');

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
      setErrorMessage(getErrorMessage(error, 'No se pudo cargar la agenda seleccionada.'));
      setDaySlots([]);
      setAppointments([]);
      setSelectedSlotId('');
    } finally {
      setIsRefreshingAgenda(false);
    }
  }, [selectedDate, selectedProfessionalId]);

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
      setErrorMessage(getErrorMessage(error, 'No se pudo reservar el turno.'));
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
      setErrorMessage(getErrorMessage(error, 'No se pudo cancelar el turno.'));
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
      setErrorMessage(getErrorMessage(error, 'No se pudieron generar los slots.'));
    } finally {
      setIsCreatingSlots(false);
    }
  }

  return (
    <div className="stack">
      <header className="hero section-hero-card card">
        <div className="hero-kicker">Operación diaria demo</div>
        <h1>Agenda demo</h1>
        <p>
          Vertical slice simple para mostrar disponibilidad, reservar turnos y cancelar sobre los servicios actuales,
          con foco en una experiencia más clara para demo.
        </p>
        <div className="status-bar">
          <span className="badge neutral">Profesionales activos: {professionals.length}</span>
          <span className="badge neutral">Pacientes activos: {patients.length}</span>
          <span className="badge info">Refresca datos del Directorio al volver al tab</span>
          {successMessage ? <span className="badge success">{successMessage}</span> : null}
          {errorMessage ? <span className="badge error">{errorMessage}</span> : null}
        </div>
      </header>

      <section className="overview-grid">
        <article className="overview-card card">
          <span className="summary-label">Fecha elegida</span>
          <strong>{formatLongDate(selectedDate)}</strong>
          <small>{selectedProfessional ? 'Vista filtrada por profesional.' : 'Elegí un profesional para operar.'}</small>
        </article>
        <article className="overview-card card">
          <span className="summary-label">Disponibilidad</span>
          <strong>{availableSlots.length} slots</strong>
          <small>{availableSlots.length > 0 ? 'Hay horarios listos para reservar.' : 'No hay disponibilidad cargada aún.'}</small>
        </article>
        <article className="overview-card card">
          <span className="summary-label">Turnos del día</span>
          <strong>{bookedAppointmentsCount} activos</strong>
          <small>{appointments.length} registros totales contando cancelados.</small>
        </article>
      </section>

      <div className="layout">
        <aside className="card stack card-accent">
            <div>
              <h2>Controles</h2>
              <p className="helper">Directo a los endpoints existentes, con pasos guiados y feedback visual más claro.</p>
            </div>

            <div className="field">
              <label htmlFor="professional">Profesional</label>
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

            <section className="subsection stack" aria-labelledby="generate-slots-title">
              <div>
                <h3 id="generate-slots-title">Generar slots</h3>
                <p className="helper">Usá el mismo profesional y fecha para crear disponibilidad rápida y mostrar movimiento.</p>
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

              <button
                className="button secondary"
                type="button"
                onClick={() => void handleCreateSlots()}
                disabled={isBootstrapping || isRefreshingAgenda || isCreatingSlots || !selectedProfessionalId || !selectedDate}
              >
                {isCreatingSlots ? 'Generando...' : 'Generar slots'}
              </button>
            </section>

            <div className="field">
              <label htmlFor="slot">Slot a reservar</label>
              <select
                id="slot"
                value={selectedSlotId}
                onChange={(event) => setSelectedSlotId(event.target.value)}
                disabled={availableSlots.length === 0 || isRefreshingAgenda}
              >
                {availableSlots.length === 0 ? <option value="">No hay slots disponibles</option> : null}
                {availableSlots.map((slot) => (
                  <option key={slot.id} value={slot.id}>
                    {formatDateTimeRange(slot.start_time, slot.end_time)}
                  </option>
                ))}
              </select>
            </div>

            <div className="toolbar">
              <button
                className="button"
                type="button"
                onClick={() => void handleBookAppointment()}
                disabled={isBootstrapping || isRefreshingAgenda || isBooking || !selectedSlotId || !selectedPatientId}
              >
                {isBooking ? 'Reservando...' : 'Reservar turno'}
              </button>
              <button
                className="button secondary"
                type="button"
                onClick={() => {
                  clearReleasedSlotFeedback();
                  void refreshAgenda();
                }}
                disabled={isBootstrapping || isRefreshingAgenda}
              >
                {isRefreshingAgenda ? 'Actualizando...' : 'Refrescar agenda'}
              </button>
            </div>

            <div className="inline-note">
              <strong>Tip demo:</strong> si cargás un paciente o profesional en Directorio, al volver a Agenda se actualiza la
              base disponible sin recargar toda la app.
            </div>
        </aside>

        <section className="stack">
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
                <span>Probá generar disponibilidad rápida desde el panel izquierdo para esta fecha.</span>
              </div>
            ) : (
                <>
                  {releasedSlotId && releasedSlotLabel ? (
                    <div className="slot-grid-feedback" role="status">
                      <strong>Slot liberado</strong>
                      <span>{releasedSlotLabel} ya está otra vez disponible y quedó seleccionado para que se note el cambio.</span>
                    </div>
                  ) : null}

                  <div className="slot-grid">
                    {availableSlots.map((slot) => {
                      const isSelected = slot.id === selectedSlotId;
                      const isRecentlyReleased = slot.id === releasedSlotId;

                      return (
                        <button
                          key={slot.id}
                          type="button"
                          className={`slot-button${isSelected ? ' selected' : ''}${isRecentlyReleased ? ' released' : ''}`}
                          onClick={() => {
                            setSelectedSlotId(slot.id);

                            if (slot.id !== releasedSlotId) {
                              setReleasedSlotId('');
                              setReleasedSlotLabel('');
                            }
                          }}
                        >
                          <strong>{formatDateTimeRange(slot.start_time, slot.end_time)}</strong>
                          {isRecentlyReleased ? <span className="slot-note">Liberado recién</span> : null}
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
                <p>Listado simple con cancelación inmediata y badges más legibles para explicar el estado.</p>
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
                <div className="list">
                  {appointments.map((appointment) => {
                    const slot = slotById.get(appointment.slot_id);
                    const patientName = patientNameById.get(appointment.patient_id) ?? appointment.patient_id;
                    const isCancelled = appointment.status === 'cancelled';

                    return (
                      <div key={appointment.id} className="appointment-item">
                        <div>
                          <strong>{slot ? formatDateTimeRange(slot.start_time, slot.end_time) : 'Horario no disponible'}</strong>
                          <small>{appointment.id}</small>
                        </div>

                        <div className="appointment-meta">
                          <span className={`pill${isCancelled ? ' cancelled' : ''}`}>{isCancelled ? 'Cancelado' : 'Reservado'}</span>
                          <span className="muted">Paciente: {patientName}</span>
                          <span className="muted">Slot: {appointment.slot_id}</span>
                        </div>

                        <div>
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

function getErrorMessage(error: unknown, fallbackMessage: string) {
  if (error instanceof ApiError) {
    return error.message;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return fallbackMessage;
}
