"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { apiGet, apiSend } from "@/lib/api";

type Pending = { id: string; title: string; slug: string; username: string; submitted_at: string };

export default function ModerationPage() {
  const [items, setItems] = useState<Pending[]>([]);
  const [note, setNote] = useState("");

  async function load() {
    try {
      const data = await apiGet<{ data: Pending[] }>("v1/admin/portfolios/pending");
      setItems(data.data);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat.");
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function act(id: string, action: string) {
    try {
      await apiSend(`v1/admin/portfolios/${id}/${action}`, "POST", { note });
      toast.success("Berhasil.");
      setNote("");
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
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
            <Input value={note} onChange={(e) => setNote(e.target.value)} placeholder="Catatan review..." />
          </div>
          {items.map((it) => (
            <div key={it.id} className="flex flex-col gap-2 border-b py-2">
              <p className="font-semibold">{it.title}</p>
              <p className="text-muted-foreground text-xs">@{it.username} · {it.submitted_at}</p>
              <div className="flex gap-2">
                <Button size="sm" onClick={() => act(it.id, "approve")}>Setujui</Button>
                <Button size="sm" variant="outline" onClick={() => act(it.id, "request-changes")}>Minta perbaikan</Button>
                <Button size="sm" variant="destructive" onClick={() => act(it.id, "reject")}>Tolak</Button>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </main>
  );
}
