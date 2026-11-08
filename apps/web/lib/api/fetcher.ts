import type { CsrfToken, Problem } from "./gen/models";

const API_PREFIX = "/api/v1";
const CSRF_COOKIE = "__Host-gk_csrf";
const CSRF_HEADER = "X-CSRF-Token";

export class ApiError extends Error {
  readonly status: number;
  readonly problem: Problem | null;

  constructor(status: number, problem: Problem | null) {
    super(problem?.title ?? `HTTP ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.problem = problem;
  }
}

const getBaseUrl = () => {
  if (typeof window === "undefined") {
    return process.env.API_INTERNAL_URL ?? "http://localhost:8091";
  }

  return "";
};

const withApiPrefix = (url: string) => {
  if (url.startsWith(`${API_PREFIX}/`) || url === API_PREFIX) {
    return url;
  }

  return `${API_PREFIX}${url.startsWith("/") ? url : `/${url}`}`;
};

const readCookie = (name: string): string | null => {
  if (typeof document === "undefined") {
    return null;
  }

  const match = document.cookie.match(
    new RegExp(`(?:^|; )${name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}=([^;]*)`),
  );

  return match ? decodeURIComponent(match[1]) : null;
};

const parseProblem = async (response: Response): Promise<Problem | null> => {
  const contentType = response.headers.get("content-type") ?? "";

  if (
    !contentType.includes("application/problem+json") &&
    !contentType.includes("application/json")
  ) {
    return null;
  }

  try {
    return (await response.json()) as Problem;
  } catch {
    return null;
  }
};

const toApiError = async (response: Response) =>
  new ApiError(response.status, await parseProblem(response));

let csrfRequest: Promise<string> | null = null;

const requestCsrfToken = async (): Promise<string> => {
  const response = await fetch(
    `${getBaseUrl()}${withApiPrefix("/auth/csrf")}`,
    {
      method: "GET",
      credentials: "include",
      headers: { Accept: "application/json" },
      cache: "no-store",
    },
  );

  if (!response.ok) {
    throw await toApiError(response);
  }

  const data = (await response.json()) as Partial<CsrfToken>;

  return data.csrfToken ?? readCookie(CSRF_COOKIE) ?? "";
};

/**
 * Returns the CSRF token for the current browser tab. Concurrent callers share
 * the same in-flight request, and the cookie is preferred when it already
 * exists. `force` is used after the server rejects a token as stale.
 */
const getCsrfToken = async (force = false): Promise<string> => {
  if (!force) {
    const cookieToken = readCookie(CSRF_COOKIE);

    if (cookieToken) {
      return cookieToken;
    }
  }

  if (!csrfRequest) {
    csrfRequest = requestCsrfToken().finally(() => {
      csrfRequest = null;
    });
  }

  return csrfRequest;
};

export const apiFetch = async <T>(
  url: string,
  options?: RequestInit,
): Promise<T> => {
  const method = (options?.method ?? "GET").toUpperCase();
  const isBrowser = typeof window !== "undefined";
  const needsCsrf = isBrowser && method !== "GET" && method !== "HEAD";
  const headers = new Headers(options?.headers);

  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  const target = `${getBaseUrl()}${withApiPrefix(url)}`;

  if (needsCsrf) {
    headers.set(CSRF_HEADER, await getCsrfToken());
  }

  const send = () =>
    fetch(target, {
      ...options,
      credentials: "include",
      headers,
    });

  let response = await send();

  if (needsCsrf && response.status === 403) {
    const problem = await parseProblem(response);

    if (problem?.code === "CSRF_FAILED") {
      headers.set(CSRF_HEADER, await getCsrfToken(true));
      response = await send();
    } else {
      throw new ApiError(response.status, problem);
    }
  }

  if (!response.ok) {
    throw await toApiError(response);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (
    contentType.includes("application/json") ||
    contentType.includes("+json")
  ) {
    return (await response.json()) as T;
  }

  return (await response.blob()) as T;
};
