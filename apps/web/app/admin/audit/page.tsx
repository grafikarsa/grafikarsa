import type { Metadata } from "next";
import { Suspense } from "react";

import { AuditView } from "@/components/admin/audit/audit-view";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata: Metadata = {
  title: "Audit",
};

export default function AdminAuditPage() {
  return (
    <Suspense
      fallback={
        <div className="space-y-4">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-4 w-72" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      }
    >
      <AuditView />
    </Suspense>
  );
}
