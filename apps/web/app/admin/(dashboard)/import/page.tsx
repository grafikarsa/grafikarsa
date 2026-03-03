'use client';

import { useState, useCallback } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useDropzone } from 'react-dropzone';
import { toast } from 'sonner';
import {
  Upload,
  FileSpreadsheet,
  Download,
  AlertCircle,
  CheckCircle2,
  Loader2,
  X,
  FileText,
  Users,
  School,
  Info,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  adminImportApi,
  StudentImportDryRunResponse,
  StudentImportResponse,
  StudentImportError,
} from '@/lib/api/admin';
import { cn } from '@/lib/utils';

export default function ImportStudentsPage() {
  const [file, setFile] = useState<File | null>(null);
  const [dryRunResult, setDryRunResult] = useState<StudentImportDryRunResponse | null>(null);
  const [importResult, setImportResult] = useState<StudentImportResponse | null>(null);

  const onDrop = useCallback((acceptedFiles: File[]) => {
    if (acceptedFiles.length > 0) {
      setFile(acceptedFiles[0]);
      setDryRunResult(null);
      setImportResult(null);
    }
  }, []);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'text/csv': ['.csv'],
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': ['.xlsx'],
    },
    maxFiles: 1,
    maxSize: 5 * 1024 * 1024,
  });

  const dryRunMutation = useMutation({
    mutationFn: (file: File) => adminImportApi.dryRunImport(file),
    onSuccess: (response) => {
      setDryRunResult(response.data as StudentImportDryRunResponse);
      toast.success('Validasi selesai');
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Gagal memvalidasi file');
    },
  });

  const importMutation = useMutation({
    mutationFn: (file: File) => adminImportApi.importStudents(file, false),
    onSuccess: (response) => {
      setImportResult(response.data as StudentImportResponse);
      setDryRunResult(null);
      toast.success('Import selesai');
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Gagal mengimport data');
    },
  });

  const handleValidate = () => {
    if (file) {
      dryRunMutation.mutate(file);
    }
  };

  const handleImport = () => {
    if (file) {
      importMutation.mutate(file);
    }
  };

  const handleReset = () => {
    setFile(null);
    setDryRunResult(null);
    setImportResult(null);
  };

  const handleDownloadTemplate = () => {
    const csvContent = `tingkat,kode_jurusan,rombel,nama_lengkap,nis
10,rpl,A,Budi Santoso,25327004990001
10,rpl,A,Siti Aminah,25327004990002
11,dkv,B,Ahmad Rizki,24327004990001`;

    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'template_import_siswa.csv';
    a.click();
    window.URL.revokeObjectURL(url);
  };

  // Determine current step
  const currentStep = importResult ? 3 : dryRunResult ? 2 : file ? 1 : 0;

  return (
    <div className="space-y-6">
      {/* Step Indicator */}
      <div className="flex items-center justify-center gap-2">
        {[
          { label: 'Upload', icon: Upload },
          { label: 'Validasi', icon: FileText },
          { label: 'Import', icon: CheckCircle2 },
        ].map((step, index) => {
          const StepIcon = step.icon;
          const isActive = currentStep >= index;
          const isComplete = currentStep > index;
          return (
            <div key={step.label} className="flex items-center">
              {index > 0 && (
                <div
                  className={cn(
                    'h-0.5 w-8 sm:w-16',
                    isActive ? 'bg-primary' : 'bg-muted'
                  )}
                />
              )}
              <div
                className={cn(
                  'flex items-center gap-2 rounded-full px-3 py-1.5 text-sm font-medium transition-colors',
                  isComplete
                    ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                    : isActive
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted text-muted-foreground'
                )}
              >
                <StepIcon className="h-4 w-4" />
                <span className="hidden sm:inline">{step.label}</span>
              </div>
            </div>
          );
        })}
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Left Column - Instructions */}
        <div className="space-y-4 lg:col-span-1">
          {/* Format Info Card */}
          <Card className="p-0">
            <div className="border-b p-4">
              <div className="flex items-center gap-2">
                <div className="rounded-lg bg-blue-100 p-2 dark:bg-blue-900/30">
                  <Info className="h-4 w-4 text-blue-600" />
                </div>
                <h3 className="font-semibold">Format File</h3>
              </div>
            </div>
            <CardContent className="p-4 space-y-3">
              <p className="text-sm text-muted-foreground">
                File CSV/XLSX dengan 5 kolom:
              </p>
              <div className="space-y-2">
                {[
                  { col: 'tingkat', desc: '10, 11, atau 12' },
                  { col: 'kode_jurusan', desc: 'rpl, dkv, tkj, dll' },
                  { col: 'rombel', desc: 'A, B, C, dll' },
                  { col: 'nama_lengkap', desc: 'Nama siswa' },
                  { col: 'nis', desc: 'Nomor Induk Siswa' },
                ].map((item, i) => (
                  <div
                    key={item.col}
                    className="flex items-center gap-2 text-sm"
                  >
                    <Badge variant="outline" className="w-6 h-6 p-0 justify-center shrink-0">
                      {i + 1}
                    </Badge>
                    <code className="rounded bg-muted px-1.5 py-0.5 text-xs font-mono">
                      {item.col}
                    </code>
                    <span className="text-muted-foreground text-xs">{item.desc}</span>
                  </div>
                ))}
              </div>
              <Button
                variant="outline"
                size="sm"
                className="w-full mt-2"
                onClick={handleDownloadTemplate}
              >
                <Download className="mr-2 h-4 w-4" />
                Download Template
              </Button>
            </CardContent>
          </Card>

          {/* Credential Info */}
          <Alert className="border-amber-200 bg-amber-50 dark:border-amber-900 dark:bg-amber-950/20">
            <AlertCircle className="h-4 w-4 text-amber-600" />
            <AlertTitle className="text-amber-800 dark:text-amber-400">
              Kredensial Login
            </AlertTitle>
            <AlertDescription className="text-amber-700 dark:text-amber-300 text-sm">
              Siswa login dengan NIS sebagai username dan password.
            </AlertDescription>
          </Alert>
        </div>

        {/* Right Column - Main Content */}
        <div className="space-y-4 lg:col-span-2">
          {/* Upload Area */}
          {!importResult && (
            <Card className="p-0">
              <div className="border-b p-4">
                <div className="flex items-center gap-2">
                  <div className="rounded-lg bg-primary/10 p-2">
                    <Upload className="h-4 w-4 text-primary" />
                  </div>
                  <h3 className="font-semibold">Upload File</h3>
                </div>
              </div>
              <CardContent className="p-4">
                {!file ? (
                  <div
                    {...getRootProps()}
                    className={cn(
                      'flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 transition-colors cursor-pointer',
                      isDragActive
                        ? 'border-primary bg-primary/5'
                        : 'border-muted-foreground/25 hover:border-primary/50 hover:bg-muted/50'
                    )}
                  >
                    <input {...getInputProps()} />
                    <div className="rounded-full bg-muted p-4">
                      <FileSpreadsheet className="h-8 w-8 text-muted-foreground" />
                    </div>
                    <p className="mt-4 text-sm font-medium">
                      {isDragActive ? 'Drop file di sini...' : 'Drag & drop file di sini'}
                    </p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      CSV atau XLSX (max 5MB)
                    </p>
                    <Button variant="secondary" size="sm" className="mt-4">
                      Pilih File
                    </Button>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between rounded-lg border bg-muted/30 p-4">
                      <div className="flex items-center gap-3">
                        <div className="rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
                          <FileSpreadsheet className="h-5 w-5 text-green-600" />
                        </div>
                        <div>
                          <p className="font-medium">{file.name}</p>
                          <p className="text-xs text-muted-foreground">
                            {(file.size / 1024).toFixed(1)} KB
                          </p>
                        </div>
                      </div>
                      <Button variant="ghost" size="icon" onClick={handleReset}>
                        <X className="h-4 w-4" />
                      </Button>
                    </div>

                    {!dryRunResult && (
                      <Button
                        onClick={handleValidate}
                        disabled={dryRunMutation.isPending}
                        className="w-full"
                      >
                        {dryRunMutation.isPending && (
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        )}
                        <FileText className="mr-2 h-4 w-4" />
                        Validasi File
                      </Button>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Validation Result */}
          {dryRunResult && (
            <Card className="p-0">
              <div className="border-b p-4">
                <div className="flex items-center gap-2">
                  <div className="rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
                    <FileText className="h-4 w-4 text-amber-600" />
                  </div>
                  <h3 className="font-semibold">Hasil Validasi</h3>
                </div>
              </div>
              <CardContent className="p-4 space-y-4">
                {/* Stats Grid */}
                <div className="grid grid-cols-3 gap-3">
                  <div className="rounded-lg border p-3 text-center">
                    <div className="text-2xl font-bold">{dryRunResult.total_rows}</div>
                    <div className="text-xs text-muted-foreground">Total Baris</div>
                  </div>
                  <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-center dark:border-green-900 dark:bg-green-950/20">
                    <div className="text-2xl font-bold text-green-600">
                      {dryRunResult.students_to_create}
                    </div>
                    <div className="text-xs text-green-600">Siswa Valid</div>
                  </div>
                  <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-center dark:border-red-900 dark:bg-red-950/20">
                    <div className="text-2xl font-bold text-red-600">
                      {dryRunResult.validation_errors?.length ?? 0}
                    </div>
                    <div className="text-xs text-red-600">Error</div>
                  </div>
                </div>

                {/* New Classes Alert */}
                {dryRunResult.classes_to_create &&
                  dryRunResult.classes_to_create.length > 0 && (
                    <Alert className="border-blue-200 bg-blue-50 dark:border-blue-900 dark:bg-blue-950/20">
                      <School className="h-4 w-4 text-blue-600" />
                      <AlertTitle className="text-blue-800 dark:text-blue-400">
                        {dryRunResult.classes_to_create.length} Kelas Baru
                      </AlertTitle>
                      <AlertDescription>
                        <div className="mt-2 flex flex-wrap gap-1">
                          {dryRunResult.classes_to_create.map((cls, i) => (
                            <Badge
                              key={i}
                              variant="secondary"
                              className="bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300"
                            >
                              {cls.nama}
                            </Badge>
                          ))}
                        </div>
                      </AlertDescription>
                    </Alert>
                  )}

                {/* Errors */}
                {dryRunResult.validation_errors &&
                  dryRunResult.validation_errors.length > 0 && (
                    <div>
                      <h4 className="mb-2 text-sm font-medium text-red-600">
                        Error ({dryRunResult.validation_errors.length})
                      </h4>
                      <ScrollArea className="h-40 rounded-lg border border-red-200 bg-red-50/50 dark:border-red-900 dark:bg-red-950/10">
                        <div className="p-3 space-y-2">
                          {dryRunResult.validation_errors.map((err, i) => (
                            <ErrorRow key={i} error={err} />
                          ))}
                        </div>
                      </ScrollArea>
                    </div>
                  )}

                {/* Actions */}
                <div className="flex gap-2 pt-2">
                  <Button
                    onClick={handleImport}
                    disabled={
                      importMutation.isPending || dryRunResult.students_to_create === 0
                    }
                    className="flex-1"
                  >
                    {importMutation.isPending && (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    )}
                    <Users className="mr-2 h-4 w-4" />
                    Import {dryRunResult.students_to_create} Siswa
                  </Button>
                  <Button variant="outline" onClick={handleReset}>
                    Batal
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Import Result */}
          {importResult && (
            <Card className="p-0 border-green-200 dark:border-green-900">
              <div className="border-b border-green-200 bg-green-50 p-4 dark:border-green-900 dark:bg-green-950/20">
                <div className="flex items-center gap-2">
                  <div className="rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
                    <CheckCircle2 className="h-4 w-4 text-green-600" />
                  </div>
                  <h3 className="font-semibold text-green-800 dark:text-green-400">
                    Import Berhasil!
                  </h3>
                </div>
              </div>
              <CardContent className="p-4 space-y-4">
                {/* Stats Grid */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                  <div className="rounded-lg border p-3 text-center">
                    <div className="text-xl font-bold">{importResult.total_rows}</div>
                    <div className="text-xs text-muted-foreground">Total Baris</div>
                  </div>
                  <div className="rounded-lg border border-blue-200 bg-blue-50 p-3 text-center dark:border-blue-900 dark:bg-blue-950/20">
                    <div className="text-xl font-bold text-blue-600">
                      {importResult.created_classes}
                    </div>
                    <div className="text-xs text-blue-600">Kelas Dibuat</div>
                  </div>
                  <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-center dark:border-green-900 dark:bg-green-950/20">
                    <div className="text-xl font-bold text-green-600">
                      {importResult.created_students}
                    </div>
                    <div className="text-xs text-green-600">Siswa Dibuat</div>
                  </div>
                  <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-center dark:border-amber-900 dark:bg-amber-950/20">
                    <div className="text-xl font-bold text-amber-600">
                      {importResult.skipped}
                    </div>
                    <div className="text-xs text-amber-600">Dilewati</div>
                  </div>
                </div>

                {/* Errors */}
                {importResult.errors && importResult.errors.length > 0 && (
                  <div>
                    <h4 className="mb-2 text-sm font-medium text-red-600">
                      Error ({importResult.errors.length})
                    </h4>
                    <ScrollArea className="h-32 rounded-lg border border-red-200 bg-red-50/50 dark:border-red-900 dark:bg-red-950/10">
                      <div className="p-3 space-y-2">
                        {importResult.errors.map((err, i) => (
                          <ErrorRow key={i} error={err} />
                        ))}
                      </div>
                    </ScrollArea>
                  </div>
                )}

                {/* Credential Reminder */}
                <Alert className="border-green-200 bg-green-50 dark:border-green-900 dark:bg-green-950/20">
                  <CheckCircle2 className="h-4 w-4 text-green-600" />
                  <AlertTitle className="text-green-800 dark:text-green-400">
                    Kredensial Siswa
                  </AlertTitle>
                  <AlertDescription className="text-green-700 dark:text-green-300 text-sm">
                    Siswa dapat login dengan NIS sebagai username dan password.
                  </AlertDescription>
                </Alert>

                <Button onClick={handleReset} className="w-full">
                  <Upload className="mr-2 h-4 w-4" />
                  Import Lagi
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}

function ErrorRow({ error }: { error: StudentImportError }) {
  return (
    <div className="flex items-start gap-2 rounded-md bg-white p-2 text-sm dark:bg-red-950/30">
      <AlertCircle className="h-4 w-4 text-red-500 mt-0.5 shrink-0" />
      <div className="min-w-0">
        <span className="font-medium">Baris {error.row}</span>
        {error.nis && (
          <span className="text-muted-foreground"> (NIS: {error.nis})</span>
        )}
        {error.nama && (
          <span className="text-muted-foreground"> - {error.nama}</span>
        )}
        <span className="text-red-600 block sm:inline">: {error.error}</span>
      </div>
    </div>
  );
}
