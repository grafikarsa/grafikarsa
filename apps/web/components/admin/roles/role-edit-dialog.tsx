"use client";

import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { CircleAlert } from "lucide-react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { applyRoleFormError } from "@/components/admin/roles/form-error";
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
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  getGetAdminRolesQueryKey,
  usePatchAdminRole,
} from "@/lib/api/gen/endpoints";
import type { AdminRole } from "@/lib/api/gen/models";

const roleSchema = z.object({
  name: z
    .string()
    .trim()
    .min(2, "Nama minimal 2 karakter.")
    .max(60, "Nama maksimal 60 karakter."),
  description: z.string(),
});

type RoleValues = z.infer<typeof roleSchema>;

function FormError({ message }: { message: string | null }) {
  if (!message) {
    return null;
  }

  return (
    <Alert variant="destructive">
      <CircleAlert aria-hidden="true" />
      <AlertTitle>Gagal menyimpan perubahan</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

export function RoleEditDialog({
  role,
  onClose,
}: {
  role: AdminRole;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm<RoleValues>({
    resolver: zodResolver(roleSchema),
    defaultValues: {
      name: role.name,
      description: role.description ?? "",
    },
  });

  const updateMutation = usePatchAdminRole({
    mutation: {
      onSuccess: () => {
        toast.success("Role diperbarui");
        void queryClient.invalidateQueries({
          queryKey: getGetAdminRolesQueryKey(),
        });
        onClose();
      },
      onError: (error) =>
        setFormError(
          applyRoleFormError(form, error, "Role gagal diperbarui."),
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
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Ubah role</DialogTitle>
          <DialogDescription>
            Slug <span className="font-mono">{role.slug}</span> tidak dapat
            diubah.
          </DialogDescription>
        </DialogHeader>

        <FormError message={formError} />

        <Form {...form}>
          <form
            id="edit-role-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);
              updateMutation.mutate({
                id: role.id,
                data: {
                  name: values.name,
                  description: values.description.trim(),
                },
              });
            })}
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Nama</FormLabel>
                  <FormControl>
                    <Input autoComplete="off" maxLength={60} {...field} />
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
                  <FormLabel>Deskripsi</FormLabel>
                  <FormControl>
                    <Textarea
                      rows={3}
                      placeholder="Kosongkan untuk menghapus deskripsi…"
                      {...field}
                    />
                  </FormControl>
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
            disabled={updateMutation.isPending}
          >
            Batal
          </Button>
          <Button
            type="submit"
            form="edit-role-form"
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? "Menyimpan…" : "Simpan"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
