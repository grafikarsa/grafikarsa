"use client";

import { useMemo, useState, type FormEvent } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Inbox, Search } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

import NotFound from "@/app/not-found";
import { AuditDetailSheet } from "@/components/admin/audit/audit-detail-sheet";
import {
  actionLabel,
  shortId,
  targetTypeLabel,
} from "@/components/admin/audit/audit-labels";
import { AdminPageHeader } from "@/components/admin/admin-page-header";
import { parseDateOnly } from "@/components/admin/date";
import { DatePicker } from "@/components/admin/date-picker";
import { apiErrorMessage } from "@/components/admin/form-error";
import { QueryError } from "@/components/admin/query-error";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { getAdminAuditLogs } from "@/lib/api/gen/endpoints";
import type {
  AuditLogEntry,
  GetAdminAuditLogsParams,
} from "@/lib/api/gen/models";
import { ApiError } from "@/lib/api/fetcher";
import { ADMIN_PERMISSIONS, can } from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

const PAGE_SIZE = 25;
const ALL_ACTIONS = "__all__";

const UUID_PATTERN =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

const DATE_TIME = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "medium",
  timeStyle: "short",
});

const ABSOLUTE_DATE_TIME = new Intl.DateTimeFormat("id-ID", {
  dateStyle: "full",
  timeStyle: "long",
});

type Filters = {
  actionPrefix: string;
  from: string;
  to: string;
  targetId: string;
};

function readFilters(params: URLSearchParams): Filters {
  return {
    actionPrefix: params.get("actionPrefix") ?? "",
    from: params.get("from") ?? "",
    to: params.get("to") ?? "",
    targetId: params.get("targetId") ?? "",
  };
}

/** Converts a date-only filter to an ISO timestamp. The end date is exclusive. */
function dateFilterToISO(value: string, endExclusive = false): string | undefined {
  const date = parseDateOnly(value);

  if (!date) {
    return undefined;
  }

  if (endExclusive) {
    date.setDate(date.getDate() + 1);
  }

  return date.toISOString();
}

function formatDateTime(value: string): string {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return DATE_TIME.format(date);
}

function formatAbsoluteDateTime(value: string): string {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return ABSOLUTE_DATE_TIME.format(date);
}

function FiltersSkeleton() {
  return (
    <div className="mb-4 grid gap-4 rounded-xl border p-4 md:grid-cols-2 xl:grid-cols-4">
      {Array.from({ length: 4 }).map((_, index) => (
        <div key={index} className="space-y-1.5">
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-9 w-full" />
        </div>
      ))}
    </div>
  );
}

