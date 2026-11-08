"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Inbox, Plus, Search } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

import NotFound from "@/app/not-found";
import { AdminPageHeader } from "@/components/admin/admin-page-header";
import { apiErrorMessage } from "@/components/admin/form-error";
import { QueryError } from "@/components/admin/query-error";
import {
  ACCOUNT_STATUS_LABELS,
  ACCOUNT_STATUS_VARIANTS,
  ACCOUNT_TYPE_LABELS,
  formatDateTime,
  formatRelativeTime,
} from "@/components/admin/users/labels";
import { permissionScope } from "@/components/admin/users/permissions";
import { CreateUserDialog } from "@/components/admin/users/create-user-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
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
  getAdminUsers,
  getGetAdminUsersQueryKey,
  useGetAdminAcademicYears,
  useGetAdminClasses,
  useGetAdminMajors,
} from "@/lib/api/gen/endpoints";
import type {
  AccountStatus,
  AccountType,
  GetAdminUsersParams,
} from "@/lib/api/gen/models";
import { ApiError } from "@/lib/api/fetcher";
import { ADMIN_PERMISSIONS, can } from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

const ACCOUNT_TYPES: AccountType[] = ["student", "teacher", "staff"];
const ACCOUNT_STATUSES: AccountStatus[] = [
  "active",
  "alumni",
  "transferred",
  "inactive",
];

function isAccountType(value: string | null): value is AccountType {
  return ACCOUNT_TYPES.includes(value as AccountType);
}

function isAccountStatus(value: string | null): value is AccountStatus {
  return ACCOUNT_STATUSES.includes(value as AccountStatus);
}

