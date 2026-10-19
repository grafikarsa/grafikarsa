"use client";

import { useEffect } from "react";
import { bootstrapSession } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

// Menukar cookie refresh (__Host-rt) menjadi access token di memori saat app
// dibuka/di-reload, lalu memuat user. Tanpa ini, reload = logout.
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const setLoading = useAuthStore((s) => s.setLoading);

  useEffect(() => {
    bootstrapSession()
      .catch(() => null)
      .finally(() => setLoading(false));
  }, [setLoading]);

  return <>{children}</>;
}
