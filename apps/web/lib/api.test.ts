import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiErrorWithCode, apiGet, apiGetWithMeta, apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

// Test alur request API — bagian FE yang paling rawan gagal diam-diam:
// kalau refresh token tidak bekerja, seluruh aplikasi yang butuh auth
// ikut mati tanpa pesan yang jelas.

type Resp = {
  status: number;
  body: unknown;
};

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

const ok = <T>(data: T, meta?: unknown) => ({
  success: true,
  data,
  ...(meta ? { meta } : {}),
});

describe("api client", () => {
  let calls: {
    url: string;
    method: string;
    headers: HeadersInit | undefined;
    body?: string;
  }[] = [];
  let responses: Resp[] = [];

  beforeEach(() => {
    calls = [];
    responses = [];
    useAuthStore.getState().setSession("token-abc", "csrf-xyz");
    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string, init?: RequestInit) => {
        calls.push({
          url,
          method: init?.method ?? "GET",
          headers: init?.headers,
          body: typeof init?.body === "string" ? init.body : undefined,
        });
        const next = responses.shift();
        if (!next)
          return jsonResponse(500, {
            success: false,
            error: { code: "X", message: "no response" },
          });
        return jsonResponse(next.status, next.body);
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    useAuthStore.getState().logout();
  });

  it("mengirim header Authorization dari access token di memori", async () => {
    responses.push({ status: 200, body: ok({ nama: "Budi" }) });
    await apiGet<{ nama: string }>("v1/me");
    const headers = calls[0].headers as Record<string, string>;
    expect(headers.Authorization).toBe("Bearer token-abc");
    expect(headers["X-CSRF-Token"]).toBe("csrf-xyz");
  });

  it("tidak mengirim header Authorization kalau belum login", async () => {
    useAuthStore.getState().logout();
    responses.push({ status: 200, body: ok({}) });
    await apiGet("v1/feed");
    const headers = (calls[0].headers ?? {}) as Record<string, string>;
    expect(headers.Authorization).toBeUndefined();
  });

  it("melempar ApiErrorWithCode dengan kode dari server", async () => {
    responses.push({
      status: 403,
      body: { success: false, error: { code: "FORBIDDEN_SCOPE", message: "Di luar jurusanmu." } },
    });
    await expect(apiGet("v1/admin/users")).rejects.toMatchObject({
      code: "FORBIDDEN_SCOPE",
      message: "Di luar jurusanmu.",
    });
  });

  it("memakai nama endpoint tanpa slash ganda", async () => {
    responses.push({ status: 200, body: ok({}) });
    await apiGet("/v1/feed");
    expect(calls[0].url).toBe("/api/v1/feed");
  });

  it("mencoba refresh sekali lalu mengulang request saat 401", async () => {
    // 1) request pertama 401, 2) refresh 200, 3) ulangan request 200
    responses.push({
      status: 401,
      body: { success: false, error: { code: "UNAUTHORIZED", message: "Sesi berakhir." } },
    });
    responses.push({
      status: 200,
      body: ok({ access_token: "token-baru", csrf_token: "csrf-baru" }),
    });
    responses.push({ status: 200, body: ok({ nama: "Budi" }) });

    const hasil = await apiGet<{ nama: string }>("v1/me");

    expect(hasil.nama).toBe("Budi");
    expect(calls.map((c) => c.url)).toEqual(["/api/v1/me", "/api/v1/auth/refresh", "/api/v1/me"]);
    // Token baru harus tersimpan di store.
    expect(useAuthStore.getState().accessToken).toBe("token-baru");
  });

  it("tidak mengulang request kalau refresh gagal", async () => {
    responses.push({
      status: 401,
      body: { success: false, error: { code: "UNAUTHORIZED", message: "Sesi berakhir." } },
    });
    responses.push({
      status: 401,
      body: { success: false, error: { code: "UNAUTHORIZED", message: "Sesi berakhir." } },
    });

    await expect(apiGet("v1/me")).rejects.toBeInstanceOf(ApiErrorWithCode);
    // Hanya satu percobaan request + satu refresh — tidak loops.
    expect(calls.length).toBe(2);
  });

  it("apiGetWithMeta mengembalikan data sekaligus meta", async () => {
    responses.push({
      status: 200,
      body: ok([{ id: "1" }], { page: 2, per_page: 20, total: 47 }),
    });
    const hasil = await apiGetWithMeta<{ id: string }>("v1/admin/users");
    expect(hasil.data).toHaveLength(1);
    expect(hasil.meta.total).toBe(47);
    expect(hasil.meta.page).toBe(2);
  });

  it("apiGetWithMeta memakai meta default kalau server tidak mengirimnya", async () => {
    responses.push({ status: 200, body: ok([]) });
    const hasil = await apiGetWithMeta("v1/admin/users");
    expect(hasil.data).toEqual([]);
    expect(hasil.meta.total).toBe(0);
  });

  it("apiSend mengirim body JSON dan method yang diminta", async () => {
    responses.push({ status: 200, body: ok({ id: "1" }) });
    await apiSend("v1/admin/users", "POST", { username: "budi" });
    expect(calls[0].method).toBe("POST");
    const headers = calls[0].headers as Record<string, string>;
    expect(headers["Content-Type"]).toBe("application/json");
    expect(headers.Authorization).toBe("Bearer token-abc");
    expect(JSON.parse(calls[0].body ?? "{}")).toEqual({ username: "budi" });
  });
});
