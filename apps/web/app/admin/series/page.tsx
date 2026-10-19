"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select-field";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { apiGet, apiSend } from "@/lib/api";

type Series = { id: string; slug: string; name: string; description: string };
type BlockType = "text" | "image" | "table" | "youtube" | "button" | "embed" | "figma" | "canva" | "ppt" | "pdf" | "doc";

export default function SeriesPage() {
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [instruction, setInstruction] = useState("");
  const [blockType, setBlockType] = useState<BlockType>("text");
  const [selected, setSelected] = useState<string>("");

  const list = useQuery({
    queryKey: ["admin-series"],
    queryFn: () => apiGet<Series[]>("v1/admin/series"),
  });

  const detail = useQuery({
    queryKey: ["series-detail", selected],
    queryFn: () => apiGet<{ blocks: { block_type: string; instruction: string }[] }>(`v1/series/${selected}`),
    enabled: Boolean(selected),
  });

  const create = useMutation({
    mutationFn: () =>
      apiSend("v1/admin/series", "POST", {
        name,
        slug,
        description,
        blocks: instruction
          ? [{ block_type: blockType, instruction }]
          : [{ block_type: "text", instruction: "Isi instruksi template" }],
      }),
    onSuccess: () => {
      toast.success("Series dibuat (minimal 1 block template).");
      setName("");
      setSlug("");
      setDescription("");
      setInstruction("");
      qc.invalidateQueries({ queryKey: ["admin-series"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const items = Array.isArray(list.data) ? list.data : [];

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Series (template portfolio)</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="grid gap-2 md:grid-cols-2">
            <div className="grid gap-1">
              <Label htmlFor="sname">Nama</Label>
              <Input id="sname" value={name} onChange={(e) => setName(e.target.value)} placeholder="PJBL Semester 1" />
            </div>
            <div className="grid gap-1">
              <Label htmlFor="sslug">Slug</Label>
              <Input id="sslug" value={slug} onChange={(e) => setSlug(e.target.value)} placeholder="pjbl-s1" />
            </div>
          </div>
          <div className="grid gap-1">
            <Label htmlFor="sdesc">Deskripsi</Label>
            <Textarea id="sdesc" value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>
          <div className="grid gap-2 md:grid-cols-[180px_1fr_auto] md:items-end">
            <div className="grid gap-1">
              <Label>Block pertama</Label>
              <SelectField
                value={blockType}
                onValueChange={(v) => setBlockType(v as BlockType)}
                options={["text", "image", "table", "youtube", "button", "embed", "figma", "canva", "ppt", "pdf", "doc"].map((t) => ({ value: t, label: t }))}
              />
            </div>
            <div className="grid gap-1">
              <Label htmlFor="sinstr">Instruksi</Label>
              <Input id="sinstr" value={instruction} onChange={(e) => setInstruction(e.target.value)} placeholder="Slide 1: latar belakang masalah" />
            </div>
            <Button onClick={() => create.mutate()} disabled={!name || !slug || create.isPending}>
              Buat series
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Daftar series</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nama</TableHead>
                <TableHead>Slug</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((s) => (
                <TableRow key={s.id}>
                  <TableCell>{s.name}</TableCell>
                  <TableCell className="font-mono text-xs">{s.slug}</TableCell>
                  <TableCell className="text-right">
                    <Button variant="outline" size="sm" onClick={() => setSelected(s.id)}>
                      Lihat block
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
              {items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-sm text-muted-foreground">
                    {list.isLoading ? "Memuat..." : "Belum ada series."}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>

          {selected && (
            <div className="mt-3 rounded border p-3">
              <p className="mb-2 text-sm font-medium">Block template</p>
              {(detail.data?.blocks ?? []).map((b, i) => (
                <p key={i} className="text-xs">
                  [{b.block_type}] {b.instruction}
                </p>
              ))}
              {(detail.data?.blocks ?? []).length === 0 && (
                <p className="text-muted-foreground text-xs">{detail.isLoading ? "Memuat..." : "Tanpa block."}</p>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
