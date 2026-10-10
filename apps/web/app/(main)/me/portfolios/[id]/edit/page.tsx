"use client";

import { DndContext, closestCenter, type DragEndEvent } from "@dnd-kit/core";
import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useRouter } from "next/navigation";
import { use, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { apiSend, apiGet } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

type Block = { id: string; block_type: string; position: number; payload: Record<string, unknown> };

function SortableBlock({ b, onDelete }: { b: Block; onDelete: () => void }) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: b.id });
  const style = { transform: CSS.Transform.toString(transform), transition };
  const p = b.payload as Record<string, string>;
  return (
    <div ref={setNodeRef} style={style} className="flex items-center gap-2 border-b py-2">
      <button {...attributes} {...listeners} className="cursor-grab px-2" title="Seret untuk susun ulang">⠿</button>
      <div className="flex-1">
        <p className="text-xs text-muted-foreground">[{b.block_type}]</p>
        <p className="text-sm">{String(p.markdown ?? p.url ?? p.video_id ?? p.label ?? "").slice(0, 80)}</p>
      </div>
      <Button variant="ghost" size="sm" onClick={onDelete}>Hapus</Button>
    </div>
  );
}

export default function EditorPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [status, setStatus] = useState("");
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [text, setText] = useState("");
  const [yt, setYt] = useState("");
  const [savedAt, setSavedAt] = useState<string | null>(null);
  const titleTimer = useRef<NodeJS.Timeout | null>(null);

  async function load(silent = false) {
    try {
      const p = await apiGet<{ title: string; status: string; blocks: Block[] }>(`v1/portfolios/id/${id}`);
      setTitle(p.title);
      setStatus(p.status);
      setBlocks(p.blocks);
    } catch {
      if (!silent) toast.error("Gagal memuat.");
    }
  }

  useEffect(() => {
    load().catch(() => null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  function onTitleChange(v: string) {
    setTitle(v);
    if (titleTimer.current) clearTimeout(titleTimer.current);
    titleTimer.current = setTimeout(async () => {
      try {
        await apiSend(`v1/portfolios/id/${id}`, "PATCH", { title: v });
        setSavedAt(new Date().toLocaleTimeString("id-ID"));
      } catch {
        /* abaikan autosave gagal */
      }
    }, 10000);
  }

  async function addBlock(block_type: string, payload: Record<string, unknown>) {
    await apiSend(`v1/portfolios/${id}/blocks`, "POST", { block_type, payload });
    load(true);
  }

  async function addText() {
    if (!text.trim()) return;
    await addBlock("text", { markdown: text });
    setText("");
  }

  function ytId(url: string): string | null {
    const m = url.match(/(?:v=|youtu\.be\/|embed\/)([A-Za-z0-9_-]{11})/);
    return m ? m[1] : null;
  }

  async function addYt() {
    const vid = ytId(yt);
    if (!vid) {
      toast.error("URL YouTube tidak valid.");
      return;
    }
    await addBlock("youtube", { video_id: vid });
    setYt("");
  }

  async function removeBlock(bid: string) {
    await apiSend(`v1/blocks/${bid}`, "DELETE");
    load(true);
  }

  async function onDragEnd(e: DragEndEvent) {
    const { active, over } = e;
    if (!over || active.id === over.id) return;
    const order = blocks.map((b) => b.id);
    const from = order.indexOf(String(active.id));
    const to = order.indexOf(String(over.id));
    const next = [...order];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    setBlocks(next.map((bid, i) => blocks.find((b) => b.id === bid)!).filter(Boolean));
    await apiSend(`v1/portfolios/${id}/blocks/reorder`, "PUT", {
      orders: next.map((bid, i) => ({ id: bid, position: i })),
    });
    toast.success("Urutan tersimpan.");
  }

  async function uploadThumb(file: File) {
    const fd = new FormData();
    fd.append("purpose", "thumbnail");
    fd.append("file", file);
    const { accessToken, csrfToken } = useAuthStore.getState();
    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/v1/media/upload");
    if (accessToken) xhr.setRequestHeader("Authorization", `Bearer ${accessToken}`);
    if (csrfToken) xhr.setRequestHeader("X-CSRF-Token", csrfToken);
    xhr.onload = async () => {
      const json = JSON.parse(xhr.responseText);
      if (!json.success) {
        toast.error(json.error.message);
        return;
      }
      await apiSend(`v1/portfolios/id/${id}`, "PATCH", { thumbnail_media_id: json.data.media_id });
      toast.success("Thumbnail tersimpan.");
    };
    xhr.send(fd);
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
          <CardTitle>Edit portfolio {savedAt && <span className="text-xs font-normal text-muted-foreground">· Tersimpan {savedAt}</span>}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid gap-2">
            <Label>Judul (autosave 10 dtk)</Label>
            <Input value={title} onChange={(e) => onTitleChange(e.target.value)} disabled={status === "pending_review"} />
          </div>
          <div className="grid gap-2">
            <Label>Thumbnail 4:3 (maks 5MB)</Label>
            <input type="file" accept="image/jpeg,image/png,image/webp" onChange={(e) => e.target.files?.[0] && uploadThumb(e.target.files[0])} />
          </div>
          <DndContext collisionDetection={closestCenter} onDragEnd={onDragEnd}>
            <SortableContext items={blocks.map((b) => b.id)} strategy={verticalListSortingStrategy}>
              {blocks.map((b) => (
                <SortableBlock key={b.id} b={b} onDelete={() => removeBlock(b.id)} />
              ))}
            </SortableContext>
          </DndContext>
          <div className="flex gap-2">
            <Textarea value={text} onChange={(e) => setText(e.target.value)} placeholder="Tambah block teks..." />
            <Button variant="outline" onClick={addText}>Tambah</Button>
          </div>
          <div className="flex gap-2">
            <Input value={yt} onChange={(e) => setYt(e.target.value)} placeholder="URL YouTube..." />
            <Button variant="outline" onClick={addYt}>+ Video</Button>
          </div>
          <Button onClick={submit} disabled={status === "pending_review"}>Kirim review</Button>
        </CardContent>
      </Card>
    </main>
  );
}
