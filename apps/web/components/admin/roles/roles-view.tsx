"use client";

import { useMemo, useState } from "react";
import { Inbox, Plus } from "lucide-react";

import NotFound from "@/app/not-found";
import { AdminPageHeader } from "@/components/admin/admin-page-header";
import { apiErrorMessage } from "@/components/admin/form-error";
import { QueryError } from "@/components/admin/query-error";
import { RoleCreateDialog } from "@/components/admin/roles/role-create-dialog";
import { RoleSheet } from "@/components/admin/roles/role-sheet";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
  useGetAdminPermissions,
  useGetAdminRoles,
} from "@/lib/api/gen/endpoints";
import { ApiError } from "@/lib/api/fetcher";
import {
  ADMIN_PERMISSIONS,
  can,
  canGlobally,
} from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

function TableSkeleton() {
  return (
    <>
      {Array.from({ length: 5 }).map((_, index) => (
        <TableRow key={index}>
          <TableCell>
            <Skeleton className="h-4 w-32" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-16 rounded-4xl" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-10" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-10" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

export function RolesView() {
  const { session, status } = useSession();
  const [createOpen, setCreateOpen] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const authenticated = status === "authenticated";
  const canView =
    authenticated &&
    (can(session, ADMIN_PERMISSIONS.roleManage) ||
      can(session, ADMIN_PERMISSIONS.roleAssign));
  const canManage =
    authenticated && canGlobally(session, ADMIN_PERMISSIONS.roleManage);

  const rolesQuery = useGetAdminRoles({
    query: { enabled: canView },
  });
  const permissionsQuery = useGetAdminPermissions({
    query: { enabled: canView },
  });

  const roles = useMemo(() => rolesQuery.data?.items ?? [], [rolesQuery.data]);
  const permissions = useMemo(
    () => permissionsQuery.data?.items ?? [],
    [permissionsQuery.data],
  );
  const selectedRole = useMemo(
    () => roles.find((role) => role.id === selectedId) ?? null,
    [roles, selectedId],
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

  const notFound =
    (rolesQuery.error instanceof ApiError &&
      rolesQuery.error.status === 404) ||
    (permissionsQuery.error instanceof ApiError &&
      permissionsQuery.error.status === 404);

  if (notFound) {
    return <NotFound />;
  }

  const isError = rolesQuery.isError || permissionsQuery.isError;
  const isPending = rolesQuery.isPending || permissionsQuery.isPending;

  return (
    <div>
      <AdminPageHeader
        title="Role"
        description="Kelola role dan izin yang melekat padanya."
        action={
          canManage ? (
            <Button onClick={() => setCreateOpen(true)}>
              <Plus aria-hidden="true" />
              Buat role
            </Button>
          ) : null
        }
      />

      {isError ? (
        <QueryError
          title="Gagal memuat role"
          message={apiErrorMessage(
            rolesQuery.error ?? permissionsQuery.error,
            "Daftar role tidak dapat dimuat.",
          )}
          onRetry={() => {
            void rolesQuery.refetch();
            void permissionsQuery.refetch();
          }}
          isRetrying={rolesQuery.isFetching || permissionsQuery.isFetching}
        />
      ) : (
        <div className="rounded-xl border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama</TableHead>
                <TableHead className="w-48">Slug</TableHead>
                <TableHead className="w-28">Jenis</TableHead>
                <TableHead className="w-32">Jumlah izin</TableHead>
                <TableHead className="w-28">Pemegang</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isPending ? <TableSkeleton /> : null}

              {!isPending && roles.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="py-12 text-center">
                    <Inbox
                      aria-hidden="true"
                      className="mx-auto mb-2 size-6 text-muted-foreground"
                    />
                    <p className="font-medium">Belum ada role.</p>
                    <p className="mt-1 text-sm text-muted-foreground">
                      Buat role kustom pertama untuk memulai.
                    </p>
                  </TableCell>
                </TableRow>
              ) : null}

              {roles.map((role) => (
                <TableRow
                  key={role.id}
                  className="cursor-pointer"
                  tabIndex={0}
                  onClick={() => setSelectedId(role.id)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      setSelectedId(role.id);
                    }
                  }}
                >
                  <TableCell className="font-medium">{role.name}</TableCell>
                  <TableCell className="font-mono text-xs">
                    {role.slug}
                  </TableCell>
                  <TableCell>
                    <Badge variant={role.isSystem ? "secondary" : "outline"}>
                      {role.isSystem ? "Sistem" : "Kustom"}
                    </Badge>
                  </TableCell>
                  <TableCell>{role.permissions.length}</TableCell>
                  <TableCell>{role.holderCount}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {selectedRole ? (
        <RoleSheet
          role={selectedRole}
          permissions={permissions}
          canManage={canManage}
          onClose={() => setSelectedId(null)}
        />
      ) : null}

      {createOpen ? (
        <RoleCreateDialog
          permissions={permissions}
          onClose={() => setCreateOpen(false)}
        />
      ) : null}
    </div>
  );
}
