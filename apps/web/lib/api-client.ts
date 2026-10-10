export type ApiSuccess<T> = { success: true; data: T; meta?: unknown; request_id?: string };
export type ApiError = {
  success: false;
  error: { code: string; message: string; details?: { field: string; message: string }[] };
  request_id?: string;
};


