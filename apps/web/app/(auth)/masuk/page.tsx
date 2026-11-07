import type { Metadata } from "next";
import { Suspense } from "react";

import { LoginForm } from "@/components/auth/login-form";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata: Metadata = {
  title: "Masuk",
};

export default function MasukPage() {
  return (
    <Suspense
      fallback={
        <div className="space-y-4 rounded-xl p-4 ring-1 ring-foreground/10">
          <Skeleton className="h-6 w-24" />
          <Skeleton className="h-4 w-56" />
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-9 w-full" />
        </div>
      }
    >
      <LoginForm />
    </Suspense>
  );
}
