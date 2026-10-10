"use client";

import { DndContext, closestCenter, type DragEndEvent } from "@dnd-kit/core";
import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { use, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import {
  BLOCK_TYPES,
  BlockForm,
  type BlockType,
  type Draft,
  updateBlock,
  useUpload,
} from "@/components/portfolio/block-form";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select-field";
import { apiGet, apiSend } from "@/lib/api";

type Block = { id: string; block_type: string; position: number; payload: Record<string, unknown> };

function previewText(b: Block): string {
  const p = b.payload as Record<string, unknown>;
  const v = p.markdown ?? p.alt ?? p.url ?? p.video_id ?? p.label ?? p.title ?? "";
  return String(v).slice(0, 90);
}

function SortableBlock({
  b,
  disabled,
  onDelete,
  onSave,
}: {
  b: Block;
  disabled: boolean;
  onDelete: () => void;
  onSave: (payload: Draft) => Promise<void>;
}) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: b.id });
  const style = { transform: CSS.Transform.toString(transform), transition };
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);

  return (
    <div ref={setNodeRef} style={style} className="flex flex-col gap-2 border-b py-3">
      <div className="flex items-center gap-2">
        <button
          {...attributes}
          {...listeners}
          className="cursor-grab px-2"
          title="Seret untuk susun ulang"
          aria-label="Pindahkan block"
          disabled={disabled}
        >
          ⠿
        </button>
        <Badge variant="secondary">{b.block_type}</Badge>
        <span className="text-muted-foreground flex-1 truncate text-sm">{previewText(b)}</span>
        <Button variant="ghost" size="sm" onClick={() => setEditing((v) => !v)} disabled={disabled || busy}>
          {editing ? "Tutup" : "Edit"}
        </Button>
        <Button variant="ghost" size="sm" onClick={onDelete} disabled={disabled}>
          Hapus
        </Button>
      </div>
      {editing && (
        <div className="pl-8">
          <BlockForm
            blockType={b.block_type as BlockType}
            initial={b.payload}
            busy={busy}
            submitLabel="Simpan perubahan"
            onCancel={() => setEditing(false)}
            onSubmit={async (payload) => {
              setBusy(true);
              try {
                await onSave(payload);
                setEditing(false);
              } finally {
                setBusy(false);
              }
            }}
          />
        </div>
      )}
    </div>
  );
}

