const AGENDA_DEMO_LOCALE = 'es-AR';
const AGENDA_DEMO_TIME_ZONE = 'UTC';

const WEEKDAY_SHORT_LABELS = ['Dom', 'Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb'] as const;

export type OperationalWeekDay = {
  date: string;
  weekdayLabel: string;
};

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

export function formatWeekdayLabel(date: string) {
  const parsedDate = new Date(`${date}T00:00:00Z`);
  const weekday = WEEKDAY_SHORT_LABELS[parsedDate.getUTCDay()];
  const day = `${parsedDate.getUTCDate()}`.padStart(2, '0');
  const month = `${parsedDate.getUTCMonth() + 1}`.padStart(2, '0');

  return `${weekday} ${day}/${month}`;
}

export function getUtcWeekStart(date: string) {
  const parsedDate = new Date(`${date}T00:00:00Z`);
  const dayOfWeek = parsedDate.getUTCDay();
  const offsetToMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;

  parsedDate.setUTCDate(parsedDate.getUTCDate() + offsetToMonday);

  return formatDateInputValue(parsedDate);
}

export function mapDateToOperationalWeek(date: string) {
  const parsedDate = new Date(`${date}T00:00:00Z`);
  const dayOfWeek = parsedDate.getUTCDay();

  if (dayOfWeek === 6) {
    parsedDate.setUTCDate(parsedDate.getUTCDate() - 1);
  }

  if (dayOfWeek === 0) {
    parsedDate.setUTCDate(parsedDate.getUTCDate() - 2);
  }

  return formatDateInputValue(parsedDate);
}

export function getOperationalWeekDays(date: string): OperationalWeekDay[] {
  const weekStart = getUtcWeekStart(date);

  return Array.from({ length: 5 }, (_, index) => {
    const currentDate = new Date(`${weekStart}T00:00:00Z`);
    currentDate.setUTCDate(currentDate.getUTCDate() + index);

    const currentDateValue = formatDateInputValue(currentDate);

    return {
      date: currentDateValue,
      weekdayLabel: formatWeekdayLabel(currentDateValue),
    };
  });
}

export function normalizeOperationalTimeBands(items: Array<{ start_time: string }>) {
  return [...new Set(items.map((item) => formatTimeBand(item.start_time)))].sort();
}

export function formatTimeBand(dateTime: string) {
  const parsedDate = new Date(dateTime);
  const hours = `${parsedDate.getUTCHours()}`.padStart(2, '0');
  const minutes = `${parsedDate.getUTCMinutes()}`.padStart(2, '0');

  return `${hours}:${minutes}`;
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
