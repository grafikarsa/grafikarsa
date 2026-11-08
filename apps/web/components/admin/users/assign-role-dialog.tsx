"use client";

import { useMemo, useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { CircleAlert } from "lucide-react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { parseDateOnly, toDateOnly } from "@/components/admin/date";
import { DatePicker } from "@/components/admin/date-picker";
import { apiErrorMessage } from "@/components/admin/form-error";
import { permissionScope } from "@/components/admin/users/permissions";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  getGetAdminRolesQueryKey,
  getGetAdminUserQueryKey,
  getGetAdminUserRolesQueryKey,
  useGetAdminRoles,
  usePostAdminUserRole,
} from "@/lib/api/gen/endpoints";
import type { AdminRole, Major, Problem } from "@/lib/api/gen/models";
import { ApiError } from "@/lib/api/fetcher";
import { canGlobally } from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

const assignSchema = z
  .object({
    roleSlug: z.string().min(1, "Role wajib dipilih."),
    scope: z.string().min(1, "Cakupan wajib dipilih."),
    expiresOn: z.string(),
  })
  .superRefine((values, ctx) => {
    if (values.expiresOn && values.expiresOn <= toDateOnly(new Date())) {
      ctx.addIssue({
        code: "custom",
        path: ["expiresOn"],
        message: "Tanggal kedaluwarsa harus di masa depan.",
      });
    }
  });

type AssignValues = z.infer<typeof assignSchema>;

function toExpiryTimestamp(value: string): string | null {
  const date = parseDateOnly(value);

  if (!date) {
    return null;
  }

  date.setHours(23, 59, 59, 0);
  return date.toISOString();
}

function applyAssignError(
  form: ReturnType<typeof useForm<AssignValues>>,
  error: unknown,
): string | null {
  if (!(error instanceof ApiError)) {
    return apiErrorMessage(error, "Role gagal diberikan.");
  }

  if (error.status === 404) {
    return "Kamu tidak berwenang memberi role ini pada scope tersebut.";
  }

  const problem = error.problem;

  if (problem?.code === "VALIDATION_FAILED" && problem.errors?.length) {
    let applied = false;

    for (const item of problem.errors) {
      if (item.field === "role") {
        form.setError("roleSlug", { type: "server", message: item.message });
        applied = true;
      } else if (item.field === "majorId") {
        form.setError("scope", { type: "server", message: item.message });
        applied = true;
      } else if (item.field === "expiresAt") {
        form.setError("expiresOn", { type: "server", message: item.message });
        applied = true;
      }
    }

    if (applied) {
      return null;
    }

    return problem.errors[0]?.message ?? null;
  }

  if (error.status === 409) {
    return problem?.detail ?? "Role ini sudah dimiliki pengguna.";
  }

  return apiErrorMessage(error, "Role gagal diberikan.");
}

