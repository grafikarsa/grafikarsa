import { formatDistanceToNow } from "date-fns";
import { id as localeID } from "date-fns/locale";

import type {
  AccountStatus,
  AccountType,
  AdminUserDetailPasswordKind,
} from "@/lib/api/gen/models";

export const ACCOUNT_TYPE_LABELS: Record<AccountType, string> = {
  student: "Siswa",
  teacher: "Guru",
  staff: "Tendik",
};

export const ACCOUNT_STATUS_LABELS: Record<AccountStatus, string> = {
  active: "Aktif",
  alumni: "Alumni",
  transferred: "Mutasi",
  inactive: "Nonaktif",
};

export const ACCOUNT_STATUS_VARIANTS: Record<
  AccountStatus,
  "default" | "secondary" | "outline" | "destructive"
> = {
  active: "default",
  alumni: "secondary",
  transferred: "outline",
  inactive: "destructive",
};

export const PASSWORD_KIND_LABELS: Record<
  AdminUserDetailPasswordKind,
  string
> = {
  initial: "Sandi awal",
  temporary: "Sandi sementara",
  user: "Sandi sendiri",
};

const DATE_TIME = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "medium",
  timeStyle: "short",
});

const DATE_ONLY = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "long",
});

/** Formats an ISO timestamp as an Indonesian date and time. */
export function formatDateTime(value: string | null | undefined): string {
  if (!value) {
    return "—";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "—";
  }

  return DATE_TIME.format(date);
}

/** Formats an ISO date as an Indonesian long date. */
export function formatDate(value: string | null | undefined): string {
  if (!value) {
    return "—";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "—";
  }

  return DATE_ONLY.format(date);
}

/** Formats an ISO timestamp relative to now, e.g. "3 jam lalu". */
export function formatRelativeTime(
  value: string | null | undefined,
): string {
  if (!value) {
    return "Belum pernah";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "—";
  }

  return formatDistanceToNow(date, { addSuffix: true, locale: localeID });
}

/** Builds up to two initials from a display name. */
export function initialsOf(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);

  if (words.length === 0) {
    return "?";
  }

  return words
    .slice(0, 2)
    .map((word) => word.charAt(0).toUpperCase())
    .join("");
}

/** Returns the login identifier shown to admins: NISN or NIP/ID staf. */
export function loginIdentifier(user: {
  accountType: AccountType;
  nisn: string | null;
  staffNumber: string | null;
}): string | null {
  if (user.accountType === "student") {
    return user.nisn ? `NISN ${user.nisn}` : null;
  }

  return user.staffNumber ? `NIP/ID staf ${user.staffNumber}` : null;
}
