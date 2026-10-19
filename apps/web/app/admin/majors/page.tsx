"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { apiGet, apiSend } from "@/lib/api";

type Major = { id: string; code: string; name: string; require_series: boolean };

export default function MajorsPage() {
  const qc = useQueryClient();
  const [code, setCode] = useState("");
  const [name, setName] = useState("");

  const list = useQuery({
    queryKey: ["admin-majors"],
    queryFn: () => apiGet<Major[]>("v1/admin/majors"),
  });

  const create = useMutation({
    mutationFn: () => apiSend("v1/admin/majors", "POST", { code, name }),
    onSuccess: () => {
      toast.success("Jurusan dibuat.");
      setCode("");
      setName("");
      qc.invalidateQueries({ queryKey: ["admin-majors"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const items = Array.isArray(list.data) ? list.data : [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Jurusan</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="flex items-end gap-2">
          <div className="grid gap-1">
            <Label htmlFor="kode">Kode</Label>
            <Input id="kode" value={code} onChange={(e) => setCode(e.target.value)} placeholder="rpl" className="w-28" />
          </div>
          <div className="grid flex-1 gap-1">
            <Label htmlFor="nama">Nama</Label>
            <Input id="nama" value={name} onChange={(e) => setName(e.target.value)} placeholder="Rekayasa Perangkat Lunak" />
          </div>
          <Button onClick={() => create.mutate()} disabled={!code || !name || create.isPending}>
            Tambah
          </Button>
        </div>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Kode</TableHead>
              <TableHead>Nama</TableHead>
              <TableHead>Wajib series</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((m) => (
              <TableRow key={m.id}>
                <TableCell className="font-mono text-xs">{m.code}</TableCell>
                <TableCell>{m.name}</TableCell>
                <TableCell>{m.require_series ? "Ya" : "Tidak"}</TableCell>
              </TableRow>
            ))}
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={3} className="text-center text-sm text-muted-foreground">
                  {list.isLoading ? "Memuat..." : "Belum ada jurusan."}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
