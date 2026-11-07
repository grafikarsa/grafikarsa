"use client";

import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { CircleAlert, Inbox, MoreHorizontal, Plus } from "lucide-react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import NotFound from "@/app/not-found";
import { AdminPageHeader } from "@/components/admin/admin-page-header";
import { DatePicker } from "@/components/admin/date-picker";
import { formatDateLong } from "@/components/admin/date";
import {
  apiErrorMessage,
  applyApiError,
} from "@/components/admin/form-error";
import { QueryError } from "@/components/admin/query-error";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
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
  getGetAdminAcademicYearsQueryKey,
  useGetAdminAcademicYears,
  usePatchAdminAcademicYear,
  usePostAdminAcademicYear,
  usePostAdminAcademicYearSetCurrent,
} from "@/lib/api/gen/endpoints";
import type { AcademicYear } from "@/lib/api/gen/models";
import { ApiError } from "@/lib/api/fetcher";
import { ADMIN_PERMISSIONS, can, canGlobally } from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

const dateField = z.string().min(1, "Tanggal wajib diisi.");

const yearSchema = z
  .object({
    startYear: z
      .number()
      .int("Tahun mulai harus bilangan bulat.")
      .min(2000, "Tahun mulai minimal 2000.")
      .max(2100, "Tahun mulai maksimal 2100."),
    startsOn: dateField,
    endsOn: dateField,
  })
  .superRefine((values, ctx) => {
    if (
      values.startsOn &&
      Number(values.startsOn.slice(0, 4)) !== values.startYear
    ) {
      ctx.addIssue({
        code: "custom",
        path: ["startsOn"],
        message: "Tanggal mulai harus berada di tahun ajaran yang dipilih.",
      });
    }

    if (values.startsOn && values.endsOn && values.endsOn <= values.startsOn) {
      ctx.addIssue({
        code: "custom",
        path: ["endsOn"],
        message: "Tanggal selesai harus setelah tanggal mulai.",
      });
    }
  });

type YearFormValues = z.infer<typeof yearSchema>;

const datesSchema = z
  .object({
    startsOn: dateField,
    endsOn: dateField,
  })
  .refine((values) => !values.startsOn || !values.endsOn || values.endsOn > values.startsOn, {
    path: ["endsOn"],
    message: "Tanggal selesai harus setelah tanggal mulai.",
  });

type DatesFormValues = z.infer<typeof datesSchema>;

function defaultYearValues(): YearFormValues {
  return {
    startYear: new Date().getFullYear(),
    startsOn: "",
    endsOn: "",
  };
}

function invalidateYears(queryClient: QueryClient) {
  void queryClient.invalidateQueries({
    queryKey: getGetAdminAcademicYearsQueryKey(),
  });
}

