"use client";

import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
  CircleAlert,
  Inbox,
  MoreHorizontal,
  Plus,
  Trash2,
} from "lucide-react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import NotFound from "@/app/not-found";
import { AdminPageHeader } from "@/components/admin/admin-page-header";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  getGetAdminClassesQueryKey,
  useDeleteAdminClass,
  useGetAdminAcademicYears,
  useGetAdminClasses,
  useGetAdminMajors,
  usePatchAdminClass,
  usePostAdminClass,
} from "@/lib/api/gen/endpoints";
import type {
  Class as SchoolClass,
  ClassCreateGrade,
  Major,
} from "@/lib/api/gen/models";
import { ApiError } from "@/lib/api/fetcher";
import {
  ADMIN_PERMISSIONS,
  can,
  canForMajor,
  canGlobally,
} from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

const GRADE_OPTIONS = [
  { value: "10", label: "X" },
  { value: "11", label: "XI" },
  { value: "12", label: "XII" },
] as const;

const GRADE_LABELS: Record<number, string> = {
  10: "X",
  11: "XI",
  12: "XII",
};

const sectionField = z
  .string()
  .trim()
  .min(1, "Rombel wajib diisi.")
  .max(2, "Rombel maksimal 2 karakter.")
  .regex(/^[A-Z]{1,2}$/, "Rombel harus 1–2 huruf kapital.");

const createClassSchema = z.object({
  academicYearId: z.string().min(1, "Tahun ajaran wajib dipilih."),
  majorId: z.string().min(1, "Jurusan wajib dipilih."),
  grade: z.enum(["10", "11", "12"]),
  section: sectionField,
});

type CreateClassValues = z.infer<typeof createClassSchema>;

const editClassSchema = z.object({
  section: sectionField,
});

type EditClassValues = z.infer<typeof editClassSchema>;

