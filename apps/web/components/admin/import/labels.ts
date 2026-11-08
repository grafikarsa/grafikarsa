import type { ImportKind, ImportStatus } from "@/lib/api/gen/models";

export const IMPORT_KIND_LABELS: Record<ImportKind, string> = {
  students: "Siswa",
  teachers: "Guru",
  staff: "Tendik",
};

export const IMPORT_STATUS_LABELS: Record<ImportStatus, string> = {
  validating: "Memeriksa",
  ready: "Siap diimpor",
  invalid: "Tidak valid",
  committing: "Sedang diproses",
  committed: "Selesai",
  failed: "Gagal",
  cancelled: "Dibatalkan",
};

export const IMPORT_STATUS_VARIANTS: Record<
  ImportStatus,
  "default" | "secondary" | "outline" | "destructive"
> = {
  validating: "secondary",
  ready: "default",
  invalid: "destructive",
  committing: "secondary",
  committed: "outline",
  failed: "destructive",
  cancelled: "secondary",
};

/** Friendly labels for the CSV columns reported by the API. */
export const IMPORT_FIELD_LABELS: Record<string, string> = {
  header: "Judul kolom",
  file: "Berkas",
  nama: "Nama",
  nisn: "NISN",
  nip: "NIP",
  kelas: "Kelas",
  jurusan: "Jurusan",
  tahun_masuk: "Tahun masuk",
};

export const IMPORT_TEMPLATES: Record<ImportKind, string> = {
  students: "/templates/impor-siswa.csv",
  teachers: "/templates/impor-guru.csv",
  staff: "/templates/impor-tendik.csv",
};

export const IMPORT_FORMAT_HELP: Record<ImportKind, string[]> = {
  students: [
    "Kolom: nisn, nama, kelas, dan tahun_masuk (opsional).",
    "Kelas memakai nama kelas pada tahun ajaran berjalan, misalnya XI RPL B.",
    "NISN harus terdiri dari 10 digit angka.",
  ],
  teachers: [
    "Kolom: nip, nama, dan jurusan (opsional).",
    "Kode jurusan dipisahkan dengan tanda |, misalnya RPL|TKJ.",
    "NIP wajib diisi dan harus unik.",
  ],
  staff: [
    "Kolom: nip dan nama.",
    "NIP boleh dikosongkan untuk tenaga kependidikan tanpa NIP.",
  ],
};

export const MAX_IMPORT_FILE_BYTES = 2 * 1024 * 1024;

export function importKindLabel(value: string): string {
  return IMPORT_KIND_LABELS[value as ImportKind] ?? value;
}

export function importFieldLabel(value: string): string {
  return IMPORT_FIELD_LABELS[value] ?? value;
}

/** Mirrors the API class-name normalization so byClass keys can be matched. */
export function normalizeClassName(value: string): string {
  return value.trim().toLowerCase().replace(/\s+/g, " ");
}

export function formatFileSize(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`;
  }

  if (bytes < 1024 * 1024) {
    return `${Math.round(bytes / 1024)} KB`;
  }

  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
