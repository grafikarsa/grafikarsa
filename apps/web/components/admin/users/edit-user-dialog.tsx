"use client";

import { useMemo, useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { CircleAlert } from "lucide-react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import {
  applyApiError,
} from "@/components/admin/form-error";
import {
  ACCOUNT_TYPE_LABELS,
} from "@/components/admin/users/labels";
import { MajorCombobox } from "@/components/admin/users/major-combobox";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  getGetAdminUserQueryKey,
  getGetAdminUsersQueryKey,
  useGetAdminMajors,
  usePatchAdminUser,
} from "@/lib/api/gen/endpoints";
import type {
  AccountType,
  AdminUserDetail,
  AdminUserUpdate,
} from "@/lib/api/gen/models";

const editUserSchema = z
  .object({
    accountType: z.enum(["student", "teacher", "staff"]),
    displayName: z
      .string()
      .trim()
      .min(1, "Nama lengkap wajib diisi.")
      .max(100, "Nama lengkap maksimal 100 karakter."),
    nisn: z.string().trim(),
    staffNumber: z.string().trim(),
    teacherMajorIds: z.array(z.string()),
  })
  .superRefine((values, ctx) => {
    if (values.accountType === "student" && !/^\d{10}$/.test(values.nisn)) {
      ctx.addIssue({
        code: "custom",
        path: ["nisn"],
        message: "NISN harus 10 digit angka.",
      });
    }

    if (
      values.accountType !== "student" &&
      values.staffNumber !== "" &&
      (values.staffNumber.length < 3 || values.staffNumber.length > 30)
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["staffNumber"],
        message: "NIP/ID staf harus 3–30 karakter.",
      });
    }
  });

type EditUserValues = z.infer<typeof editUserSchema>;

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

export function EditUserDialog({
  user,
  canEditGlobally,
  onClose,
}: {
  user: AdminUserDetail;
  canEditGlobally: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const majorsQuery = useGetAdminMajors({
    query: { enabled: canEditGlobally },
  });
  const majors = useMemo(
    () => majorsQuery.data?.items ?? [],
    [majorsQuery.data],
  );

  const form = useForm<EditUserValues>({
    resolver: zodResolver(editUserSchema),
    defaultValues: {
      accountType: user.accountType,
      displayName: user.displayName,
      nisn: user.nisn ?? "",
      staffNumber: user.staffNumber ?? "",
      teacherMajorIds: user.teacherMajorIds,
    },
  });

  const accountType = useWatch({
    control: form.control,
    name: "accountType",
  });

  const updateMutation = usePatchAdminUser({
    mutation: {
      onSuccess: () => {
        toast.success("Profil pengguna diperbarui");
        void queryClient.invalidateQueries({
          queryKey: getGetAdminUserQueryKey(user.id),
        });
        void queryClient.invalidateQueries({
          queryKey: getGetAdminUsersQueryKey(),
        });
        onClose();
      },
      onError: (error) => setFormError(applyApiError(form, error)),
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
          <DialogTitle>Ubah profil</DialogTitle>
          <DialogDescription>
            Perbarui data profil {user.displayName}.
          </DialogDescription>
        </DialogHeader>

        <FormError message={formError} />

        <Form {...form}>
          <form
            id="edit-user-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);

              const payload: AdminUserUpdate = {
                displayName: values.displayName,
              };

              if (values.accountType === "student") {
                payload.nisn = values.nisn;
                payload.staffNumber = "";
              } else {
                payload.nisn = "";
                payload.staffNumber = values.staffNumber;
              }

              if (canEditGlobally) {
                payload.accountType = values.accountType;

                if (values.accountType === "teacher") {
                  payload.teacherMajorIds = values.teacherMajorIds;
                }
              }

              updateMutation.mutate({ id: user.id, data: payload });
            })}
          >
            {canEditGlobally ? (
              <FormField
                control={form.control}
                name="accountType"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Tipe akun</FormLabel>
                    <Select
                      value={field.value}
                      onValueChange={(value: AccountType) => {
                        field.onChange(value);

                        if (value === "student") {
                          form.setValue("staffNumber", "");
                          form.setValue("teacherMajorIds", []);
                        } else {
                          form.setValue("nisn", "");
                        }
                      }}
                    >
                      <FormControl>
                        <SelectTrigger aria-label="Tipe akun">
                          <SelectValue placeholder="Pilih tipe akun" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {(["student", "teacher", "staff"] as const).map(
                          (type) => (
                            <SelectItem key={type} value={type}>
                              {ACCOUNT_TYPE_LABELS[type]}
                            </SelectItem>
                          ),
                        )}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ) : null}

            <FormField
              control={form.control}
              name="displayName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Nama lengkap</FormLabel>
                  <FormControl>
                    <Input
                      autoComplete="name"
                      maxLength={100}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {accountType === "student" ? (
              <FormField
                control={form.control}
                name="nisn"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>NISN</FormLabel>
                    <FormControl>
                      <Input
                        inputMode="numeric"
                        autoComplete="off"
                        placeholder="10 digit angka"
                        maxLength={10}
                        {...field}
                        onChange={(event) =>
                          field.onChange(
                            event.target.value.replace(/\D/g, "").slice(0, 10),
                          )
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ) : (
              <FormField
                control={form.control}
                name="staffNumber"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>NIP/ID staf (opsional)</FormLabel>
                    <FormControl>
                      <Input
                        autoComplete="off"
                        placeholder="3–30 karakter"
                        maxLength={30}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      Kosongkan jika tidak ada.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            {canEditGlobally && accountType === "teacher" ? (
              <FormField
                control={form.control}
                name="teacherMajorIds"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Jurusan diampu</FormLabel>
                    <FormControl>
                      <MajorCombobox
                        majors={majors}
                        value={field.value}
                        onChange={field.onChange}
                        disabled={majorsQuery.isPending}
                        placeholder="Pilih jurusan yang diampu"
                      />
                    </FormControl>
                    <FormDescription>
                      Guru dapat mengampu lebih dari satu jurusan.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            ) : null}
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
            form="edit-user-form"
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? "Menyimpan…" : "Simpan"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
