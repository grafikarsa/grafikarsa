"use client";

import { useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Pencil, SlidersHorizontal, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { roleMutationErrorMessage } from "@/components/admin/roles/form-error";
import { groupPermissionKeys } from "@/components/admin/roles/permission-groups";
import { RoleEditDialog } from "@/components/admin/roles/role-edit-dialog";
import { RolePermissionsDialog } from "@/components/admin/roles/role-permissions-dialog";
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
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  getGetAdminRolesQueryKey,
  useDeleteAdminRole,
} from "@/lib/api/gen/endpoints";
import type { AdminPermission, AdminRole } from "@/lib/api/gen/models";

function Field({
  label,
  value,
  mono = false,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div className="space-y-1">
      <dt className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </dt>
      <dd className={mono ? "font-mono text-sm" : "text-sm"}>{value}</dd>
    </div>
  );
}

function DeleteRoleButton({
  disabled,
  onClick,
}: {
  disabled: boolean;
  onClick: () => void;
}) {
  const button = (
    <Button
      type="button"
      variant="destructive"
      size="sm"
      disabled={disabled}
      onClick={onClick}
      className={disabled ? "pointer-events-none" : undefined}
    >
      <Trash2 aria-hidden="true" />
      Hapus
    </Button>
  );

  if (!disabled) {
    return button;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0} className="inline-flex">
          {button}
        </span>
      </TooltipTrigger>
      <TooltipContent>
        Role masih dimiliki pengguna sehingga belum bisa dihapus.
      </TooltipContent>
    </Tooltip>
  );
}

export function RoleSheet({
  role,
  permissions,
  canManage,
  onClose,
}: {
  role: AdminRole;
  permissions: AdminPermission[];
  canManage: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [permissionsOpen, setPermissionsOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const groups = useMemo(
    () => groupPermissionKeys(role.permissions, permissions),
    [permissions, role.permissions],
  );

  const deleteMutation = useDeleteAdminRole({
    mutation: {
      onSuccess: () => {
        toast.success("Role dihapus");
        void queryClient.invalidateQueries({
          queryKey: getGetAdminRolesQueryKey(),
        });
        setDeleteOpen(false);
        onClose();
      },
      onError: (error) => {
        toast.error(roleMutationErrorMessage(error, "Role gagal dihapus."));
      },
    },
  });

  return (
    <>
      <Sheet
        open
        onOpenChange={(open) => {
          if (!open) {
            onClose();
          }
        }}
      >
        <SheetContent className="w-full overflow-y-auto sm:max-w-lg">
          <SheetHeader>
            <SheetTitle className="flex flex-wrap items-center gap-2">
              {role.name}
              <Badge variant={role.isSystem ? "secondary" : "outline"}>
                {role.isSystem ? "Sistem" : "Kustom"}
              </Badge>
            </SheetTitle>
            <SheetDescription>
              Detail role, izin yang dimiliki, dan jumlah pemegangnya.
            </SheetDescription>
          </SheetHeader>

          <div className="space-y-6 px-4 pb-6">
            <dl className="grid grid-cols-2 gap-4">
              <Field label="Slug" value={role.slug} mono />
              <Field
                label="Jenis"
                value={role.isSystem ? "Sistem" : "Kustom"}
              />
              <Field label="Jumlah izin" value={role.permissions.length} />
              <Field label="Pemegang" value={role.holderCount} />
            </dl>

            <div className="space-y-1">
              <h3 className="text-sm font-medium">Deskripsi</h3>
              <p className="text-sm text-muted-foreground">
                {role.description ?? "Tanpa deskripsi."}
              </p>
            </div>

            {canManage && !role.isSystem ? (
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setEditOpen(true)}
                >
                  <Pencil aria-hidden="true" />
                  Ubah
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setPermissionsOpen(true)}
                >
                  <SlidersHorizontal aria-hidden="true" />
                  Ubah izin
                </Button>
                <DeleteRoleButton
                  disabled={role.holderCount > 0}
                  onClick={() => setDeleteOpen(true)}
                />
              </div>
            ) : null}

            {role.isSystem ? (
              <p className="text-sm text-muted-foreground">
                Role sistem: izin tidak bisa diubah.
              </p>
            ) : null}

            <div className="space-y-3">
              <h3 className="text-sm font-medium">Izin</h3>

              {groups.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  Role ini belum memiliki izin.
                </p>
              ) : (
                groups.map((group) => (
                  <div key={group.prefix} className="space-y-1.5">
                    <h4 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                      {group.title}
                    </h4>
                    <ul className="space-y-1.5">
                      {group.items.map((permission) => (
                        <li
                          key={permission.key}
                          className="rounded-md border px-3 py-2"
                        >
                          <p className="font-mono text-xs">
                            {permission.key}
                          </p>
                          {permission.description ? (
                            <p className="mt-0.5 text-xs text-muted-foreground">
                              {permission.description}
                            </p>
                          ) : null}
                        </li>
                      ))}
                    </ul>
                  </div>
                ))
              )}
            </div>
          </div>
        </SheetContent>
      </Sheet>

      {editOpen ? (
        <RoleEditDialog role={role} onClose={() => setEditOpen(false)} />
      ) : null}

      {permissionsOpen ? (
        <RolePermissionsDialog
          role={role}
          permissions={permissions}
          onClose={() => setPermissionsOpen(false)}
        />
      ) : null}

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hapus role?</AlertDialogTitle>
            <AlertDialogDescription>
              Role {role.name} akan dihapus permanen. Riwayat penugasan lama
              tetap tercatat di audit log.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>
              Batal
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={(event) => {
                event.preventDefault();
                deleteMutation.mutate({ id: role.id });
              }}
            >
              {deleteMutation.isPending ? "Memproses…" : "Hapus"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
