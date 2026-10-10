"use client";

import { useRouter } from "next/navigation";
import { use, useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { apiGet, apiSend } from "@/lib/api";

type Block = { id: string; block_type: string; position: number; payload: Record<string, unknown> };

export default function EditorPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [text, setText] = useState("");

  async function load() {
    const p = await apiGet<{ title: string; status: string; blocks: Block[] }>(`v1/portfolios/id/${id}`);
    setTitle(p.title);
    setBlocks(p.blocks);
  }

  useEffect(() => {
    load().catch(() => toast.error("Gagal memuat."));
  }, [id]);

  async function saveTitle() {
    await apiSend(`v1/portfolios/id/${id}`, "PATCH", { title });
    toast.success("Judul tersimpan.");
  }

  async function addText() {
    if (!text.trim()) return;
    await apiSend(`v1/portfolios/${id}/blocks`, "POST", { block_type: "text", payload: { markdown: text } });
    setText("");
    load();
  }

  async function removeBlock(bid: string) {
    await apiSend(`v1/blocks/${bid}`, "DELETE");
    load();
  }

  async function submit() {
    try {
      await apiSend(`v1/portfolios/id/${id}/submit`, "POST");
      toast.success("Dikirim ke review.");
      router.push("/me/portfolios");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Belum lengkap.");
    }
  }

  return (
    <main className="mx-auto max-w-2xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Edit portfolio</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid gap-2">
            <Label>Judul</Label>
            <div className="flex gap-2">
              <Input value={title} onChange={(e) => setTitle(e.target.value)} />
              <Button variant="outline" onClick={saveTitle}>Simpan</Button>
            </div>
          </div>
          {blocks.map((b) => (
            <div key={b.id} className="flex items-center justify-between border-b py-2">
              <p className="text-sm">[{b.block_type}] {String((b.payload as Record<string, string>).markdown ?? "").slice(0, 60)}</p>
              <Button variant="ghost" size="sm" onClick={() => removeBlock(b.id)}>Hapus</Button>
            </div>
          ))}
          <div className="flex gap-2">
            <Input value={text} onChange={(e) => setText(e.target.value)} placeholder="Tambah block teks..." />
            <Button variant="outline" onClick={addText}>Tambah</Button>
          </div>
          <Button onClick={submit}>Kirim review</Button>
        </CardContent>
      </Card>
    </main>
  );
}
