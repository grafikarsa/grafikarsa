"use client";

import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { CircleAlert } from "lucide-react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { applyRoleFormError } from "@/components/admin/roles/form-error";
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
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  getGetAdminRolesQueryKey,
  usePostAdminRole,
} from "@/lib/api/gen/endpoints";
import type { AdminPermission } from "@/lib/api/gen/models";

const roleSchema = z.object({
  slug: z
    .string()
    .trim()
    .min(2, "Slug minimal 2 karakter.")
    .max(50, "Slug maksimal 50 karakter.")
    .regex(
      /^[a-z0-9_]+$/,
      "Slug hanya boleh berisi huruf kecil, angka, atau garis bawah.",
    ),
  name: z
    .string()
    .trim()
    .min(2, "Nama minimal 2 karakter.")
    .max(60, "Nama maksimal 60 karakter."),
  description: z.string(),
  permissions: z.array(z.string()),
});

type RoleValues = z.infer<typeof roleSchema>;

function FormError({ message }: { message: string | null }) {
  if (!message) {
    return null;
  }

  return (
    <Alert variant="destructive">
      <CircleAlert aria-hidden="true" />
      <AlertTitle>Gagal membuat role</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

export function RoleCreateDialog({
  permissions,
  onClose,
}: {
  permissions: AdminPermission[];
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm<RoleValues>({
    resolver: zodResolver(roleSchema),
    defaultValues: {
      slug: "",
      name: "",
      description: "",
      permissions: [],
    },
  });

  const createMutation = usePostAdminRole({
    mutation: {
      onSuccess: () => {
        toast.success("Role dibuat");
        void queryClient.invalidateQueries({
          queryKey: getGetAdminRolesQueryKey(),
        });
        onClose();
      },
      onError: (error) =>
        setFormError(applyRoleFormError(form, error, "Role gagal dibuat.")),
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
          <DialogTitle>Buat role</DialogTitle>
          <DialogDescription>
            Role kustom berisi izin yang kamu miliki secara global. Izin yang
            tidak kamu miliki tidak dapat dipilih.
          </DialogDescription>
        </DialogHeader>

        <FormError message={formError} />

        <Form {...form}>
          <form
            id="create-role-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);
              createMutation.mutate({
                data: {
                  slug: values.slug,
                  name: values.name,
                  description: values.description.trim() || undefined,
                  permissions: values.permissions,
                },
              });
            })}
          >
            <FormField
              control={form.control}
              name="slug"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Slug</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete="off"
                      placeholder="mis. editor_konten"
                      maxLength={50}
                      className="font-mono"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Huruf kecil, angka, atau garis bawah. Tidak dapat diubah
                    setelah dibuat.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Nama</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete="off"
                      placeholder="mis. Editor Konten"
                      maxLength={60}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Deskripsi (opsional)</FormLabel>
                  <FormControl>
                    <Textarea
                      rows={3}
                      placeholder="Jelaskan tujuan role ini…"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="permissions"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Izin</FormLabel>
                  <PermissionChecklist
                    permissions={permissions}
                    selected={field.value}
                    onChange={field.onChange}
                  />
                  <FormMessage />
                </FormItem>
              )}
            />
          </form>
        </Form>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={createMutation.isPending}
          >
            Batal
          </Button>
          <Button
            type="submit"
            form="create-role-form"
            disabled={createMutation.isPending}
          >
            {createMutation.isPending ? "Menyimpan…" : "Buat role"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
