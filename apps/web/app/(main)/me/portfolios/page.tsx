"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { apiGet, apiSend } from "@/lib/api";

type Portfolio = { id: string; title: string; slug: string; status: string };

export default function MyPortfoliosPage() {
  const [items, setItems] = useState<Portfolio[]>([]);
  const [title, setTitle] = useState("");
  const router = useRouter();

  async function load() {
    try {
      const data = await apiGet<Portfolio[]>("v1/portfolios/me");
      setItems(data);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat.");
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function create() {
    try {
      await apiSend("v1/portfolios", "POST", { title });
      setTitle("");
      load();
      toast.success("Draft dibuat.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  return (
    <main className="mx-auto max-w-2xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Portfolio saya</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex gap-2">
            <div className="grid flex-1 gap-2">
              <Label htmlFor="title">Judul baru</Label>
              <Input id="title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Judul portfolio..." />
            </div>
            <Button className="self-end" onClick={create}>Buat draft</Button>
          </div>
          {items.map((it) => (
            <div key={it.id} className="flex items-center justify-between border-b py-2">
              <div>
                <p className="font-semibold">{it.title}</p>
                <p className="text-muted-foreground text-xs">{it.status}</p>
              </div>
              <Button variant="outline" size="sm" onClick={() => router.push(`/me/portfolios/${it.id}/edit`)}>Edit</Button>
            </div>
          ))}
        </CardContent>
      </Card>
    </main>
  );
}
