"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { CircleAlert } from "lucide-react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import {
  apiErrorMessage,
  applyApiError,
} from "@/components/admin/form-error";
import {
  ACCOUNT_TYPE_LABELS,
  loginIdentifier,
} from "@/components/admin/users/labels";
import { MajorCombobox } from "@/components/admin/users/major-combobox";
import { OneTimePassword } from "@/components/admin/users/one-time-password";
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
  getGetAdminUsersQueryKey,
  useGetAdminAcademicYears,
  useGetAdminClasses,
  useGetAdminMajors,
  usePostAdminUser,
} from "@/lib/api/gen/endpoints";
import type {
  AcademicYear,
  AccountType,
  AdminUserCreated,
  AdminUserCreate,
  Major,
} from "@/lib/api/gen/models";
import { useSession } from "@/lib/auth/session";

const accountTypeValues = ["student", "teacher", "staff"] as const;

const createUserSchema = z
  .object({
    accountType: z.enum(accountTypeValues),
    displayName: z
      .string()
      .trim()
      .min(1, "Nama lengkap wajib diisi.")
      .max(100, "Nama lengkap maksimal 100 karakter."),
    nisn: z.string().trim(),
    staffNumber: z.string().trim(),
    academicYearId: z.string(),
    majorId: z.string(),
    classId: z.string(),
    teacherMajorIds: z.array(z.string()),
  })
  .superRefine((values, ctx) => {
    if (values.accountType === "student") {
      if (!/^\d{10}$/.test(values.nisn)) {
        ctx.addIssue({
          code: "custom",
          path: ["nisn"],
          message: "NISN harus 10 digit angka.",
        });
      }

      if (!values.classId) {
        ctx.addIssue({
          code: "custom",
          path: ["classId"],
          message: "Kelas wajib dipilih untuk siswa.",
        });
      }
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

type CreateUserValues = z.infer<typeof createUserSchema>;

function FormError({ message }: { message: string | null }) {
  if (!message) {
    return null;
  }

  return (
    <Alert variant="destructive">
      <CircleAlert aria-hidden="true" />
      <AlertTitle>Gagal menambahkan pengguna</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

function CreateUserForm({
  years,
  majors,
  globalCreate,
  onCreated,
  onClose,
}: {
  years: AcademicYear[];
  majors: Major[];
  globalCreate: boolean;
  onCreated: (created: AdminUserCreated) => void;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const defaultYear =
    years.find((year) => year.isCurrent)?.id ?? years[0]?.id ?? "";

  const form = useForm<CreateUserValues>({
    resolver: zodResolver(createUserSchema),
    defaultValues: {
      accountType: "student",
      displayName: "",
      nisn: "",
      staffNumber: "",
      academicYearId: defaultYear,
      majorId: majors.length === 1 ? majors[0].id : "",
      classId: "",
      teacherMajorIds: [],
    },
  });

  const accountType = useWatch({
    control: form.control,
    name: "accountType",
  });
  const academicYearId = useWatch({
    control: form.control,
    name: "academicYearId",
  });
  const majorId = useWatch({ control: form.control, name: "majorId" });

  const classesQuery = useGetAdminClasses(
    {
      academicYearId: academicYearId || undefined,
      majorId: majorId || undefined,
    },
    {
      query: {
        enabled:
          accountType === "student" &&
          Boolean(academicYearId) &&
          Boolean(majorId),
      },
    },
  );

  const classes = classesQuery.data?.items ?? [];
  const missingStudentData =
    accountType === "student" && (years.length === 0 || majors.length === 0);

  const createMutation = usePostAdminUser({
    mutation: {
      onSuccess: (data) => {
        void queryClient.invalidateQueries({
          queryKey: getGetAdminUsersQueryKey(),
        });
        toast.success("Akun dibuat");
        onCreated(data);
      },
      onError: (error) => setFormError(applyApiError(form, error)),
    },
  });

  const accountTypeOptions: AccountType[] = globalCreate
    ? ["student", "teacher", "staff"]
    : ["student"];

  return (
    <>
      <FormError message={formError} />

      {missingStudentData ? (
        <Alert>
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Data akademik belum lengkap</AlertTitle>
          <AlertDescription>
            Tambahkan tahun ajaran dan jurusan terlebih dahulu agar siswa dapat
            dibuat.
          </AlertDescription>
        </Alert>
      ) : null}

      <Form {...form}>
        <form
          id="create-user-form"
          className="space-y-4"
          onSubmit={form.handleSubmit((values) => {
            setFormError(null);

            const payload: AdminUserCreate = {
              accountType: values.accountType,
              displayName: values.displayName,
            };

            if (values.accountType === "student") {
              payload.nisn = values.nisn;
              payload.classId = values.classId;
            }

            if (
              values.accountType !== "student" &&
              values.staffNumber !== ""
            ) {
              payload.staffNumber = values.staffNumber;
            }

            if (
              values.accountType === "teacher" &&
              values.teacherMajorIds.length > 0
            ) {
              payload.teacherMajorIds = values.teacherMajorIds;
            }

            createMutation.mutate({ data: payload });
          })}
        >
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
                      form.setValue("classId", "");
                    }
                  }}
                >
                  <FormControl>
                    <SelectTrigger aria-label="Tipe akun">
                      <SelectValue placeholder="Pilih tipe akun" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {accountTypeOptions.map((type) => (
                      <SelectItem key={type} value={type}>
                        {ACCOUNT_TYPE_LABELS[type]}
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
            name="displayName"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Nama lengkap</FormLabel>
                <FormControl>
                  <Input
                    autoComplete="name"
                    placeholder="mis. Budi Santoso"
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
                  <FormDescription>
                    Dipakai untuk masuk sebelum pengguna memilih username.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          ) : null}

          {accountType === "teacher" || accountType === "staff" ? (
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
                    Dipakai untuk masuk jika diisi.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          ) : null}

          {accountType === "student" ? (
            <div className="space-y-4 rounded-xl border p-3">
              <div className="grid gap-4 sm:grid-cols-2">
                <FormField
                  control={form.control}
                  name="academicYearId"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Tahun ajaran</FormLabel>
                      <Select
                        value={field.value}
                        onValueChange={(value) => {
                          field.onChange(value);
                          form.setValue("classId", "");
                        }}
                      >
                        <FormControl>
                          <SelectTrigger aria-label="Tahun ajaran">
                            <SelectValue placeholder="Pilih tahun ajaran" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {years.map((year) => (
                            <SelectItem key={year.id} value={year.id}>
                              {year.label}
                              {year.isCurrent ? " (berjalan)" : ""}
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
                  name="majorId"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Jurusan</FormLabel>
                      <Select
                        value={field.value}
                        onValueChange={(value) => {
                          field.onChange(value);
                          form.setValue("classId", "");
                        }}
                      >
                        <FormControl>
                          <SelectTrigger aria-label="Jurusan">
                            <SelectValue placeholder="Pilih jurusan" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {majors.map((major) => (
                            <SelectItem key={major.id} value={major.id}>
                              {major.name}
                              {major.isActive ? "" : " (nonaktif)"}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <FormField
                control={form.control}
                name="classId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Kelas</FormLabel>
                    <Select
                      value={field.value}
                      onValueChange={field.onChange}
                      disabled={
                        !academicYearId ||
                        !majorId ||
                        classesQuery.isPending
                      }
                    >
                      <FormControl>
                        <SelectTrigger aria-label="Kelas">
                          <SelectValue
                            placeholder={
                              classesQuery.isPending
                                ? "Memuat kelas…"
                                : "Pilih kelas"
                            }
                          />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {classes.map((schoolClass) => (
                          <SelectItem
                            key={schoolClass.id}
                            value={schoolClass.id}
                          >
                            {schoolClass.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {!classesQuery.isPending && classes.length === 0 ? (
                      <FormDescription>
                        Belum ada kelas pada tahun ajaran dan jurusan ini.
                      </FormDescription>
                    ) : null}
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          ) : null}

          {accountType === "teacher" ? (
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
                      disabled={majors.length === 0}
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
          disabled={createMutation.isPending}
        >
          Batal
        </Button>
        <Button
          type="submit"
          form="create-user-form"
          disabled={createMutation.isPending || missingStudentData}
        >
          {createMutation.isPending ? "Menyimpan…" : "Simpan"}
        </Button>
      </DialogFooter>
    </>
  );
}

function CreatedUserDialog({
  created,
  onClose,
}: {
  created: AdminUserCreated;
  onClose: () => void;
}) {
  const identifier = loginIdentifier(created.user);

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
          <DialogTitle>Akun dibuat</DialogTitle>
          <DialogDescription>
            {created.user.displayName} ·{" "}
            {ACCOUNT_TYPE_LABELS[created.user.accountType]}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1 rounded-lg border p-3 text-sm">
            <p className="font-medium">Informasi masuk</p>
            <p className="text-muted-foreground">
              {identifier
                ? `Masuk dengan ${identifier}.`
                : "Pengguna masuk dengan NISN atau NIP/ID staf yang terdaftar."}
            </p>
            <p className="text-muted-foreground">
              Username akan dipilih pengguna saat aktivasi.
            </p>
          </div>

          <OneTimePassword
            password={created.initialPassword}
            expiresAt={created.user.passwordExpiresAt}
            label="Sandi awal"
          />
        </div>

        <DialogFooter>
          <Button variant="outline" asChild>
            <Link href={`/admin/pengguna/${created.user.id}`}>
              Buka detail
            </Link>
          </Button>
          <Button onClick={onClose}>Selesai</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function CreateUserDialog({ onClose }: { onClose: () => void }) {
  const { session, status } = useSession();
  const [created, setCreated] = useState<AdminUserCreated | null>(null);

  const scope = useMemo(
    () => permissionScope(session, "user.create"),
    [session],
  );

  const majorsQuery = useGetAdminMajors({
    query: { enabled: status === "authenticated" },
  });
  const yearsQuery = useGetAdminAcademicYears({
    query: { enabled: status === "authenticated" },
  });

  const majors = useMemo(() => {
    const items = majorsQuery.data?.items ?? [];

    return scope.global
      ? items
      : items.filter((major) => scope.majorIds.includes(major.id));
  }, [majorsQuery.data, scope.global, scope.majorIds]);

  const years = useMemo(() => yearsQuery.data?.items ?? [], [yearsQuery.data]);

  if (created) {
    return <CreatedUserDialog created={created} onClose={onClose} />;
  }

  const loading = majorsQuery.isPending || yearsQuery.isPending;
  const error = majorsQuery.error ?? yearsQuery.error;

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
          <DialogTitle>Tambah pengguna</DialogTitle>
          <DialogDescription>
            Sandi awal dibuat otomatis dan hanya ditampilkan sekali setelah
            akun tersimpan.
          </DialogDescription>
        </DialogHeader>

        {loading ? (
          <div className="space-y-3">
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-full" />
          </div>
        ) : null}

        {!loading && error ? (
          <Alert variant="destructive">
            <CircleAlert aria-hidden="true" />
            <AlertTitle>Data akademik tidak dapat dimuat</AlertTitle>
            <AlertDescription>
              {apiErrorMessage(
                error,
                "Tutup dialog lalu coba lagi beberapa saat lagi.",
              )}
            </AlertDescription>
          </Alert>
        ) : null}

        {!loading ? (
          <CreateUserForm
            years={years}
            majors={majors}
            globalCreate={scope.global}
            onCreated={setCreated}
            onClose={onClose}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