function FormError({ message }: { message: string | null }) {
  if (!message) {
    return null;
  }

  return (
    <Alert variant="destructive">
      <CircleAlert aria-hidden="true" />
      <AlertTitle>Gagal menyimpan</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

function CreateYearDialog({ onClose }: { onClose: () => void }) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm<YearFormValues>({
    resolver: zodResolver(yearSchema),
    defaultValues: defaultYearValues(),
  });

  const createMutation = usePostAdminAcademicYear({
    mutation: {
      onSuccess: () => {
        toast.success("Tahun ajaran ditambahkan");
        invalidateYears(queryClient);
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
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Tambah tahun ajaran</DialogTitle>
          <DialogDescription>
            Label tahun ajaran dibuat otomatis dari tahun mulai, misalnya
            2026/2027.
          </DialogDescription>
        </DialogHeader>

        <FormError message={formError} />

        <Form {...form}>
          <form
            id="create-year-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);
              createMutation.mutate({ data: values });
            })}
          >
            <FormField
              control={form.control}
              name="startYear"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Tahun mulai</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      inputMode="numeric"
                      min={2000}
                      max={2100}
                      value={field.value}
                      onChange={(event) =>
                        field.onChange(
                          event.target.value === ""
                            ? 0
                            : Number(event.target.value),
                        )
                      }
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="startsOn"
              render={({ field }) => (
                <FormItem className="flex flex-col">
                  <FormLabel>Tanggal mulai</FormLabel>
                  <FormControl>
                    <DatePicker
                      value={field.value}
                      onChange={field.onChange}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="endsOn"
              render={({ field }) => (
                <FormItem className="flex flex-col">
                  <FormLabel>Tanggal selesai</FormLabel>
                  <FormControl>
                    <DatePicker
                      value={field.value}
                      onChange={field.onChange}
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
            disabled={createMutation.isPending}
          >
            Batal
          </Button>
          <Button
            type="submit"
            form="create-year-form"
            disabled={createMutation.isPending}
          >
            {createMutation.isPending ? "Menyimpan…" : "Simpan"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function EditYearDialog({
  year,
  onClose,
}: {
  year: AcademicYear;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm<DatesFormValues>({
    resolver: zodResolver(datesSchema),
    defaultValues: {
      startsOn: year.startsOn,
      endsOn: year.endsOn,
    },
  });

  const updateMutation = usePatchAdminAcademicYear({
    mutation: {
      onSuccess: () => {
        toast.success("Tanggal tahun ajaran diperbarui");
        invalidateYears(queryClient);
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
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Ubah tanggal tahun ajaran</DialogTitle>
          <DialogDescription>
            Tahun ajaran {year.label}. Tanggal mulai harus berada pada tahun{" "}
            {year.startYear}.
          </DialogDescription>
        </DialogHeader>

        <FormError message={formError} />

        <Form {...form}>
          <form
            id="edit-year-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);
              updateMutation.mutate({ id: year.id, data: values });
            })}
          >
            <FormField
              control={form.control}
              name="startsOn"
              render={({ field }) => (
                <FormItem className="flex flex-col">
                  <FormLabel>Tanggal mulai</FormLabel>
                  <FormControl>
                    <DatePicker
                      value={field.value}
                      onChange={field.onChange}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="endsOn"
              render={({ field }) => (
                <FormItem className="flex flex-col">
                  <FormLabel>Tanggal selesai</FormLabel>
                  <FormControl>
                    <DatePicker
                      value={field.value}
                      onChange={field.onChange}
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
            form="edit-year-form"
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? "Menyimpan…" : "Simpan"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SetCurrentYearDialog({
  year,
  onClose,
}: {
  year: AcademicYear;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();

  const setCurrentMutation = usePostAdminAcademicYearSetCurrent({
    mutation: {
      onSuccess: () => {
        toast.success("Tahun ajaran berjalan diperbarui");
        invalidateYears(queryClient);
        onClose();
      },
      onError: (error) => {
        toast.error(
          apiErrorMessage(error, "Tahun ajaran gagal diubah."),
        );
      },
    },
  });

  return (
    <AlertDialog
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Jadikan tahun berjalan?</AlertDialogTitle>
          <AlertDialogDescription>
            Tahun ajaran {year.label} akan menjadi tahun ajaran aktif. Tahun
            ajaran lain otomatis tidak aktif.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={setCurrentMutation.isPending}>
            Batal
          </AlertDialogCancel>
          <AlertDialogAction
            disabled={setCurrentMutation.isPending}
            onClick={(event) => {
              event.preventDefault();
              setCurrentMutation.mutate({ id: year.id });
            }}
          >
            {setCurrentMutation.isPending
              ? "Memproses…"
              : "Jadikan tahun berjalan"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function TableSkeleton() {
  return (
    <>
      {Array.from({ length: 4 }).map((_, index) => (
        <TableRow key={index}>
          <TableCell>
            <Skeleton className="h-4 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-32" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-32" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-20 rounded-4xl" />
          </TableCell>
          <TableCell>
            <Skeleton className="ml-auto size-7" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

export function TahunAjaranView() {
  const { session, status } = useSession();
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<AcademicYear | null>(null);
  const [settingCurrent, setSettingCurrent] = useState<AcademicYear | null>(
    null,
  );

  const yearsQuery = useGetAdminAcademicYears();
  const years = yearsQuery.data?.items ?? [];
  const canManage =
    status === "authenticated" &&
    canGlobally(session, ADMIN_PERMISSIONS.manageYears);

  if (status !== "authenticated") {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-4 w-72" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!can(session, ADMIN_PERMISSIONS.manageYears)) {
    return <NotFound />;
  }

  if (
    yearsQuery.error instanceof ApiError &&
    yearsQuery.error.status === 404
  ) {
    return <NotFound />;
  }

  return (
    <div>
      <AdminPageHeader
        title="Tahun Ajaran"
        description="Kelola tahun ajaran, tanggal berlaku, dan tahun ajaran yang sedang berjalan."
        action={
          canManage ? (
            <Button onClick={() => setCreateOpen(true)}>
              <Plus aria-hidden="true" />
              Tambah tahun ajaran
            </Button>
          ) : null
        }
      />

      {yearsQuery.isError ? (
        <QueryError
          title="Gagal memuat tahun ajaran"
          message={apiErrorMessage(
            yearsQuery.error,
            "Daftar tahun ajaran tidak dapat dimuat.",
          )}
          onRetry={() => {
            void yearsQuery.refetch();
          }}
          isRetrying={yearsQuery.isFetching}
        />
      ) : (
        <div className="rounded-xl border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tahun ajaran</TableHead>
                <TableHead>Mulai</TableHead>
                <TableHead>Selesai</TableHead>
                <TableHead className="w-32">Status</TableHead>
                <TableHead className="w-12">
                  <span className="sr-only">Aksi</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {yearsQuery.isPending ? <TableSkeleton /> : null}

              {!yearsQuery.isPending && years.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="py-12 text-center">
                    <Inbox
                      aria-hidden="true"
                      className="mx-auto mb-2 size-6 text-muted-foreground"
                    />
                    <p className="font-medium">Belum ada tahun ajaran.</p>
                    <p className="mt-1 text-sm text-muted-foreground">
                      Tambahkan tahun ajaran pertama untuk memulai.
                    </p>
                  </TableCell>
                </TableRow>
              ) : null}

              {years.map((year) => (
                <TableRow key={year.id}>
                  <TableCell className="font-medium">{year.label}</TableCell>
                  <TableCell>{formatDateLong(year.startsOn)}</TableCell>
                  <TableCell>{formatDateLong(year.endsOn)}</TableCell>
                  <TableCell>
                    <Badge variant={year.isCurrent ? "default" : "secondary"}>
                      {year.isCurrent ? "Berjalan" : "Tidak aktif"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    {canManage ? (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label={`Aksi untuk tahun ajaran ${year.label}`}
                          >
                            <MoreHorizontal aria-hidden="true" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onSelect={() => setEditing(year)}>
                            Ubah tanggal
                          </DropdownMenuItem>
                          {!year.isCurrent ? (
                            <>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem
                                onSelect={() => setSettingCurrent(year)}
                              >
                                Jadikan tahun berjalan
                              </DropdownMenuItem>
                            </>
                          ) : null}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {createOpen ? (
        <CreateYearDialog onClose={() => setCreateOpen(false)} />
      ) : null}

      {editing ? (
        <EditYearDialog year={editing} onClose={() => setEditing(null)} />
      ) : null}

      {settingCurrent ? (
        <SetCurrentYearDialog
          year={settingCurrent}
          onClose={() => setSettingCurrent(null)}
        />
      ) : null}
    </div>
  );
}
