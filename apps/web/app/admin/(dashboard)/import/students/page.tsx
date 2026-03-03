'use client';

import { useState, useCallback } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Upload, FileSpreadsheet, AlertCircle, CheckCircle2, Loader2, Download } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  adminImportApi,
  StudentImportDryRunResponse,
  StudentImportResponse,
  StudentImportError,
} from '@/lib/api/admin';

export default function ImportStudentsPage() {
  const [file, setFile] = useState<File | null>(null);
  const [dragActive, setDragActive] = useState(false);
  const [dryRunResult, setDryRunResult] = useState<StudentImportDryRunResponse | null>(null);
  const [importResult, setImportResult] = useState<StudentImportResponse | null>(null);

  // Dry run mutation
  const dryRunMutation = useMutation({
    mutationFn: (file: File) => adminImportApi.dryRunImport(file),
    onSuccess: (response) => {
      if (response.data) {
        setDryRunResult(response.data);
        setImportResult(null);
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Gagal memvalidasi file');
    },
  });

  // Import mutation
  const importMutation = useMutation({
    mutationFn: (file: File) => adminImportApi.importStudents(file, false),
    onSuccess: (response) => {
      if (response.data) {
        setImportResult(response.data as StudentImportResponse);
        setDryRunResult(null);
        setFile(null);
        toast.success(response.message || 'Import berhasil');
      }
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Gagal mengimport data');
    },
  });

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      const droppedFile = e.dataTransfer.files[0];
      if (isValidFile(droppedFile)) {
        setFile(droppedFile);
        setDryRunResult(null);
        setImportResult(null);
      } else {
        toast.error('File harus berformat CSV atau XLSX');
      }
    }
  }, []);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const selectedFile = e.target.files[0];
      if (isValidFile(selectedFile)) {
        setFile(selectedFile);
        setDryRunResult(null);
        setImportResult(null);
      } else {
        toast.error('File harus berformat CSV atau XLSX');
      }
    }
  };

  const isValidFile = (file: File) => {
    const validTypes = [
      'text/csv',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      'application/vnd.ms-excel',
    ];
    const validExtensions = ['.csv', '.xlsx', '.xls'];
    const hasValidType = validTypes.includes(file.type);
    const hasValidExtension = validExtensions.some((ext) => file.name.toLowerCase().endsWith(ext));
    return hasValidType || hasValidExtension;
  };

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

  const isLoading = dryRunMutation.isPending || importMutation.isPending;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Import Data Siswa</h1>
        <p className="text-muted-foreground">Upload file CSV atau XLSX untuk import data siswa secara massal</p>
      </div>

      {/* Format Info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Format File</CardTitle>
          <CardDescription>File harus memiliki 5 kolom dengan urutan berikut:</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-20">Kolom</TableHead>
                  <TableHead>Field</TableHead>
                  <TableHead>Contoh</TableHead>
                  <TableHead>Keterangan</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow>
                  <TableCell>1</TableCell>
                  <TableCell>Tingkat</TableCell>
                  <TableCell><code>10</code>, <code>11</code>, <code>12</code></TableCell>
                  <TableCell>Tingkatan kelas</TableCell>
                </TableRow>
                <TableRow>
                  <TableCell>2</TableCell>
                  <TableCell>Kode Jurusan</TableCell>
                  <TableCell><code>rpl</code>, <code>dkv</code>, <code>tkj</code></TableCell>
                  <TableCell>Harus sesuai master jurusan</TableCell>
                </TableRow>
                <TableRow>
                  <TableCell>3</TableCell>
                  <TableCell>Rombel</TableCell>
                  <TableCell><code>A</code>, <code>B</code>, <code>C</code></TableCell>
                  <TableCell>Rombongan belajar (A-Z)</TableCell>
                </TableRow>
                <TableRow>
                  <TableCell>4</TableCell>
                  <TableCell>Nama Lengkap</TableCell>
                  <TableCell><code>Budi Santoso</code></TableCell>
                  <TableCell>Nama lengkap siswa</TableCell>
                </TableRow>
                <TableRow>
                  <TableCell>5</TableCell>
                  <TableCell>NIS</TableCell>
                  <TableCell><code>25327004990242</code></TableCell>
                  <TableCell>Nomor Induk Siswa (angka)</TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
          <p className="mt-4 text-sm text-muted-foreground">
            <strong>Catatan:</strong> Username = NIS, Email = NIS@grafikarsa.com, Password = NIS
          </p>
        </CardContent>
      </Card>

      {/* Upload Area */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Upload File</CardTitle>
        </CardHeader>
        <CardContent>
          <div
            className={`relative border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
              dragActive ? 'border-primary bg-primary/5' : 'border-muted-foreground/25 hover:border-primary/50'
            }`}
            onDragEnter={handleDrag}
            onDragLeave={handleDrag}
            onDragOver={handleDrag}
            onDrop={handleDrop}
          >
            <input
              type="file"
              accept=".csv,.xlsx,.xls"
              onChange={handleFileChange}
              className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
              disabled={isLoading}
            />
            <div className="flex flex-col items-center gap-2">
              {file ? (
                <>
                  <FileSpreadsheet className="h-10 w-10 text-primary" />
                  <p className="font-medium">{file.name}</p>
                  <p className="text-sm text-muted-foreground">{(file.size / 1024).toFixed(1)} KB</p>
                </>
              ) : (
                <>
                  <Upload className="h-10 w-10 text-muted-foreground" />
                  <p className="font-medium">Drag & drop file atau klik untuk memilih</p>
                  <p className="text-sm text-muted-foreground">Format: CSV, XLSX (max 5MB)</p>
                </>
              )}
            </div>
          </div>

          {file && (
            <div className="mt-4 flex gap-2">
              <Button onClick={handleValidate} disabled={isLoading} variant="outline">
                {dryRunMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Memvalidasi...
                  </>
                ) : (
                  'Validasi'
                )}
              </Button>
              <Button onClick={handleImport} disabled={isLoading || !dryRunResult}>
                {importMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Mengimport...
                  </>
                ) : (
                  'Import'
                )}
              </Button>
              <Button variant="ghost" onClick={handleReset} disabled={isLoading}>
                Reset
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Dry Run Result */}
      {dryRunResult && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <AlertCircle className="h-5 w-5 text-amber-500" />
              Hasil Validasi (Preview)
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="text-center p-3 bg-muted rounded-lg">
                <p className="text-2xl font-bold">{dryRunResult.total_rows}</p>
                <p className="text-sm text-muted-foreground">Total Baris</p>
              </div>
              <div className="text-center p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <p className="text-2xl font-bold text-green-600">{dryRunResult.students_to_create}</p>
                <p className="text-sm text-muted-foreground">Siswa Valid</p>
              </div>
              <div className="text-center p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <p className="text-2xl font-bold text-blue-600">{dryRunResult.classes_to_create?.length ?? 0}</p>
                <p className="text-sm text-muted-foreground">Kelas Baru</p>
              </div>
              <div className="text-center p-3 bg-red-50 dark:bg-red-900/20 rounded-lg">
                <p className="text-2xl font-bold text-red-600">{dryRunResult.validation_errors?.length ?? 0}</p>
                <p className="text-sm text-muted-foreground">Error</p>
              </div>
            </div>

            {(dryRunResult.classes_to_create?.length ?? 0) > 0 && (
              <>
                <Separator />
                <div>
                  <h4 className="font-medium mb-2">Kelas yang akan dibuat:</h4>
                  <div className="flex flex-wrap gap-2">
                    {dryRunResult.classes_to_create?.map((kelas, i) => (
                      <Badge key={i} variant="secondary">
                        {kelas.nama}
                      </Badge>
                    ))}
                  </div>
                </div>
              </>
            )}

            {(dryRunResult.validation_errors?.length ?? 0) > 0 && (
              <>
                <Separator />
                <div>
                  <h4 className="font-medium mb-2 text-red-600">Error ({dryRunResult.validation_errors?.length ?? 0}):</h4>
                  <div className="max-h-60 overflow-y-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-16">Baris</TableHead>
                          <TableHead>NIS</TableHead>
                          <TableHead>Nama</TableHead>
                          <TableHead>Error</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {dryRunResult.validation_errors?.map((err, i) => (
                          <TableRow key={i}>
                            <TableCell>{err.row}</TableCell>
                            <TableCell>{err.nis || '-'}</TableCell>
                            <TableCell>{err.nama || '-'}</TableCell>
                            <TableCell className="text-red-600">{err.error}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      )}

      {/* Import Result */}
      {importResult && (
        <Alert className="border-green-500 bg-green-50 dark:bg-green-900/20">
          <CheckCircle2 className="h-5 w-5 text-green-600" />
          <AlertTitle className="text-green-600">Import Berhasil!</AlertTitle>
          <AlertDescription>
            <div className="mt-2 grid grid-cols-2 md:grid-cols-4 gap-4">
              <div>
                <p className="font-bold">{importResult.total_rows}</p>
                <p className="text-sm">Total Baris</p>
              </div>
              <div>
                <p className="font-bold text-green-600">{importResult.created_students}</p>
                <p className="text-sm">Siswa Dibuat</p>
              </div>
              <div>
                <p className="font-bold text-blue-600">{importResult.created_classes}</p>
                <p className="text-sm">Kelas Dibuat</p>
              </div>
              <div>
                <p className="font-bold text-amber-600">{importResult.skipped}</p>
                <p className="text-sm">Dilewati</p>
              </div>
            </div>

            {(importResult.errors?.length ?? 0) > 0 && (
              <div className="mt-4">
                <p className="font-medium text-amber-600 mb-2">Baris yang dilewati:</p>
                <div className="max-h-40 overflow-y-auto text-sm">
                  {importResult.errors?.map((err, i) => (
                    <p key={i}>
                      Baris {err.row}: {err.error}
                    </p>
                  ))}
                </div>
              </div>
            )}
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}
