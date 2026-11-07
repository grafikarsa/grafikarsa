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
  FormDescription,
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
  getGetAdminMajorsQueryKey,
  getGetMajorsQueryKey,
  useGetAdminMajors,
  usePatchAdminMajor,
  usePostAdminMajor,
} from "@/lib/api/gen/endpoints";
import type { Major } from "@/lib/api/gen/models";
import { ApiError } from "@/lib/api/fetcher";
import { ADMIN_PERMISSIONS, can, canGlobally } from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

const majorSchema = z.object({
  code: z
    .string()
    .trim()
    .min(2, "Kode jurusan minimal 2 karakter.")
    .max(20, "Kode jurusan maksimal 20 karakter.")
    .regex(
      /^[a-z0-9-]+$/,
      "Gunakan huruf kecil, angka, dan tanda hubung saja.",
    ),
  name: z
    .string()
    .trim()
    .min(2, "Nama jurusan minimal 2 karakter.")
    .max(100, "Nama jurusan maksimal 100 karakter."),
  shortName: z
    .string()
    .trim()
    .min(2, "Nama singkat minimal 2 karakter.")
    .max(20, "Nama singkat maksimal 20 karakter.")
    .regex(/^[A-Z0-9]+$/, "Gunakan huruf kapital atau angka saja."),
  sortOrder: z
    .number()
    .int("Urutan harus bilangan bulat.")
    .min(0, "Urutan minimal 0.")
    .max(32767, "Urutan maksimal 32767."),
});

type MajorFormValues = z.infer<typeof majorSchema>;

const CREATE_DEFAULTS: MajorFormValues = {
  code: "",
  name: "",
  shortName: "",
  sortOrder: 0,
};

