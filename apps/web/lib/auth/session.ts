"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";

import {
  getGetAuthSessionQueryKey,
  postAuthLogout,
  useGetAuthSession,
} from "@/lib/api/gen/endpoints";
import type { NextStep, Session } from "@/lib/api/gen/models";

export const sessionQueryKey = getGetAuthSessionQueryKey();

export type SessionState =
  | { status: "loading"; session: null }
  | { status: "anonymous"; session: null }
  | { status: "authenticated"; session: Session };

export function useSession(): SessionState {
  const query = useGetAuthSession({
    query: {
      retry: false,
    },
  });

  if (query.data) {
    return { status: "authenticated", session: query.data };
  }

  if (query.isPending) {
    return { status: "loading", session: null };
  }

  return { status: "anonymous", session: null };
}

export function requiredPathFor(nextStep: NextStep): string | null {
  switch (nextStep) {
    case "activate":
      return "/aktivasi";
    case "change_password":
      return "/ganti-sandi";
    default:
      return null;
  }
}

export function useLogout() {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationFn: () => postAuthLogout(),
    onSuccess: () => {
      queryClient.clear();
      router.push("/");
    },
  });
}