function TableSkeleton() {
  return (
    <>
      {Array.from({ length: 6 }).map((_, index) => (
        <TableRow key={index}>
          <TableCell>
            <Skeleton className="h-4 w-28" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-32" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-36" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-8 w-16" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

export function AuditView() {
  const { session, status } = useSession();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const filters = useMemo(
    () => readFilters(new URLSearchParams(searchParams.toString())),
    [searchParams],
  );

  const [targetEditor, setTargetEditor] = useState({
    source: filters.targetId,
    value: filters.targetId,
    error: null as string | null,
  });
  const [selected, setSelected] = useState<AuditLogEntry | null>(null);

  // The URL is the source of truth for the applied filter. When it changes
  // (submit, reset, or browser navigation), the draft follows it without an
  // effect.
  const targetEditorIsStale = targetEditor.source !== filters.targetId;
  const targetInput = targetEditorIsStale ? filters.targetId : targetEditor.value;
  const targetError = targetEditorIsStale ? null : targetEditor.error;

  const queryParams = useMemo<GetAdminAuditLogsParams>(() => {
    const params: GetAdminAuditLogsParams = { limit: PAGE_SIZE };

    if (filters.actionPrefix) {
      params.actionPrefix = filters.actionPrefix;
    }
    const from = dateFilterToISO(filters.from);
    if (from) {
      params.from = from;
    }
    const to = dateFilterToISO(filters.to, true);
    if (to) {
      params.to = to;
    }
    if (filters.targetId) {
      params.targetId = filters.targetId;
    }

    return params;
  }, [filters]);

  const canView = status === "authenticated" && can(session, ADMIN_PERMISSIONS.auditView);
  const query = useInfiniteQuery({
    queryKey: ["admin-audit-logs", queryParams],
    queryFn: ({ pageParam, signal }) =>
      getAdminAuditLogs(
        { ...queryParams, cursor: pageParam },
        { signal },
      ),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    enabled: canView,
    retry: false,
  });

  function applyFilters(next: Partial<Filters>) {
    const merged = { ...filters, ...next };
    const params = new URLSearchParams();

    for (const [key, value] of Object.entries(merged)) {
      const trimmed = value.trim();
      if (trimmed) {
        params.set(key, trimmed);
      }
    }

    const queryString = params.toString();
    router.replace(queryString ? `${pathname}?${queryString}` : pathname, {
      scroll: false,
    });
  }

  function submitTarget(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = targetInput.trim();

    if (value && !UUID_PATTERN.test(value)) {
      setTargetEditor({
        source: filters.targetId,
        value: targetInput,
        error: "ID target harus berupa UUID yang valid.",
      });
      return;
    }

    setTargetEditor({ source: filters.targetId, value, error: null });
    applyFilters({ targetId: value });
  }

  if (status !== "authenticated") {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-72" />
        <FiltersSkeleton />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!can(session, ADMIN_PERMISSIONS.auditView)) {
    return <NotFound />;
  }

  if (query.error instanceof ApiError && query.error.status === 404) {
    return <NotFound />;
  }

  const items = query.data?.pages.flatMap((page) => page.items) ?? [];
  const hasIP = items.some((item) => item.actorIp != null);
  const hasActiveFilters = Boolean(
    filters.actionPrefix || filters.from || filters.to || filters.targetId,
  );
  const columnCount = hasIP ? 6 : 5;

  return (
    <div>
      <AdminPageHeader
        title="Audit Log"
        description="Telusuri jejak aktivitas administratif. Admin jurusan hanya melihat aktivitas pada jurusannya."
      />

      <div className="mb-4 rounded-xl border p-4">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div className="space-y-1.5">
            <Label htmlFor="audit-action">Aksi</Label>
            <Select
              value={filters.actionPrefix || ALL_ACTIONS}
              onValueChange={(value) =>
                applyFilters({
                  actionPrefix: value === ALL_ACTIONS ? "" : value,
                })
              }
            >
              <SelectTrigger id="audit-action" className="w-full">
                <SelectValue placeholder="Semua aksi" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_ACTIONS}>Semua aksi</SelectItem>
                <SelectGroup>
                  <SelectLabel>Akun &amp; auth</SelectLabel>
                  <SelectItem value="auth.">Akun &amp; auth (auth.)</SelectItem>
                </SelectGroup>
                <SelectGroup>
                  <SelectLabel>Pengguna</SelectLabel>
                  <SelectItem value="user.">Pengguna (user.)</SelectItem>
                </SelectGroup>
                <SelectGroup>
                  <SelectLabel>Role</SelectLabel>
                  <SelectItem value="role.">Role (role.)</SelectItem>
                </SelectGroup>
                <SelectGroup>
                  <SelectLabel>Akademik</SelectLabel>
                  <SelectItem value="academic.">Akademik (academic.)</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="audit-from">Dari tanggal</Label>
            <DatePicker
              id="audit-from"
              value={filters.from}
              onChange={(value) => applyFilters({ from: value })}
              placeholder="Semua tanggal"
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="audit-to">Sampai tanggal</Label>
            <DatePicker
              id="audit-to"
              value={filters.to}
              onChange={(value) => applyFilters({ to: value })}
              placeholder="Semua tanggal"
            />
          </div>

          <form onSubmit={submitTarget} className="space-y-1.5">
            <Label htmlFor="audit-target">ID target</Label>
            <div className="flex gap-2">
              <Input
                id="audit-target"
                placeholder="UUID target (opsional)"
                value={targetInput}
                onChange={(event) =>
                  setTargetEditor({
                    source: filters.targetId,
                    value: event.target.value,
                    error: null,
                  })
                }
              />
              <Button
                type="submit"
                variant="secondary"
                size="icon"
                aria-label="Terapkan ID target"
              >
                <Search aria-hidden="true" />
              </Button>
            </div>
            {targetError ? (
              <p className="text-xs text-destructive">{targetError}</p>
            ) : null}
          </form>
        </div>

        {hasActiveFilters ? (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="mt-3"
            onClick={() => {
              router.replace(pathname, { scroll: false });
            }}
          >
            Reset filter
          </Button>
        ) : null}
      </div>

      {query.isError ? (
        <QueryError
          title="Gagal memuat audit log"
          message={apiErrorMessage(
            query.error,
            "Daftar audit log tidak dapat dimuat.",
          )}
          onRetry={() => {
            void query.refetch();
          }}
          isRetrying={query.isFetching}
        />
      ) : (
        <>
          <div className="rounded-xl border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Waktu</TableHead>
                  <TableHead>Aktor</TableHead>
                  <TableHead>Aksi</TableHead>
                  <TableHead>Target</TableHead>
                  {hasIP ? <TableHead>IP</TableHead> : null}
                  <TableHead className="w-20">
                    <span className="sr-only">Aksi</span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {query.isPending ? <TableSkeleton /> : null}

                {!query.isPending && items.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={columnCount} className="py-12 text-center">
                      <Inbox
                        aria-hidden="true"
                        className="mx-auto mb-2 size-6 text-muted-foreground"
                      />
                      <p className="font-medium">Belum ada aktivitas audit.</p>
                      <p className="mt-1 text-sm text-muted-foreground">
                        {hasActiveFilters
                          ? "Tidak ada entri yang cocok dengan filter ini."
                          : "Aktivitas administratif akan muncul di sini."}
                      </p>
                    </TableCell>
                  </TableRow>
                ) : null}

                {items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="whitespace-nowrap text-muted-foreground">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <time dateTime={item.occurredAt}>
                            {formatDateTime(item.occurredAt)}
                          </time>
                        </TooltipTrigger>
                        <TooltipContent>
                          {formatAbsoluteDateTime(item.occurredAt)}
                        </TooltipContent>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      {item.actorId ? (
                        <div className="space-y-0.5">
                          <p className="font-medium">
                            {item.actorName ?? "Pengguna"}
                          </p>
                          {item.actorUsername ? (
                            <p className="text-xs text-muted-foreground">
                              @{item.actorUsername}
                            </p>
                          ) : null}
                        </div>
                      ) : (
                        <span className="text-muted-foreground">Sistem</span>
                      )}
                    </TableCell>
                    <TableCell>
                      {actionLabel(item.action) ? (
                        <span title={item.action}>
                          {actionLabel(item.action)}
                        </span>
                      ) : (
                        <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
                          {item.action}
                        </code>
                      )}
                    </TableCell>
                    <TableCell>
                      {item.targetId ? (
                        <div className="space-y-0.5">
                          <p>
                            {targetTypeLabel(item.targetType) ?? "Target"}
                          </p>
                          <p
                            className="font-mono text-xs text-muted-foreground"
                            title={item.targetId}
                          >
                            {shortId(item.targetId)}
                          </p>
                        </div>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    {hasIP ? (
                      <TableCell>
                        {item.actorIp ? (
                          <code className="font-mono text-xs">
                            {item.actorIp}
                          </code>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                    ) : null}
                    <TableCell className="text-right">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => setSelected(item)}
                      >
                        Detail
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {query.hasNextPage ? (
            <div className="mt-4 flex justify-center">
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  void query.fetchNextPage();
                }}
                disabled={query.isFetchingNextPage}
              >
                {query.isFetchingNextPage ? "Memuat…" : "Muat lebih banyak"}
              </Button>
            </div>
          ) : null}
        </>
      )}

      <AuditDetailSheet
        entry={selected}
        onClose={() => setSelected(null)}
      />
    </div>
  );
}