function invalidateMajors(queryClient: QueryClient) {
  void queryClient.invalidateQueries({
    queryKey: getGetAdminMajorsQueryKey(),
  });
  void queryClient.invalidateQueries({ queryKey: getGetMajorsQueryKey() });
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

function CreateMajorDialog({ onClose }: { onClose: () => void }) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm<MajorFormValues>({
    resolver: zodResolver(majorSchema),
    defaultValues: CREATE_DEFAULTS,
  });

  const createMutation = usePostAdminMajor({
    mutation: {
      onSuccess: () => {
        toast.success("Jurusan ditambahkan");
        invalidateMajors(queryClient);
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
          <DialogTitle>Tambah jurusan</DialogTitle>
          <DialogDescription>
            Jurusan baru langsung aktif dan grup cerminnya dibuat otomatis.
          </DialogDescription>
        </DialogHeader>

        <FormError message={formError} />

        <Form {...form}>
          <form
            id="create-major-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);
              createMutation.mutate({ data: values });
            })}
          >
            <FormField
              control={form.control}
              name="code"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Kode jurusan</FormLabel>
                  <FormControl>
                    <Input
                      autoCapitalize="none"
                      spellCheck={false}
                      placeholder="rpl"
                      {...field}
                      onChange={(event) =>
                        field.onChange(event.target.value.toLowerCase())
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    Huruf kecil, angka, dan tanda hubung (2–20 karakter).
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
                  <FormLabel>Nama jurusan</FormLabel>
                  <FormControl>
                    <Input placeholder="Rekayasa Perangkat Lunak" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="shortName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Nama singkat</FormLabel>
                  <FormControl>
                    <Input
                      autoCapitalize="characters"
                      placeholder="RPL"
                      {...field}
                      onChange={(event) =>
                        field.onChange(event.target.value.toUpperCase())
                      }
                    />
                  </FormControl>
                  <FormDescription>
                    Dipakai pada nama kelas, misalnya “XI RPL A”.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="sortOrder"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Urutan</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      inputMode="numeric"
                      min={0}
                      max={32767}
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
            form="create-major-form"
            disabled={createMutation.isPending}
          >
            {createMutation.isPending ? "Menyimpan…" : "Simpan"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function EditMajorDialog({
  major,
  onClose,
}: {
  major: Major;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm<MajorFormValues>({
    resolver: zodResolver(majorSchema),
    defaultValues: {
      code: major.code,
      name: major.name,
      shortName: major.shortName,
      sortOrder: major.sortOrder,
    },
  });

  const updateMutation = usePatchAdminMajor({
    mutation: {
      onSuccess: () => {
        toast.success("Jurusan diperbarui");
        invalidateMajors(queryClient);
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
          <DialogTitle>Ubah jurusan</DialogTitle>
          <DialogDescription>
            Kode jurusan <span className="font-medium">{major.code}</span> tidak
            dapat diubah.
          </DialogDescription>
        </DialogHeader>

        <FormError message={formError} />

        <Form {...form}>
          <form
            id="edit-major-form"
            className="space-y-4"
            onSubmit={form.handleSubmit((values) => {
              setFormError(null);
              updateMutation.mutate({
                id: major.id,
                data: {
                  name: values.name,
                  shortName: values.shortName,
                  sortOrder: values.sortOrder,
                },
              });
            })}
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Nama jurusan</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="shortName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Nama singkat</FormLabel>
                  <FormControl>
                    <Input
                      autoCapitalize="characters"
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

            <FormField
              control={form.control}
              name="sortOrder"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Urutan</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      inputMode="numeric"
                      min={0}
                      max={32767}
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
            form="edit-major-form"
            disabled={updateMutation.isPending}
          >
            {updateMutation.isPending ? "Menyimpan…" : "Simpan"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ToggleMajorDialog({
  major,
  onClose,
}: {
  major: Major;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const deactivating = major.isActive;

  const toggleMutation = usePatchAdminMajor({
    mutation: {
      onSuccess: () => {
        toast.success(
          deactivating ? "Jurusan dinonaktifkan" : "Jurusan diaktifkan",
        );
        invalidateMajors(queryClient);
        onClose();
      },
      onError: (error) => {
        toast.error(apiErrorMessage(error, "Jurusan gagal diperbarui."));
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
          <AlertDialogTitle>
            {deactivating ? "Nonaktifkan jurusan?" : "Aktifkan jurusan?"}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {deactivating
              ? `Jurusan ${major.name} tidak akan muncul di daftar publik. Grup jurusan dipindahkan ke riwayat, tetapi data lama tetap tersimpan.`
              : `Jurusan ${major.name} akan muncul kembali di daftar publik dan grupnya kembali aktif.`}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={toggleMutation.isPending}>
            Batal
          </AlertDialogCancel>
          <AlertDialogAction
            variant={deactivating ? "destructive" : "default"}
            disabled={toggleMutation.isPending}
            onClick={(event) => {
              event.preventDefault();
              toggleMutation.mutate({
                id: major.id,
                data: { isActive: !major.isActive },
              });
            }}
          >
            {toggleMutation.isPending
              ? "Memproses…"
              : deactivating
                ? "Nonaktifkan"
                : "Aktifkan"}
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
            <Skeleton className="h-4 w-8" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-16" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-40" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-4 w-12" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-16 rounded-4xl" />
          </TableCell>
          <TableCell>
            <Skeleton className="ml-auto size-7" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

export function JurusanView() {
  const { session, status } = useSession();
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<Major | null>(null);
  const [toggling, setToggling] = useState<Major | null>(null);

  const majorsQuery = useGetAdminMajors();
  const majors = majorsQuery.data?.items ?? [];
  const canManage =
    status === "authenticated" &&
    canGlobally(session, ADMIN_PERMISSIONS.manageMajors);

  if (status !== "authenticated") {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-4 w-72" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!can(session, ADMIN_PERMISSIONS.manageMajors)) {
    return <NotFound />;
  }

  const notFound =
    majorsQuery.error instanceof ApiError && majorsQuery.error.status === 404;

  if (notFound) {
    return <NotFound />;
  }

  return (
    <div>
      <AdminPageHeader
        title="Jurusan"
        description="Kelola daftar jurusan yang dipakai pada kelas dan grup jurusan."
        action={
          canManage ? (
            <Button onClick={() => setCreateOpen(true)}>
              <Plus aria-hidden="true" />
              Tambah jurusan
            </Button>
          ) : null
        }
      />

      {majorsQuery.isError ? (
        <QueryError
          title="Gagal memuat jurusan"
          message={apiErrorMessage(
            majorsQuery.error,
            "Daftar jurusan tidak dapat dimuat.",
          )}
          onRetry={() => {
            void majorsQuery.refetch();
          }}
          isRetrying={majorsQuery.isFetching}
        />
      ) : (
        <div className="rounded-xl border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">Urutan</TableHead>
                <TableHead className="w-24">Kode</TableHead>
                <TableHead>Nama</TableHead>
                <TableHead className="w-28">Singkatan</TableHead>
                <TableHead className="w-28">Status</TableHead>
                <TableHead className="w-12">
                  <span className="sr-only">Aksi</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {majorsQuery.isPending ? <TableSkeleton /> : null}

              {!majorsQuery.isPending && majors.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-12 text-center">
                    <Inbox
                      aria-hidden="true"
                      className="mx-auto mb-2 size-6 text-muted-foreground"
                    />
                    <p className="font-medium">Belum ada jurusan.</p>
                    <p className="mt-1 text-sm text-muted-foreground">
                      Tambahkan jurusan pertama untuk memulai.
                    </p>
                  </TableCell>
                </TableRow>
              ) : null}

              {majors.map((major) => (
                <TableRow key={major.id}>
                  <TableCell className="tabular-nums">
                    {major.sortOrder}
                  </TableCell>
                  <TableCell className="font-mono text-xs">
                    {major.code}
                  </TableCell>
                  <TableCell className="font-medium">{major.name}</TableCell>
                  <TableCell>{major.shortName}</TableCell>
                  <TableCell>
                    <Badge variant={major.isActive ? "default" : "secondary"}>
                      {major.isActive ? "Aktif" : "Nonaktif"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    {canManage ? (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label={`Aksi untuk ${major.name}`}
                          >
                            <MoreHorizontal aria-hidden="true" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onSelect={() => setEditing(major)}
                          >
                            Ubah
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            variant={
                              major.isActive ? "destructive" : "default"
                            }
                            onSelect={() => setToggling(major)}
                          >
                            {major.isActive ? "Nonaktifkan" : "Aktifkan"}
                          </DropdownMenuItem>
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
        <CreateMajorDialog onClose={() => setCreateOpen(false)} />
      ) : null}

      {editing ? (
        <EditMajorDialog
          major={editing}
          onClose={() => setEditing(null)}
        />
      ) : null}

      {toggling ? (
        <ToggleMajorDialog
          major={toggling}
          onClose={() => setToggling(null)}
        />
      ) : null}
    </div>
  );
}
