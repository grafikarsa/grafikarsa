"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { apiGet, apiSend } from "@/lib/api";
import { Turnstile } from "@/components/security/turnstile";

type Thread = { id: string; category: string; title: string; status: string; upvotes: number };

export default function FeedbackPage() {
  const [items, setItems] = useState<Thread[]>([]);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [category, setCategory] = useState("bug");
  const [captcha, setCaptcha] = useState("");

  async function load() {
    const data = await apiGet<Thread[]>("v1/feedback");
    setItems(data);
  }

  useEffect(() => {
    load().catch(() => null);
  }, []);

  async function send() {
    try {
      await apiSend("v1/feedback", "POST", { category, title, body, captcha_token: captcha || undefined });
      setTitle("");
      setBody("");
      load();
      toast.success("Terkirim.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  async function vote(id: string) {
    try {
      await apiSend(`v1/feedback/${id}/vote`, "POST");
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Login untuk vote.");
    }
  }

  return (
    <main className="mx-auto max-w-2xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Masukan</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <div className="grid gap-2">
            <Label>Judul</Label>
            <select
              id="kategori"
              className="rounded border px-2 py-2 text-sm"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
            >
              <option value="bug">Bug</option>
              <option value="idea">Ide</option>
              <option value="question">Pertanyaan</option>
              <option value="praise">Apresiasi</option>
            </select>
            <Input id="judul" value={title} onChange={(e) => setTitle(e.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label>Isi</Label>
            <Textarea value={body} onChange={(e) => setBody(e.target.value)} />
            <Turnstile onToken={setCaptcha} />
          </div>
          <Button onClick={send}>Kirim</Button>
          {items.map((t) => (
            <div key={t.id} className="flex items-center justify-between border-b py-2">
              <div>
                <p className="font-semibold">{t.title}</p>
                <p className="text-muted-foreground text-xs">{t.category} · {t.status} · {t.upvotes} vote</p>
              </div>
              <Button variant="outline" size="sm" onClick={() => vote(t.id)}>Vote</Button>
            </div>
          ))}
        </CardContent>
      </Card>
    </main>
  );
}
