"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { apiGet, apiSend } from "@/lib/api";

type Pending = { id: string; title: string; slug: string; username: string; submitted_at: string };

function sla(submitted: string): {
  label: string;
  variant: "default" | "destructive" | "secondary";
} {
  const hours = (Date.now() - new Date(submitted).getTime()) / 36e5;
  if (hours > 72) return { label: "Lewat SLA", variant: "destructive" };
  if (hours > 48) return { label: "Hampir SLA", variant: "secondary" };
  return { label: "Baru", variant: "default" };
}

export default function ModerationPage() {
  const [items, setItems] = useState<Pending[]>([]);
  const [note, setNote] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());

  async function load() {
    try {
      const data = await apiGet<Pending[]>("v1/admin/portfolios/pending");
      setItems(Array.isArray(data) ? data : []);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat.");
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function act(id: string, action: string) {
    const needNote = action === "reject" || action === "request-changes";
    if (needNote && note.trim().length < 5) {
      toast.error("Catatan minimal 5 karakter untuk tolak/minta perbaikan.");
      return;
    }
    try {
      await apiSend(`v1/admin/portfolios/${id}/${action}`, "POST", { note });
      toast.success("Berhasil.");
      setNote("");
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  async function bulkApprove() {
    const ids = [...selected].slice(0, 20);
    if (ids.length === 0) return;
    let ok = 0;
    for (const id of ids) {
      try {
        await apiSend(`v1/admin/portfolios/${id}/approve`, "POST", {});
        ok++;
      } catch {
        /* lanjut */
      }
    }
    toast.success(`${ok}/${ids.length} disetujui.`);
    setSelected(new Set());
    load();
  }

  function toggle(id: string) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else if (next.size < 20) next.add(id);
    else toast.error("Maksimal 20 sekaligus.");
    setSelected(next);
  }

  return (
    <main className="mx-auto max-w-3xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Moderasi ({items.length} antre)</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid gap-2">
            <Label>Catatan (wajib untuk tolak/perbaikan)</Label>
            <Input
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Catatan review..."
            />
          </div>
          {selected.size > 0 && (
            <Button onClick={bulkApprove}>Setujui {selected.size} terpilih</Button>
          )}
          {items.map((it) => (
            <div key={it.id} className="flex flex-col gap-2 border-b py-2">
              <div className="flex items-center gap-2">
                <Checkbox checked={selected.has(it.id)} onCheckedChange={() => toggle(it.id)} />
                <p className="font-semibold">{it.title}</p>
                <Badge variant={sla(it.submitted_at).variant}>{sla(it.submitted_at).label}</Badge>
              </div>
              <p className="text-muted-foreground text-xs">
                @{it.username} · {it.submitted_at}
              </p>
              <div className="flex gap-2">
                <Button size="sm" onClick={() => act(it.id, "approve")}>
                  Setujui
                </Button>
                <Button size="sm" variant="outline" onClick={() => act(it.id, "request-changes")}>
                  Minta perbaikan
                </Button>
                <Button size="sm" variant="destructive" onClick={() => act(it.id, "reject")}>
                  Tolak
                </Button>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </main>
  );
}
