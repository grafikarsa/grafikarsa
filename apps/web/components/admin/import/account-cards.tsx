"use client";

import { useMemo, useState } from "react";
import { CircleAlert, Download } from "lucide-react";

import { apiErrorMessage } from "@/components/admin/form-error";
import { normalizeClassName } from "@/components/admin/import/labels";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  getAdminAccountCards,
  useGetAdminAcademicYears,
  useGetAdminClasses,
} from "@/lib/api/gen/endpoints";
import type { ImportJob } from "@/lib/api/gen/models";

export function AccountCardsSection({ job }: { job: ImportJob }) {
  const yearsQuery = useGetAdminAcademicYears();
  const currentYear = (yearsQuery.data?.items ?? []).find(
    (year) => year.isCurrent,
  );
  const classesQuery = useGetAdminClasses(
    { academicYearId: currentYear?.id },
    { query: { enabled: Boolean(currentYear?.id) } },
  );
  const [downloading, setDownloading] = useState<string | null>(null);
  const [downloadError, setDownloadError] = useState<string | null>(null);

  const classesByName = useMemo(() => {
    const map = new Map<string, { id: string; name: string; slug: string }>();

    for (const schoolClass of classesQuery.data?.items ?? []) {
      map.set(normalizeClassName(schoolClass.name), {
        id: schoolClass.id,
        name: schoolClass.name,
        slug: schoolClass.slug,
      });
    }

    return map;
  }, [classesQuery.data]);

  const byClass = Object.entries(job.byClass).sort(([a], [b]) =>
    a.localeCompare(b, "id"),
  );

  if (byClass.length === 0) {
    return null;
  }

  const download = async (
    classId: string,
    slug: string,
    className: string,
  ) => {
    setDownloadError(null);
    setDownloading(classId);

    try {
      const blob = await getAdminAccountCards({ classId });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");

      link.href = url;
      link.download = `kartu-akun-${slug}.pdf`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) {
      setDownloadError(
        apiErrorMessage(
          error,
          `Kartu akun kelas ${className} tidak dapat diunduh.`,
        ),
      );
    } finally {
      setDownloading(null);
    }
  };

  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <h3 className="text-sm font-medium">Kartu akun</h3>
        <p className="text-sm text-muted-foreground">
          Unduh kartu akun berisi nama, NISN, dan sandi awal siswa yang belum
          aktivasi.
        </p>
      </div>

      {yearsQuery.isError || classesQuery.isError ? (
        <Alert variant="destructive">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Kelas tidak dapat dimuat</AlertTitle>
          <AlertDescription>
            Kartu akun belum bisa diunduh karena daftar kelas gagal dimuat.
          </AlertDescription>
        </Alert>
      ) : null}

      {downloadError ? (
        <Alert variant="destructive">
          <CircleAlert aria-hidden="true" />
          <AlertTitle>Unduhan gagal</AlertTitle>
          <AlertDescription>{downloadError}</AlertDescription>
        </Alert>
      ) : null}

      <div className="flex flex-wrap gap-2">
        {byClass.map(([className]) => {
          const schoolClass = classesByName.get(normalizeClassName(className));

          if (!schoolClass) {
            return (
              <p key={className} className="text-sm text-muted-foreground">
                Kelas {className.toUpperCase()} tidak ditemukan pada tahun
                ajaran berjalan.
              </p>
            );
          }

          return (
            <Button
              key={className}
              type="button"
              variant="outline"
              onClick={() => {
                void download(
                  schoolClass.id,
                  schoolClass.slug,
                  schoolClass.name,
                );
              }}
              disabled={downloading !== null}
            >
              <Download aria-hidden="true" />
              {downloading === schoolClass.id
                ? "Mengunduh…"
                : `Unduh kartu akun ${schoolClass.name}`}
            </Button>
          );
        })}
      </div>
    </div>
  );
}
