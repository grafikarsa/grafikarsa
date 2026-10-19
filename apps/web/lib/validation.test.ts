import { describe, expect, it } from "vitest";
import {
  bioSchema,
  displayNameSchema,
  fieldErrors,
  firstIssueMessage,
  loginSchema,
  profileSchema,
  usernameSchema,
} from "./validation";

// Skema validasi klien harus mencerminkan aturan backend (Go). Kalau aturan
// server berubah, tes ini harus ikut diperbarui — dan sebaliknya.
//
// Aturan backend yang dicerminkan:
//   users.username_format : ^[a-z0-9._-]{3,30}$   (migrations/00003_users_auth.sql)
//   users.name            : 2..100 karakter       (dto/validator)
//   users.bio             : maksimal 500 karakter
describe("usernameSchema", () => {
  it("menerima username valid", () => {
    for (const u of ["budi", "budi.santoso", "user_01", "a-b-c", "123"]) {
      expect(usernameSchema.safeParse(u).success).toBe(true);
    }
  });

  it("menolak username invalid", () => {
    for (const u of ["ab", "Budi", "budi santoso", "budi@santoso", "x".repeat(31), ""]) {
      expect(usernameSchema.safeParse(u).success).toBe(false);
    }
  });
});

describe("displayNameSchema", () => {
  it("menerima nama 2-100 karakter (setelah trim)", () => {
    expect(displayNameSchema.safeParse("Budi").success).toBe(true);
    expect(displayNameSchema.safeParse("  Budi  ").success).toBe(true);
    expect(displayNameSchema.safeParse("a".repeat(100)).success).toBe(true);
  });

  it("menolak nama terlalu pendek atau panjang", () => {
    expect(displayNameSchema.safeParse("a").success).toBe(false);
    expect(displayNameSchema.safeParse("   ").success).toBe(false);
    expect(displayNameSchema.safeParse("a".repeat(101)).success).toBe(false);
  });
});

describe("bioSchema", () => {
  it("menerima bio sampai 500 karakter", () => {
    expect(bioSchema.safeParse("a".repeat(500)).success).toBe(true);
    expect(bioSchema.safeParse("a".repeat(501)).success).toBe(false);
  });
});

describe("profileSchema", () => {
  it("lolos untuk profil yang benar", () => {
    const r = profileSchema.safeParse({ name: "Budi", bio: "Siswa RPL", username: "budi" });
    expect(r.success).toBe(true);
  });

  it("menolak profil dengan username salah dan memberi pesan per-field", () => {
    const r = profileSchema.safeParse({ name: "Budi", bio: "", username: "B" });
    expect(r.success).toBe(false);
    if (!r.success) {
      const errs = fieldErrors(r.error);
      expect(errs.username).toBeTruthy();
      expect(errs.name).toBeUndefined();
    }
  });
});

describe("loginSchema", () => {
  it("lolos untuk username + password", () => {
    expect(loginSchema.safeParse({ username: "budi", password: "rahasia123" }).success).toBe(true);
  });

  it("menolak username invalid atau password kosong", () => {
    expect(loginSchema.safeParse({ username: "B", password: "x" }).success).toBe(false);
    expect(loginSchema.safeParse({ username: "budi", password: "" }).success).toBe(false);
  });
});

describe("firstIssueMessage", () => {
  it("mengembalikan pesan pertama yang bisa ditampilkan", () => {
    const r = profileSchema.safeParse({ name: "", bio: "", username: "" });
    expect(r.success).toBe(false);
    if (!r.success) {
      const msg = firstIssueMessage(r.error);
      expect(msg.length).toBeGreaterThan(0);
    }
  });
});
