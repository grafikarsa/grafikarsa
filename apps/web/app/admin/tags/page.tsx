"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
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
import { apiGet, apiSend } from "@/lib/api";

type Tag = { id: string; slug: string; name: string };

export default function TagsPage() {
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [editID, setEditID] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editSlug, setEditSlug] = useState("");

  const list = useQuery({
    queryKey: ["admin-tags"],
    queryFn: () => apiGet<Tag[]>("v1/admin/tags"),
  });

  const create = useMutation({
    mutationFn: () => apiSend("v1/admin/tags", "POST", { name, slug }),
    onSuccess: () => {
      toast.success("Tag dibuat.");
      setName("");
      setSlug("");
      qc.invalidateQueries({ queryKey: ["admin-tags"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const update = useMutation({
    mutationFn: (id: string) =>
      apiSend(`v1/admin/tags/${id}`, "PATCH", { name: editName, slug: editSlug }),
    onSuccess: () => {
      toast.success("Tag diperbarui.");
      setEditID(null);
      qc.invalidateQueries({ queryKey: ["admin-tags"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const remove = useMutation({
    mutationFn: (id: string) => apiSend(`v1/admin/tags/${id}`, "DELETE"),
    onSuccess: () => {
      toast.success("Tag dihapus.");
      qc.invalidateQueries({ queryKey: ["admin-tags"] });
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : "Gagal menghapus (mungkin masih dipakai)."),
  });

  const items = Array.isArray(list.data) ? list.data : [];

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Tags</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="grid gap-2 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
            <div className="grid gap-1">
              <Label htmlFor="tname">Nama</Label>
              <Input id="tname" value={name} onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="grid gap-1">
              <Label htmlFor="tslug">Slug</Label>
              <Input id="tslug" value={slug} onChange={(e) => setSlug(e.target.value)} />
            </div>
            <Button onClick={() => create.mutate()} disabled={!name || !slug || create.isPending}>
              Tambah
            </Button>
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama</TableHead>
                <TableHead>Slug</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((t) =>
                editID === t.id ? (
                  <TableRow key={t.id}>
                    <TableCell>
                      <Input
                        value={editName}
                        onChange={(e) => setEditName(e.target.value)}
                        aria-label="Nama tag"
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        value={editSlug}
                        onChange={(e) => setEditSlug(e.target.value)}
                        aria-label="Slug tag"
                      />
                    </TableCell>
                    <TableCell className="flex justify-end gap-2">
                      <Button
                        size="sm"
                        onClick={() => update.mutate(t.id)}
                        disabled={update.isPending}
                      >
                        Simpan
                      </Button>
                      <Button size="sm" variant="ghost" onClick={() => setEditID(null)}>
                        Batal
                      </Button>
                    </TableCell>
                  </TableRow>
                ) : (
                  <TableRow key={t.id}>
                    <TableCell>{t.name}</TableCell>
                    <TableCell className="font-mono text-xs">{t.slug}</TableCell>
                    <TableCell className="flex justify-end gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setEditID(t.id);
                          setEditName(t.name);
                          setEditSlug(t.slug);
                        }}
                      >
                        Edit
                      </Button>
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => {
                          if (
                            confirm(
                              `Hapus tag "${t.name}"? Tag yang masih dipakai portfolio akan ditolak.`,
                            )
                          ) {
                            remove.mutate(t.id);
                          }
                        }}
                      >
                        Hapus
                      </Button>
                    </TableCell>
                  </TableRow>
                ),
              )}
              {items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-sm text-muted-foreground">
                    {list.isLoading ? "Memuat..." : "Belum ada tag."}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
