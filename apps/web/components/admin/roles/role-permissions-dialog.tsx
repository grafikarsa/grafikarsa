"use client";

import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { CircleAlert } from "lucide-react";
import { toast } from "sonner";

import { roleMutationErrorMessage } from "@/components/admin/roles/form-error";
import { PermissionChecklist } from "@/components/admin/roles/permission-checklist";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  getGetAdminRolesQueryKey,
  usePutAdminRolePermissions,
} from "@/lib/api/gen/endpoints";
import type { AdminPermission, AdminRole } from "@/lib/api/gen/models";

export function RolePermissionsDialog({
  role,
  permissions,
  onClose,
}: {
  role: AdminRole;
  permissions: AdminPermission[];
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState(role.permissions);
  const [formError, setFormError] = useState<string | null>(null);

  const updateMutation = usePutAdminRolePermissions({
    mutation: {
      onSuccess: () => {
        toast.success("Izin role diperbarui");
        void queryClient.invalidateQueries({
          queryKey: getGetAdminRolesQueryKey(),
        });
        onClose();
      },
      onError: (error) =>
        setFormError(
          roleMutationErrorMessage(error, "Izin role gagal diperbarui."),
        ),
    },
  });

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <DialogContent className="max-h-[90svh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Ubah izin role</DialogTitle>
          <DialogDescription>
            Izin untuk role {role.name}. Perubahan menggantikan seluruh izin
            role ini.
          </DialogDescription>
        </DialogHeader>

        {formError ? (
          <Alert variant="destructive">
            <CircleAlert aria-hidden="true" />
            <AlertTitle>Gagal menyimpan izin</AlertTitle>
            <AlertDescription>{formError}</AlertDescription>
          </Alert>
        ) : null}

        <PermissionChecklist
          permissions={permissions}
          selected={selected}
          onChange={setSelected}
        />

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={updateMutation.isPending}
          >
            Batal
          </Button>
          <Button
            type="button"
            onClick={() => {
              setFormError(null);
              updateMutation.mutate({
                id: role.id,
                data: { permissions: selected },
              });
            }}
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? "Menyimpan…" : "Simpan izin"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
