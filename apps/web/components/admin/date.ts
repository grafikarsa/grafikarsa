const LONG_DATE = new Intl.DateTimeFormat("id-ID", {
  day: "numeric",
  month: "long",
  year: "numeric",
});

/** Parses an ISO date (YYYY-MM-DD) as a local date, avoiding UTC shifts. */
export function parseDateOnly(value: string): Date | undefined {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);

  if (!match) {
    return undefined;
  }

  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const date = new Date(year, month - 1, day);

  if (
    date.getFullYear() !== year ||
    date.getMonth() !== month - 1 ||
    date.getDate() !== day
  ) {
    return undefined;
  }

  return date;
}

/** Formats an ISO date as an Indonesian long date, e.g. "1 Juli 2026". */
export function formatDateLong(value: string): string {
  const date = parseDateOnly(value);

  if (!date) {
    return "—";
  }

  return LONG_DATE.format(date);
}

/** Converts a local date to an ISO date (YYYY-MM-DD). */
export function toDateOnly(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");

  return `${year}-${month}-${day}`;
}
