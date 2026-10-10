"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select-field";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { apiGet, apiSend } from "@/lib/api";

type Item = { id: string; version: string; title: string; status: string; reads?: number };

const KINDS = ["added", "improved", "fixed", "removed"];

export default function AdminChangelogsPage() {
  const qc = useQueryClient();
  const [version, setVersion] = useState("");
  const [title, setTitle] = useState("");
  const [summary, setSummary] = useState("");
  const [publishAt, setPublishAt] = useState("");
  const [sections, setSections] = useState<{ kind: string; title: string; body_md: string }[]>([
    { kind: "added", title: "", body_md: "" },
  ]);

  const list = useQuery({
    queryKey: ["admin-changelogs"],
    queryFn: () => apiGet<Item[]>("v1/admin/changelogs"),
  });

  const create = useMutation({
    mutationFn: () =>
      apiSend("v1/admin/changelogs", "POST", {
        version,
        title,
        summary,
        channel: "stable",
        audience: "all",
        publish_at: publishAt || undefined,
        sections: sections.filter((s) => s.title),
      }),
    onSuccess: () => {
      toast.success(publishAt ? "Changelog dijadwalkan." : "Changelog dibuat (draft).");
      setVersion("");
      setTitle("");
      setSummary("");
      setPublishAt("");
      setSections([{ kind: "added", title: "", body_md: "" }]);
      qc.invalidateQueries({ queryKey: ["admin-changelogs"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const publish = useMutation({
    mutationFn: (id: string) => apiSend(`v1/admin/changelogs/${id}/publish`, "POST"),
    onSuccess: () => {
      toast.success("Rilis dipublikasikan + notifikasi dikirim.");
      qc.invalidateQueries({ queryKey: ["admin-changelogs"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const archive = useMutation({
    mutationFn: (id: string) => apiSend(`v1/admin/changelogs/${id}/archive`, "POST"),
    onSuccess: () => {
      toast.success("Rilis diarsipkan.");
      qc.invalidateQueries({ queryKey: ["admin-changelogs"] });
    },
  });

  const items = Array.isArray(list.data) ? list.data : [];

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Changelog</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="grid gap-2 md:grid-cols-3">
            <div className="grid gap-1">
              <Label htmlFor="cv">Versi</Label>
              <Input id="cv" value={version} onChange={(e) => setVersion(e.target.value)} placeholder="0.5.0" />
            </div>
            <div className="grid gap-1 md:col-span-2">
              <Label htmlFor="ct">Judul</Label>
              <Input id="ct" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Perbaikan moderasi" />
            </div>
          </div>
          <div className="grid gap-1">
            <Label htmlFor="cs">Ringkasan</Label>
            <Textarea id="cs" value={summary} onChange={(e) => setSummary(e.target.value)} />
          </div>
          <div className="grid gap-1 md:w-64">
            <Label htmlFor="cp">Jadwal terbit (opsional)</Label>
            <Input id="cp" type="datetime-local" value={publishAt} onChange={(e) => setPublishAt(e.target.value)} />
            <p className="text-muted-foreground text-xs">
              Dikosongkan berarti draft manual; diisi masa depan berarti otomatis terbit.
            </p>
          </div>

          <div className="flex flex-col gap-2">
            {sections.map((s, i) => (
              <div key={i} className="grid gap-1 md:grid-cols-[130px_1fr_1fr]">
                <SelectField
                  value={s.kind}
                  onValueChange={(v) => setSections(sections.map((x, j) => (j === i ? { ...x, kind: v } : x)))}
                  options={KINDS.map((k) => ({ value: k, label: k }))}
                />
                <Input
                  value={s.title}
                  placeholder="Judul section"
                  onChange={(e) => setSections(sections.map((x, j) => (j === i ? { ...x, title: e.target.value } : x)))}
                />
                <Input
                  value={s.body_md}
                  placeholder="Detail"
                  onChange={(e) => setSections(sections.map((x, j) => (j === i ? { ...x, body_md: e.target.value } : x)))}
                />
              </div>
            ))}
            <Button variant="outline" size="sm" className="self-start" onClick={() => setSections([...sections, { kind: "fixed", title: "", body_md: "" }])}>
              Tambah section
            </Button>
          </div>

          <Button onClick={() => create.mutate()} disabled={!version || !title || create.isPending}>
            Simpan changelog
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Riwayat rilis</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Versi</TableHead>
                <TableHead>Judul</TableHead>
                <TableHead>Status</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="font-mono text-xs">{c.version}</TableCell>
                  <TableCell>{c.title}</TableCell>
                  <TableCell>
                    <Badge variant={c.status === "published" ? "default" : "secondary"}>{c.status}</Badge>
                  </TableCell>
                  <TableCell className="flex justify-end gap-2">
                    {c.status !== "published" && (
                      <Button size="sm" onClick={() => publish.mutate(c.id)}>
                        Terbitkan
                      </Button>
                    )}
                    {c.status === "published" && (
                      <Button size="sm" variant="outline" onClick={() => archive.mutate(c.id)}>
                        Arsipkan
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-sm text-muted-foreground">
                    {list.isLoading ? "Memuat..." : "Belum ada changelog."}
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
