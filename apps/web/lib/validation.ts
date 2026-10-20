import { z } from "zod";

// Skema validasi sisi klien.
//
// Aturan di sini mencerminkan validasi backend (Go) — bukan menggantikannya.
// Server tetap jadi sumber kebenaran; validasi klien hanya supaya kesalahan
// ketahuan lebih awal tanpa perjalanan bolak-balik.
//
// Keduanya harus dijaga sinkron. Kalau aturan server berubah, ubah juga di
// sini (dan sebaliknya).

/** Username: huruf kecil, angka, titik, underscore, strip; 3-30 karakter. */
export const usernameSchema = z
  .string()
  .min(3, "Username minimal 3 karakter.")
  .max(30, "Username maksimal 30 karakter.")
  .regex(
    /^[a-z0-9._-]+$/,
    "Username hanya boleh huruf kecil, angka, titik, underscore, dan strip.",
  );

/** Nama tampilan: 2-100 karakter setelah trimming. */
export const displayNameSchema = z
  .string()
  .trim()
  .min(2, "Nama minimal 2 karakter.")
  .max(100, "Nama maksimal 100 karakter.");

/** Bio: maksimal 500 karakter. */
export const bioSchema = z.string().max(500, "Bio maksimal 500 karakter.");

/** Formulir profil. */
export const profileSchema = z.object({
  name: displayNameSchema,
  bio: bioSchema,
  username: usernameSchema,
});

export type ProfileInput = z.infer<typeof profileSchema>;

/** Formulir login. Password minimal 10 karakter, sama seperti aturan server. */
export const loginSchema = z.object({
  username: usernameSchema,
  password: z.string().min(1, "Password wajib diisi."),
});
export type LoginInput = z.infer<typeof loginSchema>;

/**
 * Format ulang pesan error zxcvbn-style supaya bisa ditampilkan per-field.
 * zod v4 menambah issues dengan `path`; pesan pertama per field dipakai.
 */
export function firstIssueMessage(error: z.ZodError): string {
  const issue = error.issues[0];
  return issue?.message ?? "Data tidak valid.";
}

/**
 * Pesan error per-field (path -> pesan pertama).
 * Dipakai form yang menampilkan pesan di bawah masing-masing input.
 */
export function fieldErrors(error: z.ZodError): Record<string, string> {
  const out: Record<string, string> = {};
  for (const issue of error.issues) {
    const key = issue.path.length > 0 ? String(issue.path[0]) : "_form";
    if (!(key in out)) out[key] = issue.message;
  }
  return out;
}
