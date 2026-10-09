export type ApiSuccess<T> = { success: true; data: T; meta?: unknown; request_id?: string };
export type ApiError = {
  success: false;
  error: { code: string; message: string; details?: { field: string; message: string }[] };
  request_id?: string;
};

export async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(`/api/${path.replace(/^\//, "")}`);
  const json = (await res.json()) as ApiSuccess<T> | ApiError;
  if (!json.success) throw new Error(json.error.message);
  return json.data;
}
