import type { ErrorCode, Schemas } from "@/lib/api-contract";

// Bentuk envelope dipakai bersama oleh BFF client (lib/api.ts) dan komponen.
// Tipe diturunkan dari kontrak OpenAPI supaya perubahan backend terdeteksi.
export type ApiSuccess<T> = Schemas["SuccessEnvelope"] & {
  data: T;
  meta?: Schemas["PaginationMeta"] | null;
};

export type ApiError = {
  success: false;
  error: {
    code: ErrorCode | (string & {});
    message: string;
    details?: { field: string; message: string }[];
  };
  request_id?: string;
};