function FormError({ message }: { message: string | null }) {
  if (!message) {
    return null;
  }

  return (
    <Alert variant="destructive">
      <CircleAlert aria-hidden="true" />
      <AlertTitle>Gagal memberi role</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

function rolesQueryError(problem: Problem | null): string {
  return problem?.detail ?? problem?.title ?? "Daftar role tidak dapat dimuat.";
}

export function AssignRoleDialog({
  userId,
  majors,
  onClose,
}: {
  userId: string;
  majors: Major[];
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const { session } = useSession();
  const [formError, setFormError] = useState<string | null>(null);

  const assignScope = useMemo(
    () => permissionScope(session, "role.assign"),
    [session],
  );
  const assignableMajors = useMemo(
    () =>
      assignScope.global
        ? majors
        : majors.filter((major) => assignScope.majorIds.includes(major.id)),
    [assignScope.global, assignScope.majorIds, majors],
  );

  const rolesQuery = useGetAdminRoles();
  const assignableRoles = useMemo(
    () =>
      (rolesQuery.data?.items ?? []).filter(
        (role) => role.slug !== "member" && role.slug !== "teacher",
      ),
    [rolesQuery.data],
  );

  const form = useForm<AssignValues>({
    resolver: zodResolver(assignSchema),
    defaultValues: {
      roleSlug: "",
      scope: assignScope.global ? "global" : (assignableMajors[0]?.id ?? ""),
      expiresOn: "",
    },
  });

  const assignMutation = usePostAdminUserRole({
    mutation: {
      onSuccess: () => {
        toast.success("Role diberikan");
        void queryClient.invalidateQueries({
          queryKey: getGetAdminUserRolesQueryKey(userId),
        });
        void queryClient.invalidateQueries({
          queryKey: getGetAdminUserQueryKey(userId),
        });
        void queryClient.invalidateQueries({
          queryKey: getGetAdminRolesQueryKey(),
        });
        onClose();
      },
      onError: (error) => setFormError(applyAssignError(form, error)),
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
      <DialogContent className="max-h-[90svh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Beri role</DialogTitle>
          <DialogDescription>
            Pilih role dan cakupannya. Kamu hanya dapat memberi role yang
            seluruh izinnya kamu miliki pada cakupan tersebut.
          </DialogDescription>
        </DialogHeader>

        <FormError message={formError} />

        {rolesQuery.isError ? (
          <Alert variant="destructive">
            <CircleAlert aria-hidden="true" />
            <AlertTitle>Gagal memuat role</AlertTitle>
            <AlertDescription>
              {rolesQueryError(rolesQuery.error ?? null)}
            </AlertDescription>
          </Alert>
        ) : null}

        <Form {...form}>
          <form
            id="assign-role-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);
              assignMutation.mutate({
                id: userId,
                data: {
                  roleSlug: values.roleSlug,
                  majorId: values.scope === "global" ? null : values.scope,
                  expiresAt: values.expiresOn
                    ? toExpiryTimestamp(values.expiresOn)
                    : null,
                },
              });
            })}
          >
            <FormField
              control={form.control}
              name="roleSlug"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Role</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger
                        aria-label="Role"
                        disabled={
                          rolesQuery.isPending || assignableRoles.length === 0
                        }
                      >
                        <SelectValue
                          placeholder={
                            rolesQuery.isPending
                              ? "Memuat role…"
                              : "Pilih role"
                          }
                        />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {assignableRoles.map((role: AdminRole) => (
                        <SelectItem key={role.slug} value={role.slug}>
                          {role.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="scope"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Cakupan</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
                    disabled={assignableMajors.length === 0 && !assignScope.global}
                  >
                    <FormControl>
                      <SelectTrigger aria-label="Cakupan">
                        <SelectValue placeholder="Pilih cakupan" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {canGlobally(session, "role.assign") ? (
                        <SelectItem value="global">Global</SelectItem>
                      ) : null}
                      {assignableMajors.map((major) => (
                        <SelectItem key={major.id} value={major.id}>
                          {major.code} · {major.name}
                          {major.isActive ? "" : " (nonaktif)"}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    Global berlaku untuk seluruh sekolah.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="expiresOn"
              render={({ field }) => (
                <FormItem className="flex flex-col">
                  <FormLabel>Berlaku sampai (opsional)</FormLabel>
                  <FormControl>
                    <div className="flex gap-2">
                      <DatePicker
                        value={field.value}
                        onChange={field.onChange}
                        placeholder="Tanpa batas"
                        className="flex-1"
                      />
                      {field.value ? (
                        <Button
                          type="button"
                          variant="ghost"
                          onClick={() => field.onChange("")}
                        >
                          Hapus
                        </Button>
                      ) : null}
                    </div>
                  </FormControl>
                  <FormDescription>
                    Kosongkan jika role berlaku tanpa batas.
                  </FormDescription>
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
            disabled={assignMutation.isPending}
          >
            Batal
          </Button>
          <Button
            type="submit"
            form="assign-role-form"
            disabled={assignMutation.isPending || assignableRoles.length === 0}
          >
            {assignMutation.isPending ? "Memproses…" : "Beri role"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
