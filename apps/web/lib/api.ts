import { useAuthStore } from "@/lib/auth-store";
import type { ApiError, ApiSuccess } from "./api-client";

export class ApiErrorWithCode extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.code = code;
  }
}

async function raw<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/${path.replace(/^\//, "")}`, init);
  let json: ApiSuccess<T> | ApiError;
  try {
    json = (await res.json()) as ApiSuccess<T> | ApiError;
  } catch {
    throw new ApiErrorWithCode("NETWORK_ERROR", `Server error ${res.status}.`);
  }
  if (!json.success) throw new ApiErrorWithCode(json.error.code, json.error.message);
  return json.data;
}

export async function login(username: string, password: string) {
  const res = await fetch("/api/v1/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  const json = await res.json();
  if (!json.success) throw new Error(json.error.message);
  const { access_token, csrf_token, user } = json.data;
  useAuthStore.getState().setAuth(access_token, csrf_token, user);
  return user;
}

export function authHeaders(): HeadersInit {
  const { accessToken, csrfToken } = useAuthStore.getState();
  const h: Record<string, string> = {};
  if (accessToken) h.Authorization = `Bearer ${accessToken}`;
  if (csrfToken) h["X-CSRF-Token"] = csrfToken;
  return h;
}

export async function apiGet<T>(path: string): Promise<T> {
  return raw<T>(path, { headers: authHeaders() });
}

export async function apiSend<T>(path: string, method: string, body?: unknown): Promise<T> {
  return raw<T>(path, {
    method,
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: body ? JSON.stringify(body) : undefined,
  });
}

export async function logout() {
  try {
    await apiSend("v1/auth/logout", "POST");
  } catch {
    /* abaikan */
  }
  useAuthStore.getState().logout();
}
