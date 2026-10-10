"use client";

import { useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
import { toast } from "sonner";
import { Turnstile } from "@/components/security/turnstile";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select-field";
import { Textarea } from "@/components/ui/textarea";
import { apiGet, apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

type Thread = { id: string; category: string; title: string; status: string; upvotes: number };
type ThreadDetail = {
  id: string;
  title: string;
  body: string;
  status: string;
  priority: string;
  sla_hours: number;
  sla_breached: boolean;
  messages: { body: string; is_internal: boolean }[];
};

const CATEGORIES = [
  { value: "bug", label: "Bug" },
  { value: "idea", label: "Ide" },
  { value: "question", label: "Pertanyaan" },
  { value: "praise", label: "Apresiasi" },
];

const LS_KEY = "grafikarsa:last_feedback";

function FeedbackInner() {
  const params = useSearchParams();
  const user = useAuthStore((s) => s.user);
  const urlToken = params.get("t") ?? "";
  const threadParam = params.get("thread") ?? "";

  const [items, setItems] = useState<Thread[]>([]);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [category, setCategory] = useState("bug");
  const [captcha, setCaptcha] = useState("");
  const [detail, setDetail] = useState<ThreadDetail | null>(null);
  const [reply, setReply] = useState("");
  const [myToken, setMyToken] = useState<string | null>(urlToken || null);

  async function load() {
    try {
      const data = await apiGet<Thread[]>("v1/feedback");
      setItems(Array.isArray(data) ? data : []);
    } catch {
      setItems([]);
    }
  }

  async function loadThread(id: string, t: string) {
    try {
      const d = await apiGet<ThreadDetail>(`v1/feedback/${id}${t ? `?t=${encodeURIComponent(t)}` : ""}`);
      setDetail(d);
      setMyToken(t || null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Thread tidak bisa dibuka.");
    }
  }

  useEffect(() => {
    load().catch(() => null);
    if (threadParam) {
      loadThread(threadParam, urlToken).catch(() => null);
      return;
    }
    if (urlToken) {
      // Token guest tersimpan di browser ini -> buka thread miliknya.
      try {
        const saved = JSON.parse(localStorage.getItem(LS_KEY) ?? "{}");
        if (saved?.token === urlToken && saved?.id) {
          loadThread(saved.id, urlToken).catch(() => null);
        }
      } catch {
        /* abaikan */
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [urlToken, threadParam]);

  async function send() {
    try {
      const res = await apiSend<{ id: string; follow_up_token?: string }>("v1/feedback", "POST", {
        category,
        title,
        body,
        captcha_token: captcha || undefined,
      });
      if (res.follow_up_token) {
        localStorage.setItem(LS_KEY, JSON.stringify({ id: res.id, token: res.follow_up_token }));
        setMyToken(res.follow_up_token);
        toast.success("Terkirim. Simpan tautan balasan di bawah.");
      } else {
        toast.success("Terkirim.");
      }
      setTitle("");
      setBody("");
      load().catch(() => null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal mengirim.");
    }
  }

  async function vote(id: string) {
    try {
      await apiSend(`v1/feedback/${id}/vote`, "POST");
      load().catch(() => null);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Login untuk vote.");
    }
  }

  async function sendReply() {
    if (!detail) return;
    try {
      await apiSend(`v1/feedback/${detail.id}/reply`, "POST", { body: reply, t: myToken ?? undefined });
      setReply("");
      loadThread(detail.id, myToken ?? "").catch(() => null);
      toast.success("Balasan terkirim.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal membalas.");
    }
  }

  return (
    <main className="mx-auto flex max-w-2xl flex-col gap-4 p-4">
      {myToken && (
        <Card>
          <CardContent className="flex flex-col gap-2 p-3">
            <p className="text-xs">Tautan balasanmu (simpan untuk membuka thread ini tanpa login):</p>
            <code className="bg-muted truncate rounded p-2 text-xs">{`/feedback?t=${myToken}`}</code>
          </CardContent>
        </Card>
      )}

      {detail && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              {detail.title}
              {detail.sla_breached && <Badge variant="destructive">lewat SLA</Badge>}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <p className="text-muted-foreground text-xs">
              {detail.status} · prioritas {detail.priority} · target {detail.sla_hours} jam
            </p>
            <p className="text-sm">{detail.body}</p>
            <div className="flex flex-col gap-2 rounded border p-3">
              {(detail.messages ?? []).map((m, i) => (
                <p key={i} className={`text-sm ${m.is_internal ? "text-amber-600 italic" : ""}`}>
                  {m.is_internal ? "[internal] " : ""}
                  {m.body}
                </p>
              ))}
              {(detail.messages ?? []).length === 0 && (
                <p className="text-muted-foreground text-xs">Belum ada balasan.</p>
              )}
            </div>
            <Textarea value={reply} onChange={(e) => setReply(e.target.value)} placeholder="Balasan..." rows={3} />
            <Button size="sm" className="self-start" onClick={sendReply} disabled={reply.trim().length < 2}>
              Kirim balasan
            </Button>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Masukan</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="grid gap-2">
            <Label htmlFor="kategori">Kategori</Label>
            <SelectField value={category} onValueChange={setCategory} options={CATEGORIES} className="w-full" />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="judul">Judul (min 5)</Label>
            <Input id="judul" value={title} onChange={(e) => setTitle(e.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="isi">Isi (min 10)</Label>
            <Textarea id="isi" value={body} onChange={(e) => setBody(e.target.value)} rows={5} />
          </div>
          {!user && <Turnstile onToken={setCaptcha} />}
          <Button onClick={send} disabled={title.trim().length < 5 || body.trim().length < 10}>
            Kirim
          </Button>
          <p className="text-muted-foreground text-xs">
            {user
              ? "Kamu login: balasan muncul di halaman notifikasi."
              : "Tanpa login, simpan tautan balasan yang muncul setelah mengirim."}
          </p>

          {items.map((t) => (
            <div key={t.id} className="flex items-center justify-between border-b py-2">
              <div>
                <p className="font-semibold">{t.title}</p>
                <p className="text-muted-foreground text-xs">
                  {t.category} · {t.status} · {t.upvotes} vote
                </p>
              </div>
              <Button variant="outline" size="sm" onClick={() => vote(t.id)}>
                Vote
              </Button>
            </div>
          ))}
          {items.length === 0 && <p className="text-muted-foreground text-sm">Belum ada masukan publik.</p>}
        </CardContent>
      </Card>
    </main>
  );
}

export default function FeedbackPage() {
  return (
    <Suspense fallback={<main className="p-4 text-sm">Memuat...</main>}>
      <FeedbackInner />
    </Suspense>
  );
}
