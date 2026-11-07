import type { FieldValues, Path, UseFormReturn } from "react-hook-form";

import { ApiError } from "@/lib/api/fetcher";

const FALLBACK_MESSAGE = "Terjadi kesalahan. Coba lagi.";

/** Converts API field names such as `short_name` to form field names. */
function toFormField(field: string): string {
  return field.replace(/_([a-z])/g, (_, letter: string) =>
    letter.toUpperCase(),
  );
}

/** Returns a friendly message for an API error. */
export function apiErrorMessage(
  error: unknown,
  fallback = FALLBACK_MESSAGE,
): string {
  if (error instanceof ApiError) {
    return error.problem?.detail ?? error.problem?.title ?? fallback;
  }

  return fallback;
}

/**
 * Applies `VALIDATION_FAILED` problem details to the matching form fields and
 * returns a form-level message when the error cannot be attached to a field.
 */
export function applyApiError<T extends FieldValues>(
  form: UseFormReturn<T>,
  error: unknown,
): string | null {
  if (!(error instanceof ApiError)) {
    return FALLBACK_MESSAGE;
  }

  const problem = error.problem;

  if (!problem) {
    return FALLBACK_MESSAGE;
  }

  if (problem.code === "VALIDATION_FAILED" && problem.errors?.length) {
    const values = form.getValues() as Record<string, unknown>;
    let applied = false;

    for (const item of problem.errors) {
      const field = toFormField(item.field);

      if (field in values) {
        form.setError(field as Path<T>, {
          type: "server",
          message: item.message,
        });
        applied = true;
      }
    }

    if (applied) {
      return null;
    }
  }

  if (problem.code === "CONFLICT") {
    return (
      problem.detail ?? problem.title ?? "Data dengan nilai yang sama sudah ada."
    );
  }

  return problem.detail ?? problem.title ?? FALLBACK_MESSAGE;
}
