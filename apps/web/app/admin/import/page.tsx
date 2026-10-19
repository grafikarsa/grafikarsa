"use client";

import { useMutation } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { apiUpload } from "@/lib/api";

type Row = { username: string; password: string };
type Result = { created: number; errors: string[]; dry_run: boolean; accounts: Row[] };

// CSV: username,email,name,role  (header baris pertama)
export default function ImportPage() {
  const fileRef = useRef<HTMLInputElement | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [result, setResult] = useState<Result | null>(null);

  const run = useMutation({
    mutationFn: (dryRun: boolean) => {
      if (!file) throw new Error("Pilih file CSV dulu.");
      const form = new FormData();
      form.append("file", file);
      return apiUpload<Result>(
        `v1/admin/import/students?dry_run=${dryRun ? "true" : "false"}`,
        form,
      );
    },
    onSuccess: (data, dryRun) => {
      setResult(data);
      toast.success(dryRun ? "Dry-run selesai." : `Import selesai: ${data.created} akun.`);
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal import."),
  });

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Import siswa/guru (CSV)</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="grid gap-1">
            <Label htmlFor="csv">File CSV</Label>
            <Input
              id="csv"
              ref={fileRef}
              type="file"
              accept=".csv,text/csv"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
            <p className="text-muted-foreground text-xs">
              Format: <span className="font-mono">username,email,name,role</span> — role siswa/guru.
              Password dibuat acak dan ditampilkan sekali; user wajib menggantinya saat login
              pertama.
            </p>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => run.mutate(true)}
              disabled={!file || run.isPending}
            >
              Dry-run
            </Button>
            <Button onClick={() => run.mutate(false)} disabled={!file || run.isPending}>
              Import sekarang
            </Button>
          </div>
        </CardContent>
      </Card>

      {result && (
        <Card>
          <CardHeader>
            <CardTitle>
              Hasil {result.dry_run ? "(dry-run)" : ""} — {result.created} akun
            </CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {result.errors.length > 0 && (
              <div className="rounded border border-amber-500/40 bg-amber-500/5 p-2 text-xs">
                {result.errors.map((e, i) => (
                  <p key={i}>{e}</p>
                ))}
              </div>
            )}
            {(result.accounts ?? []).length > 0 && (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Username</TableHead>
                    <TableHead>Password awal</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {result.accounts.map((a) => (
                    <TableRow key={a.username}>
                      <TableCell className="font-mono text-xs">{a.username}</TableCell>
                      <TableCell className="font-mono text-xs">{a.password}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
            <p className="text-muted-foreground text-xs">
              Simpan daftar ini sekarang — password tidak bisa ditampilkan lagi.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
