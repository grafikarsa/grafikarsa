// Human labels for audit actions and target types. Unknown actions are shown
// as their raw value in monospace by the audit table.

export const ACTION_LABELS: Record<string, string> = {
  "auth.activate": "Aktivasi akun",
  "auth.activation_complete": "Menyelesaikan aktivasi",
  "auth.password_change": "Mengubah sandi",
  "auth.password_recover": "Pemulihan sandi",
  "auth.recovery_codes_regenerate": "Membuat ulang kode pemulihan",
  "user.create": "Membuat pengguna",
  "user.create_admin": "Membuat pengguna admin",
  "user.update": "Mengubah pengguna",
  "user.reset_password": "Reset sandi",
  "user.reissue_initial_password": "Menerbitkan ulang sandi awal",
  "user.deactivate": "Menonaktifkan pengguna",
  "user.reactivate": "Mengaktifkan pengguna",
  "user.block_content": "Mencabut akses konten",
  "user.unblock_content": "Memulihkan akses konten",
  "role.assign": "Memberi role",
  "role.revoke": "Mencabut role",
  "role.create": "Membuat role",
  "role.update": "Mengubah role",
  "role.delete": "Menghapus role",
  "role.permissions_update": "Mengubah izin role",
  "academic.class_create": "Membuat kelas",
  "academic.class_update": "Mengubah kelas",
  "academic.class_delete": "Menghapus kelas",
  "academic.major_create": "Membuat jurusan",
  "academic.major_update": "Mengubah jurusan",
  "academic.year_create": "Membuat tahun ajaran",
  "academic.year_set_current": "Menetapkan tahun ajaran berjalan",
};

const TARGET_TYPE_LABELS: Record<string, string> = {
  user: "Pengguna",
  role: "Role",
  major: "Jurusan",
  class: "Kelas",
  academic_year: "Tahun ajaran",
  content: "Konten",
};

/** Returns the Indonesian label for a known action, or null when unknown. */
export function actionLabel(action: string): string | null {
  return ACTION_LABELS[action] ?? null;
}

/** Returns the Indonesian label for a known target type, or the raw type. */
export function targetTypeLabel(type: string | null | undefined): string | null {
  if (!type) {
    return null;
  }
  return TARGET_TYPE_LABELS[type] ?? type;
}

/** Returns the first segment of a UUID for compact display. */
export function shortId(id: string): string {
  return id.slice(0, 8);
}
