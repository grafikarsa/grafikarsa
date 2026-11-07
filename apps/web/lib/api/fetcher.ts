import type { Problem } from "./gen/models";

const API_PREFIX = "/api/v1";

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

export const apiFetch = async <T>(
  url: string,
  options?: RequestInit,
): Promise<T> => {
  const headers = new Headers(options?.headers);

  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  const response = await fetch(`${getBaseUrl()}${withApiPrefix(url)}`, {
    ...options,
    credentials: "include",
    headers,
  });

  if (!response.ok) {
    let problem: Problem | null = null;
    const contentType = response.headers.get("content-type") ?? "";

    if (
      contentType.includes("application/problem+json") ||
      contentType.includes("application/json")
    ) {
      try {
        problem = (await response.json()) as Problem;
      } catch {
        problem = null;
      }
    }

    throw new ApiError(response.status, problem);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
};
