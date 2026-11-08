"use client";

import { useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  CircleAlert,
  CircleCheck,
  FileWarning,
  Inbox,
} from "lucide-react";
import { toast } from "sonner";

import { AccountCardsSection } from "@/components/admin/import/account-cards";
import { apiErrorMessage } from "@/components/admin/form-error";
import {
  importFieldLabel,
  importKindLabel,
  IMPORT_STATUS_LABELS,
  IMPORT_STATUS_VARIANTS,
} from "@/components/admin/import/labels";
import { formatDateTime } from "@/components/admin/users/labels";
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
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  getGetAdminImportQueryKey,
  getGetAdminImportsQueryKey,
  usePostAdminImportCommit,
} from "@/lib/api/gen/endpoints";
import type { ImportJob } from "@/lib/api/gen/models";

function ImportStatusBadge({ job }: { job: ImportJob }) {
  return (
    <Badge variant={IMPORT_STATUS_VARIANTS[job.status]}>
      {IMPORT_STATUS_LABELS[job.status]}
    </Badge>
  );
}

function ReadySection({ job }: { job: ImportJob }) {
  const queryClient = useQueryClient();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const commitMutation = usePostAdminImportCommit({
    mutation: {
      onSuccess: (updated) => {
        queryClient.setQueryData(
          getGetAdminImportQueryKey(updated.id),
          updated,
        );
        void queryClient.invalidateQueries({
          queryKey: getGetAdminImportsQueryKey(),
        });
        setConfirmOpen(false);
        toast.success("Impor dijalankan");
      },
    },
  });

  const validRows = job.validRows ?? 0;
  const byClass = Object.entries(job.byClass).sort(([a], [b]) =>
    a.localeCompare(b, "id"),
  );
  const commitError = commitMutation.isError
    ? apiErrorMessage(
        commitMutation.error,
        "Impor tidak dapat dijalankan. Coba lagi.",
      )
    : null;

  return (
    <div className="space-y-4">
      <Alert className="border-emerald-600/30 bg-emerald-500/10 text-emerald-800 dark:border-emerald-400/30 dark:bg-emerald-400/10 dark:text-emerald-200">
        <CircleCheck aria-hidden="true" />
        <AlertTitle>Semua {validRows} baris valid</AlertTitle>
        <AlertDescription className="text-emerald-700/90 dark:text-emerald-200/80">
          Periksa ringkasan di bawah, lalu jalankan impor.
        </AlertDescription>
      </Alert>

      {job.kind === "students" ? (
        byClass.length > 0 ? (
          <div className="overflow-hidden rounded-xl border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Kelas</TableHead>
                  <TableHead className="w-32 text-right">
                    Jumlah siswa
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {byClass.map(([className, count]) => (
                  <TableRow key={className}>
                    <TableCell className="font-medium">
                      {className.toUpperCase()}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {count}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            Tidak ada kelas yang tercatat pada berkas ini.
          </p>
        )
      ) : null}

      {commitError ? (
        <Alert variant="destructive">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Gagal menjalankan impor</AlertTitle>
          <AlertDescription>{commitError}</AlertDescription>
        </Alert>
      ) : null}

      <Button
        type="button"
        onClick={() => setConfirmOpen(true)}
        disabled={commitMutation.isPending}
      >
        Impor sekarang
      </Button>

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Jalankan impor?</AlertDialogTitle>
            <AlertDialogDescription>
              {validRows} akun akan dibuat dengan sandi awal. Lanjutkan?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={commitMutation.isPending}>
              Batal
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={commitMutation.isPending}
              onClick={(event) => {
                event.preventDefault();
                commitMutation.mutate({ id: job.id });
              }}
            >
              {commitMutation.isPending ? "Memproses…" : "Lanjutkan"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function InvalidSection({ job }: { job: ImportJob }) {
  const [fieldFilter, setFieldFilter] = useState("all");
  const fields = useMemo(
    () =>
      Array.from(new Set(job.errors.map((error) => error.field))).sort(
        (a, b) => importFieldLabel(a).localeCompare(importFieldLabel(b), "id"),
      ),
    [job.errors],
  );
  const errors =
    fieldFilter === "all"
      ? job.errors
      : job.errors.filter((error) => error.field === fieldFilter);

  return (
    <div className="space-y-4">
      <Alert variant="destructive">
        <CircleAlert aria-hidden="true" />
        <AlertTitle>Ditemukan {job.errorCountTotal} kesalahan</AlertTitle>
        <AlertDescription>
          {job.errorCountTotal > job.errors.length ? (
            <p>
              Hanya {job.errors.length} kesalahan pertama yang ditampilkan.
            </p>
          ) : null}
          <p>Perbaiki file lalu unggah ulang.</p>
        </AlertDescription>
      </Alert>

      {fields.length > 0 ? (
        <div className="grid gap-1.5 sm:max-w-xs">
          <span className="text-sm font-medium">Filter kolom</span>
          <Select value={fieldFilter} onValueChange={setFieldFilter}>
            <SelectTrigger aria-label="Filter kolom kesalahan">
              <SelectValue placeholder="Semua kolom" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua kolom</SelectItem>
              {fields.map((field) => (
                <SelectItem key={field} value={field}>
                  {importFieldLabel(field)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      ) : null}

      <div className="max-h-96 overflow-auto rounded-xl border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-24">Baris</TableHead>
              <TableHead className="w-36">Kolom</TableHead>
              <TableHead>Pesan</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {errors.length === 0 ? (
              <TableRow>
                <TableCell colSpan={3} className="py-8 text-center">
                  <Inbox
                    aria-hidden="true"
                    className="mx-auto mb-2 size-5 text-muted-foreground"
                  />
                  <p className="text-sm text-muted-foreground">
                    Tidak ada kesalahan pada kolom ini.
                  </p>
                </TableCell>
              </TableRow>
            ) : (
              errors.map((error, index) => (
                <TableRow key={`${error.row}-${error.field}-${index}`}>
                  <TableCell className="tabular-nums">
                    {error.row === 0 ? "Berkas" : error.row}
                  </TableCell>
                  <TableCell>{importFieldLabel(error.field)}</TableCell>
                  <TableCell>{error.message}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

function CommittedSection({ job }: { job: ImportJob }) {
  const created = job.created ?? 0;

  return (
    <div className="space-y-4">
      <Alert className="border-emerald-600/30 bg-emerald-500/10 text-emerald-800 dark:border-emerald-400/30 dark:bg-emerald-400/10 dark:text-emerald-200">
        <CircleCheck aria-hidden="true" />
        <AlertTitle>{created} akun dibuat</AlertTitle>
        <AlertDescription className="text-emerald-700/90 dark:text-emerald-200/80">
          Akun baru memakai sandi awal dan wajib aktivasi saat pertama masuk.
        </AlertDescription>
      </Alert>

      {job.skipped.length > 0 ? (
        <div className="space-y-2">
          <h3 className="text-sm font-medium">
            {job.skipped.length} baris dilewati
          </h3>
          <div className="max-h-72 overflow-auto rounded-xl border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-24">Baris</TableHead>
                  <TableHead>Alasan</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {job.skipped.map((row) => (
                  <TableRow key={row.row}>
                    <TableCell className="tabular-nums">{row.row}</TableCell>
                    <TableCell>{row.reason}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      ) : null}

      {job.kind === "students" ? <AccountCardsSection job={job} /> : null}
    </div>
  );
}

function FailedSection() {
  return (
    <div className="space-y-4">
      <Alert variant="destructive">
        <CircleAlert aria-hidden="true" />
        <AlertTitle>Impor gagal diproses</AlertTitle>
        <AlertDescription>
          <p>
            Impor tidak dapat diselesaikan. Data yang sudah terlanjur dibuat
            tetap tersimpan.
          </p>
          <p>Periksa kembali berkas, lalu unggah ulang dan coba lagi.</p>
        </AlertDescription>
      </Alert>
      <p className="text-sm text-muted-foreground">
        Bila kegagalan berulang, hubungi administrator sistem.
      </p>
    </div>
  );
}

function ProgressSection({ label }: { label: string }) {
  return (
    <div className="space-y-3">
      <Progress
        value={100}
        aria-label={label}
        className="[&_[data-slot=progress-indicator]]:animate-pulse"
      />
      <p className="flex items-center gap-2 text-sm text-muted-foreground">
        <FileWarning aria-hidden="true" className="size-4" />
        {label}
      </p>
    </div>
  );
}

export function ImportJobDetail({ job }: { job: ImportJob }) {
  const title =
    job.status === "committed" ||
    job.status === "committing" ||
    job.status === "failed" ||
    job.status === "cancelled"
      ? "Hasil impor"
      : "Hasil pemeriksaan";

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>
          Impor {importKindLabel(job.kind)} · {formatDateTime(job.createdAt)}
        </CardDescription>
        <CardAction>
          <ImportStatusBadge job={job} />
        </CardAction>
      </CardHeader>
      <CardContent>
        {job.status === "validating" ? (
          <ProgressSection label="Sedang memeriksa berkas…" />
        ) : null}
        {job.status === "ready" ? <ReadySection job={job} /> : null}
        {job.status === "invalid" ? <InvalidSection job={job} /> : null}
        {job.status === "committing" ? (
          <ProgressSection label="Sedang membuat akun…" />
        ) : null}
        {job.status === "committed" ? <CommittedSection job={job} /> : null}
        {job.status === "failed" ? <FailedSection /> : null}
        {job.status === "cancelled" ? (
          <p className="text-sm text-muted-foreground">
            Impor ini dibatalkan sebelum dijalankan.
          </p>
        ) : null}
      </CardContent>
    </Card>
  );
}
