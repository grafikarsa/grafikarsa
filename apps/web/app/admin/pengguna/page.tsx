import type { Metadata } from "next";
import { Suspense } from "react";

import { UsersView } from "@/components/admin/users/users-view";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata: Metadata = {
  title: "Pengguna",
};

function UsersFallback() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-8 w-40" />
      <Skeleton className="h-4 w-72" />
      <Skeleton className="h-64 w-full" />
    </div>
  );
}

export default function AdminUsersPage() {
  return (
    <Suspense fallback={<UsersFallback />}>
      <UsersView />
    </Suspense>
  );
}
