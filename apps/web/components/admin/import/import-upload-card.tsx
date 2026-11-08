"use client";

import { useRef, useState } from "react";
import { CircleAlert, Download, FileUp, UploadCloud } from "lucide-react";

import { apiErrorMessage } from "@/components/admin/form-error";
import {
  formatFileSize,
  IMPORT_FORMAT_HELP,
  IMPORT_KIND_LABELS,
  IMPORT_TEMPLATES,
  MAX_IMPORT_FILE_BYTES,
} from "@/components/admin/import/labels";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { usePostAdminImport } from "@/lib/api/gen/endpoints";
import type { ImportJob, ImportKind } from "@/lib/api/gen/models";
import { cn } from "@/lib/utils";

const ALL_KINDS: ImportKind[] = ["students", "teachers", "staff"];

export function ImportUploadCard({
  canImportTeachers,
  onUploaded,
}: {
  canImportTeachers: boolean;
  onUploaded: (job: ImportJob) => void;
}) {
  const kinds = canImportTeachers ? ALL_KINDS : ["students" as const];
  const [kind, setKind] = useState<ImportKind>("students");
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [dragging, setDragging] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const uploadMutation = usePostAdminImport({
    mutation: {
      onSuccess: (job) => {
        setFile(null);
        setFileError(null);
        setDragging(false);

        if (inputRef.current) {
          inputRef.current.value = "";
        }

        onUploaded(job);
      },
    },
  });

  const acceptFile = (candidate: File | undefined) => {
    setFileError(null);

    if (!candidate) {
      setFile(null);
      return;
    }

    if (!candidate.name.toLowerCase().endsWith(".csv")) {
      setFile(null);
      setFileError("Berkas harus berformat .csv.");
      return;
    }

    if (candidate.size > MAX_IMPORT_FILE_BYTES) {
      setFile(null);
      setFileError(
        "Ukuran berkas melebihi 2 MB. Pecah berkas menjadi beberapa bagian lalu unggah ulang.",
      );
      return;
    }

    setFile(candidate);
  };

  const resetInput = () => {
    if (inputRef.current) {
      inputRef.current.value = "";
    }
  };

  const selectKind = (value: string) => {
    if (!kinds.includes(value as ImportKind)) {
      return;
    }

    setKind(value as ImportKind);
    setFile(null);
    setFileError(null);
    resetInput();
  };

  const uploadError = uploadMutation.isError
    ? apiErrorMessage(
        uploadMutation.error,
        "Berkas tidak dapat diunggah. Coba lagi.",
      )
    : null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Unggah berkas CSV</CardTitle>
        <CardDescription>
          Pilih jenis data, lalu unggah berkas untuk diperiksa sebelum impor.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-1.5 sm:max-w-xs">
          <Label htmlFor="import-kind">Jenis data</Label>
          <Select value={kind} onValueChange={selectKind}>
            <SelectTrigger id="import-kind" aria-label="Jenis data">
              <SelectValue placeholder="Pilih jenis data" />
            </SelectTrigger>
            <SelectContent>
              {kinds.map((value) => (
                <SelectItem key={value} value={value}>
                  {IMPORT_KIND_LABELS[value]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <label
          htmlFor="import-file"
          onDragOver={(event) => {
            event.preventDefault();
            setDragging(true);
          }}
          onDragLeave={(event) => {
            event.preventDefault();
            setDragging(false);
          }}
          onDrop={(event) => {
            event.preventDefault();
            setDragging(false);
            acceptFile(event.dataTransfer.files[0]);
          }}
          className={cn(
            "flex cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border border-dashed px-6 py-8 text-center transition-colors",
            dragging
              ? "border-primary bg-primary/5"
              : "border-border hover:border-primary/60 hover:bg-muted/50",
          )}
        >
          <UploadCloud
            aria-hidden="true"
            className="size-8 text-muted-foreground"
          />
          {file ? (
            <>
              <span className="font-medium break-all">{file.name}</span>
              <span className="text-xs text-muted-foreground">
                {formatFileSize(file.size)} · klik untuk mengganti berkas
              </span>
            </>
          ) : (
            <>
              <span className="font-medium">
                Seret berkas CSV ke sini atau klik untuk memilih
              </span>
              <span className="text-xs text-muted-foreground">
                Maksimal 2 MB dan 5.000 baris data.
              </span>
            </>
          )}
          <input
            ref={inputRef}
            id="import-file"
            type="file"
            accept=".csv,text/csv"
            className="sr-only"
            onChange={(event) => acceptFile(event.target.files?.[0])}
          />
        </label>

        {fileError ? (
          <Alert variant="destructive">
            <CircleAlert aria-hidden="true" />
            <AlertTitle>Berkas tidak dapat dipakai</AlertTitle>
            <AlertDescription>{fileError}</AlertDescription>
          </Alert>
        ) : null}

        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-muted/30 px-3 py-2.5">
          <div className="space-y-1 text-sm">
            <p className="font-medium">
              Format impor {IMPORT_KIND_LABELS[kind].toLowerCase()}
            </p>
            <ul className="list-inside list-disc space-y-0.5 text-xs text-muted-foreground">
              {IMPORT_FORMAT_HELP[kind].map((line) => (
                <li key={line}>{line}</li>
              ))}
              <li>Pemisah koma atau titik koma, berkas UTF-8.</li>
            </ul>
          </div>
          <Button variant="outline" size="sm" asChild>
            <a href={IMPORT_TEMPLATES[kind]} download>
              <Download aria-hidden="true" />
              Unduh template
            </a>
          </Button>
        </div>

        {uploadError ? (
          <Alert variant="destructive">
            <CircleAlert aria-hidden="true" />
            <AlertTitle>Berkas gagal diunggah</AlertTitle>
            <AlertDescription>{uploadError}</AlertDescription>
          </Alert>
        ) : null}

        <Button
          type="button"
          onClick={() => {
            if (!file || uploadMutation.isPending) {
              return;
            }

            uploadMutation.mutate({ data: { kind, file } });
          }}
          disabled={!file || uploadMutation.isPending}
        >
          <FileUp aria-hidden="true" />
          {uploadMutation.isPending ? "Memeriksa berkas…" : "Unggah & periksa"}
        </Button>
      </CardContent>
    </Card>
  );
}
