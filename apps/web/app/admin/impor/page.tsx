import type { Metadata } from "next";
import { Suspense } from "react";

import { ImportView } from "@/components/admin/import/import-view";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata: Metadata = {
  title: "Impor",
};

function ImportFallback() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-8 w-40" />
      <Skeleton className="h-4 w-72" />
      <Skeleton className="h-64 w-full" />
    </div>
  );
}

export default function AdminImportPage() {
  return (
    <Suspense fallback={<ImportFallback />}>
      <ImportView />
    </Suspense>
  );
}
