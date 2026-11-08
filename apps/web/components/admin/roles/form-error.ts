import type { FieldValues, Path, UseFormReturn } from "react-hook-form";

import { apiErrorMessage } from "@/components/admin/form-error";
import { ApiError } from "@/lib/api/fetcher";

const FORBIDDEN_MESSAGE = "Kamu tidak berwenang mengelola role.";

/**
 * Applies a role-management problem response to a form. Field errors are
 * attached when possible; otherwise a form-level message is returned.
 */
export function applyRoleFormError<T extends FieldValues>(
  form: UseFormReturn<T>,
  error: unknown,
  fallback: string,
): string | null {
  if (!(error instanceof ApiError)) {
    return apiErrorMessage(error, fallback);
  }

  if (error.status === 404) {
    return FORBIDDEN_MESSAGE;
  }

  const problem = error.problem;

  if (problem?.code === "VALIDATION_FAILED" && problem.errors?.length) {
    let applied = false;
    let formMessage: string | null = null;

    for (const item of problem.errors) {
      if (item.field === "slug" || item.field === "name") {
        form.setError(item.field as Path<T>, {
          type: "server",
          message: item.message,
        });
        applied = true;
      } else {
        formMessage = item.message;
      }
    }

    if (formMessage) {
      return formMessage;
    }

    if (applied) {
      return null;
    }
  }

  if (problem?.code === "CONFLICT") {
    return problem.detail ?? problem.title ?? fallback;
  }

  return apiErrorMessage(error, fallback);
}

/** Returns a friendly message for a role mutation without form fields. */
export function roleMutationErrorMessage(
  error: unknown,
  fallback: string,
): string {
  if (error instanceof ApiError && error.status === 404) {
    return FORBIDDEN_MESSAGE;
  }

  return apiErrorMessage(error, fallback);
}
