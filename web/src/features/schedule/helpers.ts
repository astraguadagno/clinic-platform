const AGENDA_DEMO_LOCALE = 'es-AR';
const AGENDA_DEMO_TIME_ZONE = 'UTC';

// Demo rule: backend stores and filters agenda timestamps in UTC,
// so the frontend must render and default dates in UTC as well.
export function formatDateInputValue(date = new Date()) {
  const year = date.getUTCFullYear();
  const month = `${date.getUTCMonth() + 1}`.padStart(2, '0');
  const day = `${date.getUTCDate()}`.padStart(2, '0');

  return `${year}-${month}-${day}`;
}

export function formatDateTimeRange(startTime: string, endTime: string) {
  const start = new Date(startTime);
  const end = new Date(endTime);

  return `${formatTime(start)} - ${formatTime(end)} UTC`;
}

export function formatLongDate(date: string) {
  return new Intl.DateTimeFormat(AGENDA_DEMO_LOCALE, {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    timeZone: AGENDA_DEMO_TIME_ZONE,
  }).format(new Date(`${date}T00:00:00Z`));
}

function formatTime(date: Date) {
  return new Intl.DateTimeFormat(AGENDA_DEMO_LOCALE, {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: AGENDA_DEMO_TIME_ZONE,
  }).format(date);
}