export default function EditorPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const { upload, progress } = useUpload();

  const [title, setTitle] = useState("");
  const [status, setStatus] = useState("");
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [newType, setNewType] = useState<BlockType>("text");
  const [savedAt, setSavedAt] = useState<string | null>(null);
  const [tagIDs, setTagIDs] = useState<string[]>([]);
  const [seriesID, setSeriesID] = useState("");
  const [metaLoaded, setMetaLoaded] = useState(false);
  const titleTimer = useRef<NodeJS.Timeout | null>(null);

  const tags = useQuery({
    queryKey: ["tags"],
    queryFn: () => apiGet<{ id: string; name: string }[]>("v1/tags"),
  });
  const series = useQuery({
    queryKey: ["series"],
    queryFn: () => apiGet<{ id: string; name: string }[]>("v1/series"),
  });

  async function load(silent = false) {
    try {
      const p = await apiGet<{
        title: string;
        status: string;
        blocks: Block[];
        tags?: { id: string; slug: string; name: string }[];
        series_id?: string | null;
      }>(`v1/portfolios/id/${id}`);
      setTitle(p.title);
      setStatus(p.status);
      setBlocks(p.blocks ?? []);
      // Tag & series hanya dimuat sekali supaya pilihan user tidak tertimpa
      // oleh refresh setelah menambah/mengedit block.
      if (!metaLoaded) {
        if (Array.isArray(p.tags)) setTagIDs(p.tags.map((t) => t.id));
        if (p.series_id) setSeriesID(p.series_id);
        setMetaLoaded(true);
      }
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
    // Autosave judul + penanda draft: pekerjaan tidak hilang bila tab ditutup
    // sebelum block sempat ditambahkan.
    titleTimer.current = setTimeout(async () => {
      try {
        await apiSend(`v1/portfolios/id/${id}`, "PATCH", {
          title: v,
          draft_payload: { blocks_count: blocks.length, saved_from: "editor" },
        });
        setSavedAt(new Date().toLocaleTimeString("id-ID"));
      } catch {
        /* diamkan; akan dicoba lagi saat judul berubah */
      }
    }, 10000);
  }

  async function addBlock(payload: Draft) {
    try {
      await apiSend(`v1/portfolios/${id}/blocks`, "POST", { block_type: newType, payload });
      toast.success("Block ditambahkan.");
      load(true);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menambah block.");
    }
  }

  async function removeBlock(bid: string) {
    try {
      await apiSend(`v1/blocks/${bid}`, "DELETE");
      load(true);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menghapus.");
    }
  }

  async function saveBlock(bid: string, payload: Draft) {
    try {
      await updateBlock(bid, payload);
      toast.success("Block disimpan.");
      load(true);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menyimpan block.");
    }
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
    setBlocks(next.map((bid) => blocks.find((b) => b.id === bid)!).filter(Boolean));
    try {
      await apiSend(`v1/portfolios/${id}/blocks/reorder`, "PUT", {
        orders: next.map((bid, i) => ({ id: bid, position: i })),
      });
      toast.success("Urutan tersimpan.");
    } catch {
      toast.error("Gagal menyimpan urutan.");
      load(true);
    }
  }

  async function uploadThumb(file: File) {
    try {
      const res = await upload("thumbnail", file);
      await apiSend(`v1/portfolios/id/${id}`, "PATCH", { thumbnail_media_id: res.media_id });
      toast.success("Thumbnail tersimpan.");
      load(true);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Upload thumbnail gagal.");
    }
  }

  async function saveMeta() {
    try {
      await apiSend(`v1/portfolios/id/${id}`, "PATCH", {
        tag_ids: tagIDs,
        series_id: seriesID || null,
      });
      toast.success("Tag & series disimpan.");
      load(true);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menyimpan tag.");
    }
  }

  async function applySeries() {
    try {
      const res = await apiSend<{ created: number }>(`v1/portfolios/id/${id}/apply-series`, "POST");
      toast.success(`Template series diterapkan (${res.created} block).`);
      load(true);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menerapkan template.");
    }
  }

  async function submit() {
    try {
      if (titleTimer.current) {
        clearTimeout(titleTimer.current);
        titleTimer.current = null;
      }
      await apiSend(`v1/portfolios/id/${id}/submit`, "POST");
      toast.success("Dikirim ke review.");
      router.push("/me/portfolios");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Belum lengkap.");
    }
  }

  const readOnly = status === "pending_review";

  return (
    <main className="mx-auto max-w-3xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>
            Edit portfolio{" "}
            {savedAt && <span className="text-muted-foreground text-xs font-normal">· Tersimpan {savedAt}</span>}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid gap-2">
            <Label htmlFor="title">Judul (autosave 10 dtk)</Label>
            <Input id="title" value={title} onChange={(e) => onTitleChange(e.target.value)} disabled={readOnly} />
            <p className="text-muted-foreground text-xs">Minimal 10 karakter saat dikirim review.</p>
          </div>

          <div className="grid gap-2">
            <Label>Thumbnail 4:3 (maks 5MB)</Label>
            <input
              type="file"
              accept="image/jpeg,image/png,image/webp"
              disabled={readOnly}
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (f) uploadThumb(f).catch(() => null);
              }}
            />
            {progress !== null && <span className="text-xs">Mengunggah… {progress}%</span>}
          </div>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Block konten</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              <DndContext collisionDetection={closestCenter} onDragEnd={onDragEnd}>
                <SortableContext items={blocks.map((b) => b.id)} strategy={verticalListSortingStrategy}>
                  {blocks.map((b) => (
                    <SortableBlock
                      key={b.id}
                      b={b}
                      disabled={readOnly}
                      onDelete={() => removeBlock(b.id)}
                      onSave={(payload) => saveBlock(b.id, payload)}
                    />
                  ))}
                </SortableContext>
              </DndContext>
              {blocks.length === 0 && (
                <p className="text-muted-foreground text-sm">
                  Belum ada block. Tambahkan minimal satu sebelum kirim review.
                </p>
              )}

              {!readOnly && (
                <div className="flex flex-col gap-3 rounded border p-3">
                  <div className="grid gap-1 sm:w-72">
                    <Label>Tambah block</Label>
                    <SelectField
                      value={newType}
                      onValueChange={(v) => setNewType(v as BlockType)}
                      options={BLOCK_TYPES.map((b) => ({ value: b.value, label: b.label }))}
                    />
                  </div>
                  <BlockForm blockType={newType} onSubmit={addBlock} />
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Tag & series</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              <div className="flex flex-wrap gap-2">
                {(tags.data ?? []).map((t) => (
                  <label key={t.id} className="flex items-center gap-1 rounded border px-2 py-1 text-xs">
                    <input
                      type="checkbox"
                      checked={tagIDs.includes(t.id)}
                      disabled={readOnly}
                      onChange={(e) => setTagIDs(e.target.checked ? [...tagIDs, t.id] : tagIDs.filter((x) => x !== t.id))}
                    />
                    {t.name}
                  </label>
                ))}
                {(tags.data ?? []).length === 0 && <p className="text-muted-foreground text-xs">Belum ada tag.</p>}
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <SelectField
                  value={seriesID}
                  onValueChange={setSeriesID}
                  placeholder="(tanpa series)"
                  className="w-56"
                  options={[
                    { value: "", label: "(tanpa series)" },
                    ...(series.data ?? []).map((x) => ({ value: x.id, label: x.name })),
                  ]}
                />
                <Button variant="outline" size="sm" onClick={saveMeta} disabled={readOnly}>
                  Simpan tag & series
                </Button>
                <Button variant="ghost" size="sm" onClick={applySeries} disabled={readOnly || !seriesID}>
                  Terapkan template series
                </Button>
              </div>
              <p className="text-muted-foreground text-xs">
                Minimal 1 tag wajib sebelum kirim review. Template series hanya bisa diterapkan saat portfolio belum
                punya block.
              </p>
            </CardContent>
          </Card>

          <div className="flex flex-wrap gap-2">
            <Button onClick={submit} disabled={readOnly}>
              Kirim review
            </Button>
            <Button variant="outline" onClick={() => router.push("/me/portfolios")}>
              Kembali
            </Button>
          </div>
        </CardContent>
      </Card>
    </main>
  );
}
