"use client";

import { useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Inbox, Plus, ShieldOff } from "lucide-react";
import { toast } from "sonner";

import { apiErrorMessage } from "@/components/admin/form-error";
import { QueryError } from "@/components/admin/query-error";
import { AssignRoleDialog } from "@/components/admin/users/assign-role-dialog";
import { formatDateTime } from "@/components/admin/users/labels";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
  getGetAdminRolesQueryKey,
  getGetAdminUserQueryKey,
  getGetAdminUserRolesQueryKey,
  useDeleteAdminUserRole,
  useGetAdminUserRoles,
} from "@/lib/api/gen/endpoints";
import type { AdminRoleAssignment, Major } from "@/lib/api/gen/models";

function TableSkeleton() {
  return (
    <>
      {Array.from({ length: 3 }).map((_, index) => (
        <TableRow key={index}>
          <TableCell>
            <Skeleton className="h-4 w-32" />
            <Skeleton className="mt-2 h-3 w-20" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-20 rounded-4xl" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-32" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-32" />
          </TableCell>
          <TableCell>
            <Skeleton className="ml-auto h-8 w-16" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

export function UserRolesTab({
  userId,
  majors,
  canAssign,
  fallbackAssignments,
}: {
  userId: string;
  majors: Major[];
  canAssign: boolean;
  fallbackAssignments: AdminRoleAssignment[];
}) {
  const queryClient = useQueryClient();
  const [assignOpen, setAssignOpen] = useState(false);
  const [revokeTarget, setRevokeTarget] =
    useState<AdminRoleAssignment | null>(null);

  const majorById = useMemo(
    () => new Map(majors.map((major) => [major.id, major])),
    [majors],
  );

  const rolesQuery = useGetAdminUserRoles(userId, {
    query: { enabled: canAssign },
  });

  const assignments = canAssign
    ? (rolesQuery.data?.items ?? [])
    : fallbackAssignments;

  const invalidate = () => {
    void queryClient.invalidateQueries({
      queryKey: getGetAdminUserRolesQueryKey(userId),
    });
    void queryClient.invalidateQueries({
      queryKey: getGetAdminUserQueryKey(userId),
    });
    void queryClient.invalidateQueries({
      queryKey: getGetAdminRolesQueryKey(),
    });
  };

  const revokeMutation = useDeleteAdminUserRole({
    mutation: {
      onSuccess: () => {
        toast.success("Role dicabut");
        invalidate();
        setRevokeTarget(null);
      },
      onError: (error) => {
        toast.error(apiErrorMessage(error, "Role gagal dicabut."));
      },
    },
  });

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-3">
          <div className="space-y-1">
            <CardTitle>Role</CardTitle>
            <CardDescription>
              Role yang dimiliki pengguna beserta cakupan dan masa berlakunya.
            </CardDescription>
          </div>
          {canAssign ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setAssignOpen(true)}
            >
              <Plus aria-hidden="true" />
              Beri role
            </Button>
          ) : null}
        </CardHeader>
        <CardContent>
          {canAssign && rolesQuery.isError ? (
            <QueryError
              title="Gagal memuat role pengguna"
              message={apiErrorMessage(
                rolesQuery.error,
                "Daftar role pengguna tidak dapat dimuat.",
              )}
              onRetry={() => {
                void rolesQuery.refetch();
              }}
              isRetrying={rolesQuery.isFetching}
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Role</TableHead>
                  <TableHead className="w-40">Cakupan</TableHead>
                  <TableHead className="w-44">Diberikan</TableHead>
                  <TableHead className="w-44">Berlaku sampai</TableHead>
                  {canAssign ? (
                    <TableHead className="w-24 text-right">
                      <span className="sr-only">Aksi</span>
                    </TableHead>
                  ) : null}
                </TableRow>
              </TableHeader>
              <TableBody>
                {canAssign && rolesQuery.isPending ? <TableSkeleton /> : null}

                {!(canAssign && rolesQuery.isPending) &&
                assignments.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={canAssign ? 5 : 4}
                      className="py-10 text-center"
                    >
                      <Inbox
                        aria-hidden="true"
                        className="mx-auto mb-2 size-6 text-muted-foreground"
                      />
                      <p className="text-sm text-muted-foreground">
                        Belum ada role aktif.
                      </p>
                    </TableCell>
                  </TableRow>
                ) : null}

                {assignments.map((assignment) => (
                  <TableRow key={assignment.id}>
                    <TableCell>
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium">
                          {assignment.roleName}
                        </span>
                        <span className="font-mono text-xs text-muted-foreground">
                          {assignment.roleSlug}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      {assignment.majorId ? (
                        <Badge variant="outline" className="font-mono">
                          {majorById.get(assignment.majorId)?.code ??
                            assignment.majorId}
                        </Badge>
                      ) : (
                        "Global"
                      )}
                    </TableCell>
                    <TableCell>{formatDateTime(assignment.assignedAt)}</TableCell>
                    <TableCell>
                      {assignment.expiresAt
                        ? formatDateTime(assignment.expiresAt)
                        : "Tanpa batas"}
                    </TableCell>
                    {canAssign ? (
                      <TableCell className="text-right">
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => setRevokeTarget(assignment)}
                        >
                          <ShieldOff aria-hidden="true" />
                          Cabut
                        </Button>
                      </TableCell>
                    ) : null}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {assignOpen ? (
        <AssignRoleDialog
          userId={userId}
          majors={majors}
          onClose={() => setAssignOpen(false)}
        />
      ) : null}

      <AlertDialog
        open={revokeTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setRevokeTarget(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cabut role?</AlertDialogTitle>
            <AlertDialogDescription>
              Role {revokeTarget?.roleName} akan dicabut dari pengguna ini
              {revokeTarget?.majorId
                ? ` pada cakupan ${
                    majorById.get(revokeTarget.majorId)?.code ??
                    revokeTarget.majorId
                  }`
                : " secara global"}
              . Akses yang berasal dari role ini langsung berhenti.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={revokeMutation.isPending}>
              Batal
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={revokeMutation.isPending}
              onClick={(event) => {
                event.preventDefault();

                if (revokeTarget) {
                  revokeMutation.mutate({ assignmentId: revokeTarget.id });
                }
              }}
            >
              {revokeMutation.isPending ? "Memproses…" : "Cabut"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
