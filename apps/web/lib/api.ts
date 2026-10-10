import { useAuthStore, type AuthUser } from "@/lib/auth-store";
import type { ApiError, ApiSuccess } from "./api-client";

export class ApiErrorWithCode extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.code = code;
  }
}

function authHeaders(): HeadersInit {
  const { accessToken, csrfToken } = useAuthStore.getState();
  const h: Record<string, string> = {};
  if (accessToken) h.Authorization = `Bearer ${accessToken}`;
  if (csrfToken) h["X-CSRF-Token"] = csrfToken;
  return h;
}

async function parse<T>(res: Response): Promise<T> {
  let json: ApiSuccess<T> | ApiError;
  try {
    json = (await res.json()) as ApiSuccess<T> | ApiError;
  } catch {
    throw new ApiErrorWithCode("NETWORK_ERROR", `Server error ${res.status}.`);
  }
  if (!json.success) throw new ApiErrorWithCode(json.error.code, json.error.message);
  return json.data;
}

let refreshing: Promise<boolean> | null = null;

async function refreshSession(): Promise<boolean> {
  if (!refreshing) {
    refreshing = fetch("/api/v1/auth/refresh", { method: "POST" })
      .then(async (res) => {
        if (!res.ok) return false;
        const json = (await res.json()) as ApiSuccess<{ access_token: string; csrf_token: string }> | ApiError;
        if (!json.success) return false;
        useAuthStore.getState().setSession(json.data.access_token, json.data.csrf_token);
        return true;
      })
      .catch(() => false)
      .finally(() => {
        refreshing = null;
      });
  }
  return refreshing;
}

async function request<T>(path: string, init?: RequestInit, retry = true): Promise<T> {
  const res = await fetch(`/api/${path.replace(/^\//, "")}`, init);
  if (res.status === 401 && retry) {
    // Access token di memori hilang/expired -> coba refresh sekali lalu ulangi.
    if (await refreshSession()) {
      return request<T>(path, { ...init, headers: { ...(init?.headers ?? {}), ...authHeaders() } }, false);
    }
  }
  return parse<T>(res);
}

export async function login(username: string, password: string, captchaToken?: string) {
  const res = await fetch("/api/v1/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password, captcha_token: captchaToken }),
  });
  const json = (await res.json()) as
    | ApiSuccess<{ access_token: string; csrf_token: string; user: AuthUser }>
    | ApiError;
  if (!json.success) throw new ApiErrorWithCode(json.error.code, json.error.message);
  const { access_token, csrf_token, user } = json.data;
  useAuthStore.getState().setSession(access_token, csrf_token);
  useAuthStore.getState().setUser(user);
  return user;
}

export async function logout() {
  try {
    await apiSend("v1/auth/logout", "POST");
  } catch {
    /* abaikan */
  }
  useAuthStore.getState().logout();
}

export async function apiGet<T>(path: string): Promise<T> {
  return request<T>(path, { headers: authHeaders(), cache: "no-store" });
}

export async function apiSend<T>(path: string, method: string, body?: unknown): Promise<T> {
  return request<T>(path, {
    method,
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

// bootstrapSession: dipanggil sekali saat app mount. Menukar cookie refresh
// menjadi access token (jika ada), lalu memuat user. Return null bila tamu.
export async function bootstrapSession(): Promise<AuthUser | null> {
  const store = useAuthStore.getState();
  if (!store.accessToken) {
    const ok = await refreshSession();
    if (!ok) {
      store.setUser(null);
      return null;
    }
  }
  try {
    const me = await apiGet<AuthUser>("v1/me");
    useAuthStore.getState().setUser(me);
    return me;
  } catch {
    useAuthStore.getState().logout();
    return null;
  }
}
