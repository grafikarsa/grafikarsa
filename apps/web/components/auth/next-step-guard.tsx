"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";

import { requiredPathFor, useSession } from "@/lib/auth/session";

/**
 * Keeps accounts that still need activation or a password change on the
 * required page, wherever they are in the app.
 */
export function NextStepGuard() {
  const pathname = usePathname();
  const router = useRouter();
  const { session, status } = useSession();

  useEffect(() => {
    if (status !== "authenticated") {
      return;
    }

    const requiredPath = requiredPathFor(session.nextStep);

    if (requiredPath && pathname !== requiredPath) {
      router.replace(requiredPath);
    }
  }, [pathname, router, session, status]);

  return null;
}