function invalidateClasses(queryClient: QueryClient) {
  void queryClient.invalidateQueries({
    queryKey: getGetAdminClassesQueryKey(),
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

function CreateClassDialog({
  years,
  majors,
  defaultYearId,
  defaultMajorId,
  onClose,
}: {
  years: { id: string; label: string; isCurrent: boolean }[];
  majors: Major[];
  defaultYearId: string;
  defaultMajorId: string;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const fallbackYear =
    years.find((year) => year.id === defaultYearId)?.id ??
    years.find((year) => year.isCurrent)?.id ??
    years[0]?.id ??
    "";

  const form = useForm<CreateClassValues>({
    resolver: zodResolver(createClassSchema),
    defaultValues: {
      academicYearId: fallbackYear,
      majorId: defaultMajorId,
      grade: "10",
      section: "",
    },
  });

  const createMutation = usePostAdminClass({
    mutation: {
      onSuccess: () => {
        toast.success("Kelas ditambahkan");
        invalidateClasses(queryClient);
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
          <DialogTitle>Tambah kelas</DialogTitle>
          <DialogDescription>
            Kelas baru langsung mendapat grup cermin otomatis.
          </DialogDescription>
        </DialogHeader>

        <FormError message={formError} />

        <Form {...form}>
          <form
            id="create-class-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);
              createMutation.mutate({
                data: {
                  academicYearId: values.academicYearId,
                  majorId: values.majorId,
                  grade: Number(values.grade) as ClassCreateGrade,
                  section: values.section,
                },
              });
            })}
          >
            <FormField
              control={form.control}
              name="academicYearId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Tahun ajaran</FormLabel>
                  <Select
                    value={field.value}
                    onValueChange={field.onChange}
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
                    onValueChange={field.onChange}
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

            <div className="grid gap-4 sm:grid-cols-2">
              <FormField
                control={form.control}
                name="grade"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Tingkat</FormLabel>
                    <Select
                      value={field.value}
                      onValueChange={field.onChange}
                    >
                      <FormControl>
                        <SelectTrigger aria-label="Tingkat">
                          <SelectValue placeholder="Pilih tingkat" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {GRADE_OPTIONS.map((option) => (
                          <SelectItem key={option.value} value={option.value}>
                            {option.label}
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
                name="section"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Rombel</FormLabel>
                    <FormControl>
                      <Input
                        autoCapitalize="characters"
                        placeholder="A"
                        maxLength={2}
                        {...field}
                        onChange={(event) =>
                          field.onChange(event.target.value.toUpperCase())
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
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
            form="create-class-form"
            disabled={createMutation.isPending}
          >
            {createMutation.isPending ? "Menyimpan…" : "Simpan"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function EditClassDialog({
  schoolClass,
  onClose,
}: {
  schoolClass: SchoolClass;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm<EditClassValues>({
    resolver: zodResolver(editClassSchema),
    defaultValues: { section: schoolClass.section },
  });

  const updateMutation = usePatchAdminClass({
    mutation: {
      onSuccess: () => {
        toast.success("Rombel kelas diperbarui");
        invalidateClasses(queryClient);
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
          <DialogTitle>Ubah rombel</DialogTitle>
          <DialogDescription>
            Ubah rombel kelas {schoolClass.name}. Nama kelas ikut diperbarui
            otomatis.
          </DialogDescription>
        </DialogHeader>

        <FormError message={formError} />

        <Form {...form}>
          <form
            id="edit-class-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);
              updateMutation.mutate({
                id: schoolClass.id,
                data: { section: values.section },
              });
            })}
          >
            <FormField
              control={form.control}
              name="section"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Rombel</FormLabel>
                  <FormControl>
                    <Input
                      autoCapitalize="characters"
                      placeholder="A"
                      maxLength={2}
                      {...field}
                      onChange={(event) =>
                        field.onChange(event.target.value.toUpperCase())
                      }
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
            form="edit-class-form"
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? "Menyimpan…" : "Simpan"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function DeleteClassDialog({
  schoolClass,
  onClose,
}: {
  schoolClass: SchoolClass;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();

  const deleteMutation = useDeleteAdminClass({
    mutation: {
      onSuccess: () => {
        toast.success("Kelas dihapus");
        invalidateClasses(queryClient);
        onClose();
      },
      onError: (error) => {
        toast.error(apiErrorMessage(error, "Kelas gagal dihapus."));
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
          <AlertDialogTitle>Hapus kelas?</AlertDialogTitle>
          <AlertDialogDescription>
            Kelas {schoolClass.name} akan dihapus permanen. Tindakan ini tidak
            dapat dibatalkan.
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
              deleteMutation.mutate({ id: schoolClass.id });
            }}
          >
            {deleteMutation.isPending ? "Menghapus…" : "Hapus"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function TableSkeleton() {
  return (
    <>
      {Array.from({ length: 5 }).map((_, index) => (
        <TableRow key={index}>
          <TableCell>
            <Skeleton className="h-4 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-8" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-8" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-32" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-8" />
          </TableCell>
          <TableCell>
            <Skeleton className="ml-auto size-7" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

export function KelasView() {
  const { session, status } = useSession();
  const [academicYearId, setAcademicYearId] = useState("");
  const [majorFilter, setMajorFilter] = useState("all");
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<SchoolClass | null>(null);
  const [deleting, setDeleting] = useState<SchoolClass | null>(null);

  const yearsQuery = useGetAdminAcademicYears();
  const majorsQuery = useGetAdminMajors();

  const years = yearsQuery.data?.items ?? [];
  const allMajors = majorsQuery.data?.items ?? [];

  const isGlobal =
    status === "authenticated" &&
    canGlobally(session, ADMIN_PERMISSIONS.manageClasses);
  const managedMajors = allMajors.filter(
    (major) =>
      isGlobal ||
      (status === "authenticated" &&
        canForMajor(
          session,
          ADMIN_PERMISSIONS.manageClasses,
          major.id,
        )),
  );
  const managedMajorIds = new Set(managedMajors.map((major) => major.id));
  const majorById = new Map(allMajors.map((major) => [major.id, major]));

  const effectiveYearId =
    academicYearId && years.some((year) => year.id === academicYearId)
      ? academicYearId
      : (years.find((year) => year.isCurrent)?.id ?? years[0]?.id ?? "");

  const effectiveMajorFilter =
    majorFilter !== "all" &&
    managedMajors.some((major) => major.id === majorFilter)
      ? majorFilter
      : "all";

  const classesQuery = useGetAdminClasses(
    {
      academicYearId: effectiveYearId || undefined,
      majorId:
        effectiveMajorFilter === "all" ? undefined : effectiveMajorFilter,
    },
    {
      query: {
        enabled:
          status === "authenticated" &&
          can(session, ADMIN_PERMISSIONS.manageClasses) &&
          Boolean(effectiveYearId),
      },
    },
  );

  const classes = (classesQuery.data?.items ?? []).filter(
    (schoolClass) => isGlobal || managedMajorIds.has(schoolClass.majorId),
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

  if (!can(session, ADMIN_PERMISSIONS.manageClasses)) {
    return <NotFound />;
  }

  if (
    classesQuery.error instanceof ApiError &&
    classesQuery.error.status === 404
  ) {
    return <NotFound />;
  }

  const canAdd = years.length > 0 && managedMajors.length > 0;
  const showSkeleton =
    Boolean(effectiveYearId) && classesQuery.isPending;

  return (
    <div>
      <AdminPageHeader
        title="Kelas"
        description="Kelola rombel per tahun ajaran dan jurusan yang menjadi wewenangmu."
        action={
          <Button
            onClick={() => setCreateOpen(true)}
            disabled={!canAdd}
            title={
              canAdd
                ? undefined
                : "Tambahkan tahun ajaran dan jurusan terlebih dahulu."
            }
          >
            <Plus aria-hidden="true" />
            Tambah kelas
          </Button>
        }
      />

      <div className="grid gap-3 pb-4 sm:flex sm:flex-wrap sm:items-end">
        <div className="grid gap-1.5">
          <span className="text-sm font-medium">Tahun ajaran</span>
          <Select
            value={effectiveYearId}
            onValueChange={setAcademicYearId}
            disabled={yearsQuery.isPending || years.length === 0}
          >
            <SelectTrigger className="w-full sm:w-48" aria-label="Tahun ajaran">
              <SelectValue placeholder="Pilih tahun ajaran" />
            </SelectTrigger>
            <SelectContent>
              {years.map((year) => (
                <SelectItem key={year.id} value={year.id}>
                  {year.label}
                  {year.isCurrent ? " (berjalan)" : ""}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="grid gap-1.5">
          <span className="text-sm font-medium">Jurusan</span>
          <Select
            value={effectiveMajorFilter}
            onValueChange={setMajorFilter}
            disabled={majorsQuery.isPending || managedMajors.length === 0}
          >
            <SelectTrigger className="w-full sm:w-56" aria-label="Jurusan">
              <SelectValue placeholder="Semua jurusan" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua jurusan</SelectItem>
              {managedMajors.map((major) => (
                <SelectItem key={major.id} value={major.id}>
                  {major.name}
                  {major.isActive ? "" : " (nonaktif)"}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {yearsQuery.isSuccess && years.length === 0 ? (
        <Alert className="mb-4">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Belum ada tahun ajaran</AlertTitle>
          <AlertDescription>
            Tambahkan tahun ajaran di halaman Tahun Ajaran sebelum membuat
            kelas.
          </AlertDescription>
        </Alert>
      ) : null}

      {classesQuery.isError ? (
        <QueryError
          title="Gagal memuat kelas"
          message={apiErrorMessage(
            classesQuery.error,
            "Daftar kelas tidak dapat dimuat.",
          )}
          onRetry={() => {
            void classesQuery.refetch();
          }}
          isRetrying={classesQuery.isFetching}
        />
      ) : (
        <div className="rounded-xl border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama kelas</TableHead>
                <TableHead className="w-20">Tingkat</TableHead>
                <TableHead className="w-20">Rombel</TableHead>
                <TableHead>Jurusan</TableHead>
                <TableHead className="w-28">Jumlah siswa</TableHead>
                <TableHead className="w-12">
                  <span className="sr-only">Aksi</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {showSkeleton ? <TableSkeleton /> : null}

              {!showSkeleton && classes.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-12 text-center">
                    <Inbox
                      aria-hidden="true"
                      className="mx-auto mb-2 size-6 text-muted-foreground"
                    />
                    <p className="font-medium">Belum ada kelas.</p>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {effectiveYearId
                        ? "Tidak ada kelas pada tahun ajaran dan jurusan ini."
                        : "Pilih tahun ajaran untuk melihat kelas."}
                    </p>
                  </TableCell>
                </TableRow>
              ) : null}

              {classes.map((schoolClass) => {
                const major = majorById.get(schoolClass.majorId);
                const hasStudents = schoolClass.enrollmentCount > 0;

                return (
                  <TableRow key={schoolClass.id}>
                    <TableCell className="font-medium">
                      {schoolClass.name}
                    </TableCell>
                    <TableCell>
                      {GRADE_LABELS[schoolClass.grade] ?? schoolClass.grade}
                    </TableCell>
                    <TableCell>{schoolClass.section}</TableCell>
                    <TableCell>
                      {major?.name ?? schoolClass.majorCode}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {schoolClass.enrollmentCount}
                    </TableCell>
                    <TableCell className="text-right">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label={`Aksi untuk kelas ${schoolClass.name}`}
                          >
                            <MoreHorizontal aria-hidden="true" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onSelect={() => setEditing(schoolClass)}
                          >
                            Ubah rombel
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          {hasStudents ? (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="block">
                                  <DropdownMenuItem
                                    variant="destructive"
                                    disabled
                                    className="w-full"
                                  >
                                    <Trash2 aria-hidden="true" />
                                    Hapus
                                  </DropdownMenuItem>
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="left">
                                Kelas sudah punya siswa
                              </TooltipContent>
                            </Tooltip>
                          ) : (
                            <DropdownMenuItem
                              variant="destructive"
                              onSelect={() => setDeleting(schoolClass)}
                            >
                              <Trash2 aria-hidden="true" />
                              Hapus
                            </DropdownMenuItem>
                          )}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {createOpen ? (
        <CreateClassDialog
          years={years}
          majors={managedMajors}
          defaultYearId={effectiveYearId}
          defaultMajorId={
            effectiveMajorFilter === "all" ? "" : effectiveMajorFilter
          }
          onClose={() => setCreateOpen(false)}
        />
      ) : null}

      {editing ? (
        <EditClassDialog
          schoolClass={editing}
          onClose={() => setEditing(null)}
        />
      ) : null}

      {deleting ? (
        <DeleteClassDialog
          schoolClass={deleting}
          onClose={() => setDeleting(null)}
        />
      ) : null}
    </div>
  );
}
