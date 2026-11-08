"use client";

import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { CircleAlert } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";

import NotFound from "@/app/not-found";
import { AdminPageHeader } from "@/components/admin/admin-page-header";
import { apiErrorMessage } from "@/components/admin/form-error";
import { ImportHistory } from "@/components/admin/import/import-history";
import { ImportJobDetail } from "@/components/admin/import/import-job-detail";
import { ImportUploadCard } from "@/components/admin/import/import-upload-card";
import { QueryError } from "@/components/admin/query-error";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import {
  getGetAdminImportQueryKey,
  getGetAdminImportsQueryKey,
  useGetAdminImport,
  useGetAdminImports,
} from "@/lib/api/gen/endpoints";
import type { ImportJob } from "@/lib/api/gen/models";
import { ApiError } from "@/lib/api/fetcher";
import {
  ADMIN_PERMISSIONS,
  can,
  canGlobally,
} from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

function JobSkeleton() {
  return (
    <div className="space-y-3 rounded-xl border p-4">
      <Skeleton className="h-5 w-48" />
      <Skeleton className="h-4 w-72" />
      <Skeleton className="h-24 w-full" />
    </div>
  );
}

export function ImportView() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const { session, status } = useSession();

  const jobParam = searchParams.get("job") ?? "";
  const authenticated = status === "authenticated";
  const canImport = authenticated && can(session, ADMIN_PERMISSIONS.importCsv);
  const canImportTeachers =
    authenticated && canGlobally(session, ADMIN_PERMISSIONS.importCsv);

  const selectJob = useCallback(
    (id: string | null) => {
      const next = new URLSearchParams(window.location.search);

      if (id) {
        next.set("job", id);
      } else {
        next.delete("job");
      }

      const query = next.toString();
      router.replace(query ? `${pathname}?${query}` : pathname, {
        scroll: false,
      });
    },
    [pathname, router],
  );

  const historyQuery = useGetAdminImports({
    query: { enabled: canImport },
  });

  const jobQuery = useGetAdminImport(jobParam, {
    query: {
      enabled: canImport && Boolean(jobParam),
      refetchInterval: (query) =>
        query.state.data?.status === "committing" ? 2000 : false,
    },
  });

  const handleUploaded = (job: ImportJob) => {
    queryClient.setQueryData(getGetAdminImportQueryKey(job.id), job);
    void queryClient.invalidateQueries({
      queryKey: getGetAdminImportsQueryKey(),
    });
    selectJob(job.id);
    toast.success(
      job.status === "invalid"
        ? "Berkas diperiksa. Ada kesalahan yang perlu diperbaiki."
        : "Berkas valid dan siap diimpor.",
    );
  };

  if (status !== "authenticated") {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-4 w-72" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!canImport) {
    return <NotFound />;
  }

  const jobError =
    jobQuery.error instanceof ApiError && jobQuery.error.status === 404
      ? "Impor tidak ditemukan atau bukan milikmu."
      : apiErrorMessage(jobQuery.error, "Impor tidak dapat dimuat.");

  return (
    <div className="space-y-4">
      <AdminPageHeader
        title="Impor"
        description="Buat akun siswa, guru, dan tendik secara massal dari berkas CSV."
      />

      <ImportUploadCard
        canImportTeachers={canImportTeachers}
        onUploaded={handleUploaded}
      />

      {jobParam ? (
        jobQuery.isPending ? (
          <JobSkeleton />
        ) : jobQuery.isError ? (
          <Alert variant="destructive">
            <CircleAlert aria-hidden="true" />
            <AlertTitle>Impor tidak dapat ditampilkan</AlertTitle>
            <AlertDescription>{jobError}</AlertDescription>
          </Alert>
        ) : jobQuery.data ? (
          <ImportJobDetail key={jobQuery.data.id} job={jobQuery.data} />
        ) : null
      ) : null}

      {historyQuery.isError ? (
        <QueryError
          title="Gagal memuat riwayat impor"
          message={apiErrorMessage(
            historyQuery.error,
            "Riwayat impor tidak dapat dimuat.",
          )}
          onRetry={() => {
            void historyQuery.refetch();
          }}
          isRetrying={historyQuery.isFetching}
        />
      ) : (
        <ImportHistory
          jobs={historyQuery.data?.items ?? []}
          selectedJobId={jobParam || null}
          onSelect={(id) => selectJob(id)}
          isLoading={historyQuery.isPending}
        />
      )}
    </div>
  );
}
