import { FormEvent, useEffect, useState } from 'react';
import { createPatientRequest, listPublicAvailability } from '../../api/appointments';
import type { PublicAvailabilitySlot } from '../../types/appointments';
import { listPublicProfessionals } from '../../api/directory';
import type { Professional } from '../../types/directory';

type SubmitState = 'idle' | 'submitting' | 'success' | 'error';

const shortWeekday = ['dom', 'lun', 'mar', 'mié', 'jue', 'vie', 'sáb'];
const longWeekday = ['domingo', 'lunes', 'martes', 'miércoles', 'jueves', 'viernes', 'sábado'];

export function PatientRequestSurface() {
  const [document, setDocument] = useState('');
  const [professionalId, setProfessionalId] = useState('');
  const [weekStart, setWeekStart] = useState(() => currentOperationalWeekStart());
  const [notes, setNotes] = useState('');
  const [contact, setContact] = useState('');
  const [professionals, setProfessionals] = useState<Professional[]>([]);
  const [availability, setAvailability] = useState<PublicAvailabilitySlot[]>([]);
  const [selectedSlot, setSelectedSlot] = useState<PublicAvailabilitySlot | null>(null);
  const [loadError, setLoadError] = useState('');
  const [availabilityError, setAvailabilityError] = useState('');
  const [submitState, setSubmitState] = useState<SubmitState>('idle');
  const [submitError, setSubmitError] = useState('');

  useEffect(() => {
    let alive = true;

    listPublicProfessionals()
      .then((response) => {
        if (!alive) {
          return;
        }
        setProfessionals(response.items);
        setProfessionalId((current) => current || response.items[0]?.id || '');
      })
      .catch(() => {
        if (alive) {
          setLoadError('No pudimos cargar profesionales. Probá de nuevo en unos minutos.');
        }
      });

    return () => {
      alive = false;
    };
  }, []);

  useEffect(() => {
    if (!professionalId) {
      setAvailability([]);
      setSelectedSlot(null);
      return;
    }

    let alive = true;
    setAvailabilityError('');
    setSelectedSlot(null);

    listPublicAvailability({ professional_id: professionalId, week_start: weekStart })
      .then((response) => {
        if (!alive) {
          return;
        }
        setAvailability(response.items);
      })
      .catch(() => {
        if (alive) {
          setAvailability([]);
          setAvailabilityError('No pudimos cargar horarios disponibles. Podés solicitar sin horario.');
        }
      });

    return () => {
      alive = false;
    };
  }, [professionalId, weekStart]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitState('submitting');
    setSubmitError('');

    try {
      const selectedRange = selectedSlot
        ? {
            scheduled_start: selectedSlot.start_time,
            scheduled_end: selectedSlot.end_time,
          }
        : {};

      const response = await createPatientRequest({
        document,
        professional_id: professionalId,
        notes: notes || undefined,
        contact: contact || undefined,
        ...selectedRange,
      });
      setSubmitState(response.status === 'scheduled' ? 'success' : 'success');
    } catch {
      setSubmitState('error');
      setSubmitError('No encontramos el DNI o no pudimos registrar la solicitud. Revisá los datos e intentá de nuevo.');
    }
  }

  const canSubmit = document.trim() !== '' && professionalId !== '' && submitState !== 'submitting';
  const groupedAvailability = groupAvailabilityByDay(availability);

  return (
    <main className="page page-centered">
      <section className="card stack" aria-labelledby="patient-request-title">
        <div className="hero-kicker">Solicitud de turno</div>
        <h1 id="patient-request-title">Pedí un turno con tu DNI</h1>
        <p>Ingresá tu documento y elegí profesional. Por privacidad, no mostramos tus datos personales en esta pantalla.</p>

        {loadError ? <p role="alert">{loadError}</p> : null}

        <form className="stack" onSubmit={handleSubmit}>
          <label>
            DNI / documento
            <input value={document} onChange={(event) => setDocument(event.target.value)} autoComplete="off" />
          </label>

          <label>
            Profesional
            <select value={professionalId} onChange={(event) => setProfessionalId(event.target.value)}>
              {professionals.map((professional) => (
                <option key={professional.id} value={professional.id}>
                  {professional.first_name} {professional.last_name} — {professional.specialty}
                </option>
              ))}
            </select>
          </label>

          <label>
            Semana
            <input type="date" value={weekStart} onChange={(event) => setWeekStart(toOperationalWeekStart(event.target.value))} />
          </label>

          <section className="stack" aria-labelledby="availability-title">
            <h2 id="availability-title">Horarios disponibles</h2>
            <p>Elegí un horario para reservar directo, o enviá la solicitud sin horario si no te sirve ninguno.</p>
            {availabilityError ? <p role="alert">{availabilityError}</p> : null}
            {groupedAvailability.length === 0 && !availabilityError ? <p>No hay horarios disponibles para esta semana.</p> : null}
            {groupedAvailability.map((group) => (
              <div key={group.date} role="group" aria-label={group.longLabel} className="stack">
                <strong>{group.longLabel}</strong>
                <div className="actions-row">
                  {group.slots.map((slot) => {
                    const selected = selectedSlot?.start_time === slot.start_time && selectedSlot?.end_time === slot.end_time;
                    return (
                      <button
                        key={`${slot.start_time}-${slot.end_time}`}
                        type="button"
                        aria-pressed={selected}
                        onClick={() => setSelectedSlot(slot)}
                      >
                        {formatSlotButtonLabel(slot)}
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}
          </section>

          <label>
            Nota opcional
            <textarea value={notes} onChange={(event) => setNotes(event.target.value)} />
          </label>

          <label>
            Contacto opcional
            <input value={contact} onChange={(event) => setContact(event.target.value)} />
          </label>

          <button type="submit" disabled={!canSubmit}>
            {submitState === 'submitting' ? 'Enviando...' : selectedSlot ? 'Reservar horario' : 'Solicitar sin horario'}
          </button>
        </form>

        {submitState === 'success' ? (
          <p role="status">
            {selectedSlot
              ? 'Turno reservado. Te esperamos en el horario elegido.'
              : 'Solicitud recibida. El equipo se va a contactar para coordinar el turno.'}
          </p>
        ) : null}
        {submitError ? <p role="alert">{submitError}</p> : null}
      </section>
    </main>
  );
}

function currentOperationalWeekStart() {
  const today = new Date();
  const day = today.getDay() === 0 ? 7 : today.getDay();
  const monday = new Date(Date.UTC(today.getFullYear(), today.getMonth(), today.getDate() - day + 1));
  return formatISODate(monday);
}

function toOperationalWeekStart(value: string) {
  if (!value) {
    return currentOperationalWeekStart();
  }
  const selected = parseDateOnly(value);
  const day = selected.getUTCDay() === 0 ? 7 : selected.getUTCDay();
  selected.setUTCDate(selected.getUTCDate() - day + 1);
  return formatISODate(selected);
}

function groupAvailabilityByDay(slots: PublicAvailabilitySlot[]) {
  const groups = new Map<string, { date: string; longLabel: string; slots: PublicAvailabilitySlot[] }>();
  [...slots]
    .sort((left, right) => left.start_time.localeCompare(right.start_time))
    .forEach((slot) => {
      const start = new Date(slot.start_time);
      const date = formatISODate(start);
      const current = groups.get(date) ?? { date, longLabel: formatLongDayLabel(start), slots: [] };
      current.slots.push(slot);
      groups.set(date, current);
    });
  return [...groups.values()];
}

function formatSlotButtonLabel(slot: PublicAvailabilitySlot) {
  const start = new Date(slot.start_time);
  const end = new Date(slot.end_time);
  return `${shortWeekday[start.getUTCDay()]} ${pad(start.getUTCDate())}/${pad(start.getUTCMonth() + 1)}, ${formatClock(start)}–${formatClock(end)}`;
}

function formatLongDayLabel(date: Date) {
  return `${longWeekday[date.getUTCDay()]} ${pad(date.getUTCDate())}/${pad(date.getUTCMonth() + 1)}`;
}

function formatClock(date: Date) {
  return `${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}`;
}

function parseDateOnly(value: string) {
  const [year, month, day] = value.split('-').map(Number);
  return new Date(Date.UTC(year, month - 1, day));
}

function formatISODate(date: Date) {
  return `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}`;
}

function pad(value: number) {
  return value.toString().padStart(2, '0');
}