function TableSkeleton() {
  return (
    <>
      {Array.from({ length: 6 }).map((_, index) => (
        <TableRow key={index}>
          <TableCell>
            <Skeleton className="h-4 w-40" />
            <Skeleton className="mt-2 h-3 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-14" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-16 rounded-4xl" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-24" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

export function UsersView() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { session, status } = useSession();
  const [createOpen, setCreateOpen] = useState(false);

  const qParam = searchParams.get("q") ?? "";
  const accountTypeParam = searchParams.get("tipe");
  const statusParam = searchParams.get("status");
  const majorParam = searchParams.get("jurusan");
  const classParam = searchParams.get("kelas");

  const accountType = isAccountType(accountTypeParam)
    ? accountTypeParam
    : "all";
  const accountStatus = isAccountStatus(statusParam) ? statusParam : "all";
  const majorFilter = majorParam ?? "all";
  const classFilter = classParam ?? "all";

  const [searchInput, setSearchInput] = useState(qParam);
  const searchTimer = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (searchTimer.current) {
        window.clearTimeout(searchTimer.current);
      }
    };
  }, []);

  const updateFilters = useCallback(
    (updates: Record<string, string | null>) => {
      const next = new URLSearchParams(window.location.search);

      for (const [key, value] of Object.entries(updates)) {
        if (value === null || value === "") {
          next.delete(key);
        } else {
          next.set(key, value);
        }
      }

      const query = next.toString();
      router.replace(query ? `${pathname}?${query}` : pathname, {
        scroll: false,
      });
    },
    [pathname, router],
  );

  const handleSearchChange = (value: string) => {
    setSearchInput(value);

    if (searchTimer.current) {
      window.clearTimeout(searchTimer.current);
    }

    searchTimer.current = window.setTimeout(() => {
      updateFilters({ q: value.trim() || null });
    }, 300);
  };

  const authenticated = status === "authenticated";
  const canView = authenticated && can(session, ADMIN_PERMISSIONS.userView);
  const canCreate = authenticated && can(session, "user.create");

  const majorsQuery = useGetAdminMajors({
    query: { enabled: canView },
  });
  const yearsQuery = useGetAdminAcademicYears({
    query: { enabled: canView },
  });

  const viewScope = useMemo(
    () => permissionScope(session, ADMIN_PERMISSIONS.userView),
    [session],
  );
  const allMajors = useMemo(
    () => majorsQuery.data?.items ?? [],
    [majorsQuery.data],
  );
  const majors = useMemo(
    () =>
      viewScope.global
        ? allMajors
        : allMajors.filter((major) => viewScope.majorIds.includes(major.id)),
    [allMajors, viewScope.global, viewScope.majorIds],
  );

  const years = useMemo(() => yearsQuery.data?.items ?? [], [yearsQuery.data]);
  const currentYear = useMemo(
    () => years.find((year) => year.isCurrent) ?? years[0],
    [years],
  );

  const classesQuery = useGetAdminClasses(
    {
      academicYearId: currentYear?.id,
      majorId: majorFilter === "all" ? undefined : majorFilter,
    },
    {
      query: {
        enabled: canView && Boolean(currentYear?.id),
      },
    },
  );

  const classes = useMemo(() => {
    const items = classesQuery.data?.items ?? [];

    return items.filter(
      (schoolClass) =>
        viewScope.global || viewScope.majorIds.includes(schoolClass.majorId),
    );
  }, [classesQuery.data, viewScope.global, viewScope.majorIds]);

  useEffect(() => {
    if (classFilter === "all" || classesQuery.isPending) {
      return;
    }

    if (!classes.some((schoolClass) => schoolClass.id === classFilter)) {
      updateFilters({ kelas: null });
    }
  }, [classFilter, classes, classesQuery.isPending, updateFilters]);

  const listParams = useMemo<GetAdminUsersParams>(
    () => ({
      q: qParam || undefined,
      accountType: accountType === "all" ? undefined : accountType,
      status: accountStatus === "all" ? undefined : accountStatus,
      majorId: majorFilter === "all" ? undefined : majorFilter,
      classId: classFilter === "all" ? undefined : classFilter,
      limit: 25,
    }),
    [accountStatus, accountType, classFilter, majorFilter, qParam],
  );

  const usersQuery = useInfiniteQuery({
    queryKey: getGetAdminUsersQueryKey(listParams),
    queryFn: ({ pageParam, signal }) =>
      getAdminUsers(
        { ...listParams, cursor: pageParam },
        { signal },
      ),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    enabled: canView,
  });

  const users = useMemo(
    () => usersQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [usersQuery.data],
  );

  if (status !== "authenticated") {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-4 w-72" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!canView) {
    return <NotFound />;
  }

  if (
    usersQuery.error instanceof ApiError &&
    usersQuery.error.status === 404
  ) {
    return <NotFound />;
  }

  const activeFilterCount = [
    qParam,
    accountType !== "all" ? accountType : "",
    accountStatus !== "all" ? accountStatus : "",
    majorFilter !== "all" ? majorFilter : "",
    classFilter !== "all" ? classFilter : "",
  ].filter(Boolean).length;

  return (
    <div>
      <AdminPageHeader
        title="Pengguna"
        description="Cari dan kelola akun siswa, guru, dan tendik."
        action={
          canCreate ? (
            <Button onClick={() => setCreateOpen(true)}>
              <Plus aria-hidden="true" />
              Tambah pengguna
            </Button>
          ) : null
        }
      />

      <div className="grid gap-3 pb-4 sm:grid-cols-2 lg:flex lg:flex-wrap lg:items-end">
        <div className="grid gap-1.5 sm:col-span-2 lg:w-72">
          <span className="text-sm font-medium">Cari</span>
          <div className="relative">
            <Search
              aria-hidden="true"
              className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              value={searchInput}
              onChange={(event) => handleSearchChange(event.target.value)}
              placeholder="Nama, username, NISN, atau NIP"
              className="pl-8"
              maxLength={100}
              aria-label="Cari pengguna"
            />
          </div>
        </div>

        <div className="grid gap-1.5">
          <span className="text-sm font-medium">Tipe akun</span>
          <Select
            value={accountType}
            onValueChange={(value) =>
              updateFilters({ tipe: value === "all" ? null : value })
            }
          >
            <SelectTrigger className="w-full lg:w-40" aria-label="Tipe akun">
              <SelectValue placeholder="Semua tipe" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua tipe</SelectItem>
              {ACCOUNT_TYPES.map((type) => (
                <SelectItem key={type} value={type}>
                  {ACCOUNT_TYPE_LABELS[type]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="grid gap-1.5">
          <span className="text-sm font-medium">Status</span>
          <Select
            value={accountStatus}
            onValueChange={(value) =>
              updateFilters({ status: value === "all" ? null : value })
            }
          >
            <SelectTrigger className="w-full lg:w-40" aria-label="Status">
              <SelectValue placeholder="Semua status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua status</SelectItem>
              {ACCOUNT_STATUSES.map((value) => (
                <SelectItem key={value} value={value}>
                  {ACCOUNT_STATUS_LABELS[value]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="grid gap-1.5">
          <span className="text-sm font-medium">Jurusan</span>
          <Select
            value={majorFilter}
            onValueChange={(value) =>
              updateFilters({
                jurusan: value === "all" ? null : value,
                kelas: null,
              })
            }
            disabled={majorsQuery.isPending || majors.length === 0}
          >
            <SelectTrigger className="w-full lg:w-52" aria-label="Jurusan">
              <SelectValue placeholder="Semua jurusan" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua jurusan</SelectItem>
              {majors.map((major) => (
                <SelectItem key={major.id} value={major.id}>
                  {major.name}
                  {major.isActive ? "" : " (nonaktif)"}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="grid gap-1.5">
          <span className="text-sm font-medium">Kelas</span>
          <Select
            value={classFilter}
            onValueChange={(value) =>
              updateFilters({ kelas: value === "all" ? null : value })
            }
            disabled={!currentYear || classesQuery.isPending}
          >
            <SelectTrigger className="w-full lg:w-44" aria-label="Kelas">
              <SelectValue placeholder="Semua kelas" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua kelas</SelectItem>
              {classes.map((schoolClass) => (
                <SelectItem key={schoolClass.id} value={schoolClass.id}>
                  {schoolClass.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {usersQuery.isError ? (
        <QueryError
          title="Gagal memuat pengguna"
          message={apiErrorMessage(
            usersQuery.error,
            "Daftar pengguna tidak dapat dimuat.",
          )}
          onRetry={() => {
            void usersQuery.refetch();
          }}
          isRetrying={usersQuery.isFetching}
        />
      ) : (
        <>
          <div className="rounded-xl border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nama</TableHead>
                  <TableHead className="w-20">Tipe</TableHead>
                  <TableHead className="w-36">NISN/NIP</TableHead>
                  <TableHead className="w-36">Kelas</TableHead>
                  <TableHead className="w-28">Status</TableHead>
                  <TableHead className="w-36">Terakhir masuk</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usersQuery.isPending ? <TableSkeleton /> : null}

                {!usersQuery.isPending && users.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="py-12 text-center">
                      <Inbox
                        aria-hidden="true"
                        className="mx-auto mb-2 size-6 text-muted-foreground"
                      />
                      <p className="font-medium">
                        Belum ada pengguna yang cocok.
                      </p>
                      <p className="mt-1 text-sm text-muted-foreground">
                        {activeFilterCount > 0
                          ? "Coba ubah atau hapus filter."
                          : "Tambahkan pengguna pertama untuk memulai."}
                      </p>
                    </TableCell>
                  </TableRow>
                ) : null}

                {users.map((user) => (
                  <TableRow
                    key={user.id}
                    className="cursor-pointer"
                    tabIndex={0}
                    onClick={() => router.push(`/admin/pengguna/${user.id}`)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        router.push(`/admin/pengguna/${user.id}`);
                      }
                    }}
                  >
                    <TableCell>
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium">{user.displayName}</span>
                        {user.username ? (
                          <span className="text-xs text-muted-foreground">
                            @{user.username}
                          </span>
                        ) : (
                          <Badge
                            variant="outline"
                            className="mt-0.5 w-fit text-[0.65rem]"
                          >
                            Belum aktivasi
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      {ACCOUNT_TYPE_LABELS[user.accountType]}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {user.accountType === "student"
                        ? (user.nisn ?? "—")
                        : (user.staffNumber ?? "—")}
                    </TableCell>
                    <TableCell>{user.currentClass?.name ?? "—"}</TableCell>
                    <TableCell>
                      <Badge variant={ACCOUNT_STATUS_VARIANTS[user.status]}>
                        {ACCOUNT_STATUS_LABELS[user.status]}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <span
                        title={formatDateTime(user.lastLoginAt)}
                        suppressHydrationWarning
                      >
                        {formatRelativeTime(user.lastLoginAt)}
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {usersQuery.hasNextPage ? (
            <div className="flex justify-center pt-4">
              <Button
                variant="outline"
                onClick={() => {
                  void usersQuery.fetchNextPage();
                }}
                disabled={usersQuery.isFetchingNextPage}
              >
                {usersQuery.isFetchingNextPage
                  ? "Memuat…"
                  : "Muat lebih banyak"}
              </Button>
            </div>
          ) : null}
        </>
      )}

      {createOpen ? (
        <CreateUserDialog onClose={() => setCreateOpen(false)} />
      ) : null}
    </div>
  );
}
