"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { apiGet, apiSend } from "@/lib/api";

type Year = { id: string; start_year: number; is_active: boolean };

export default function YearsPage() {
  const qc = useQueryClient();
  const [year, setYear] = useState("");

  const list = useQuery({
    queryKey: ["admin-years"],
    queryFn: () => apiGet<Year[]>("v1/admin/years"),
  });

  const create = useMutation({
    mutationFn: () => apiSend("v1/admin/years", "POST", { start_year: Number(year) }),
    onSuccess: () => {
      toast.success("Tahun ajaran dibuat.");
      setYear("");
      qc.invalidateQueries({ queryKey: ["admin-years"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const activate = useMutation({
    mutationFn: (id: string) => apiSend(`v1/admin/years/${id}/activate`, "POST"),
    onSuccess: () => {
      toast.success("Tahun aktif diperbarui.");
      qc.invalidateQueries({ queryKey: ["admin-years"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const items = Array.isArray(list.data) ? list.data : [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Tahun Ajaran</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="flex items-end gap-2">
          <div className="grid gap-1">
            <Input value={year} onChange={(e) => setYear(e.target.value)} placeholder="2027" className="w-28" />
          </div>
          <Button onClick={() => create.mutate()} disabled={!year || create.isPending}>
            Tambah
          </Button>
          <p className="text-muted-foreground text-xs">
            Hanya satu tahun yang aktif; mengaktifkan tahun lain otomatis menonaktifkan yang lama.
          </p>
        </div>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Tahun</TableHead>
              <TableHead>Status</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((y) => (
              <TableRow key={y.id}>
                <TableCell>{y.start_year}</TableCell>
                <TableCell>{y.is_active ? <Badge>Aktif</Badge> : <span className="text-muted-foreground text-xs">nonaktif</span>}</TableCell>
                <TableCell className="text-right">
                  {!y.is_active && (
                    <Button variant="outline" size="sm" onClick={() => activate.mutate(y.id)}>
                      Aktifkan
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={3} className="text-center text-sm text-muted-foreground">
                  {list.isLoading ? "Memuat..." : "Belum ada tahun ajaran."}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
