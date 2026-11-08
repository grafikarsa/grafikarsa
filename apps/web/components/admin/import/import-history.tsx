"use client";

import { History } from "lucide-react";

import {
  importKindLabel,
  IMPORT_STATUS_LABELS,
  IMPORT_STATUS_VARIANTS,
} from "@/components/admin/import/labels";
import { formatDateTime } from "@/components/admin/users/labels";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { ImportJob } from "@/lib/api/gen/models";
import { cn } from "@/lib/utils";

function rowCount(job: ImportJob): string {
  if (job.totalRows === null) {
    return "—";
  }

  return `${job.validRows ?? 0}/${job.totalRows}`;
}

export function ImportHistory({
  jobs,
  selectedJobId,
  onSelect,
  isLoading = false,
}: {
  jobs: ImportJob[];
  selectedJobId: string | null;
  onSelect: (id: string) => void;
  isLoading?: boolean;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Riwayat impor</CardTitle>
        <CardDescription>
          Pilih riwayat untuk melihat hasil pemeriksaan atau impor sebelumnya.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-hidden rounded-xl border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Waktu</TableHead>
                <TableHead className="w-24">Jenis</TableHead>
                <TableHead className="w-36">Status</TableHead>
                <TableHead className="w-28 text-right">Baris valid</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 3 }).map((_, index) => (
                  <TableRow key={index}>
                    <TableCell>
                      <Skeleton className="h-4 w-36" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-4 w-12" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="h-5 w-24 rounded-4xl" />
                    </TableCell>
                    <TableCell>
                      <Skeleton className="ml-auto h-4 w-10" />
                    </TableCell>
                  </TableRow>
                ))
              ) : jobs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="py-10 text-center">
                    <History
                      aria-hidden="true"
                      className="mx-auto mb-2 size-5 text-muted-foreground"
                    />
                    <p className="font-medium">Belum ada riwayat impor.</p>
                    <p className="mt-1 text-sm text-muted-foreground">
                      Unggah berkas CSV untuk memulai.
                    </p>
                  </TableCell>
                </TableRow>
              ) : (
                jobs.map((job) => (
                  <TableRow
                    key={job.id}
                    className={cn(
                      "cursor-pointer",
                      selectedJobId === job.id && "bg-muted/60",
                    )}
                    tabIndex={0}
                    aria-selected={selectedJobId === job.id}
                    onClick={() => onSelect(job.id)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        onSelect(job.id);
                      }
                    }}
                  >
                    <TableCell>{formatDateTime(job.createdAt)}</TableCell>
                    <TableCell>{importKindLabel(job.kind)}</TableCell>
                    <TableCell>
                      <Badge variant={IMPORT_STATUS_VARIANTS[job.status]}>
                        {IMPORT_STATUS_LABELS[job.status]}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {rowCount(job)}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}
