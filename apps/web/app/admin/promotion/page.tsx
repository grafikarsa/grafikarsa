"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { apiGet, apiSend } from "@/lib/api";

type Year = { id: string; start_year: number; is_active: boolean };
type Job = { id: string; status: string; executed_at: string };

export default function PromotionPage() {
  const [years, setYears] = useState<Year[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [preview, setPreview] = useState<Record<string, unknown> | null>(null);
  const [jobId, setJobId] = useState("");
  const [confirm, setConfirm] = useState("");

  async function load() {
    const y = await apiGet<Year[]>("v1/admin/years");
    setYears(y);
    const j = await apiGet<Job[]>("v1/admin/promotion/jobs").catch(() => []);
    setJobs(j);
  }

  useEffect(() => {
    load().catch((err: unknown) => toast.error(err instanceof Error ? err.message : "Gagal."));
  }, []);

  async function doPreview() {
    try {
      const r = await apiSend<{ id: string; preview: Record<string, unknown> }>("v1/admin/promotion/jobs/preview", "POST", {
        from_year_id: from,
        to_year_id: to,
      });
      setJobId(r.id);
      setPreview(r.preview);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  async function approve() {
    await apiSend(`v1/admin/promotion/jobs/${jobId}/approve`, "POST", { exceptions: [] });
    toast.success("Disetujui. Lanjut eksekusi.");
  }

  async function execute() {
    try {
      await apiSend(`v1/admin/promotion/jobs/${jobId}/execute`, "POST", { confirm });
      toast.success("Eksekusi selesai.");
      setPreview(null);
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  return (
    <main className="mx-auto max-w-3xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Kenaikan kelas</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid gap-2">
            <Label>Langkah 1 — Pilih tahun</Label>
            <div className="flex gap-2">
              <select value={from} onChange={(e) => setFrom(e.target.value)} className="border rounded p-2">
                <option value="">Dari...</option>
                {years.map((y) => (
                  <option key={y.id} value={y.id}>{y.start_year} {y.is_active && "(aktif)"}</option>
                ))}
              </select>
              <select value={to} onChange={(e) => setTo(e.target.value)} className="border rounded p-2">
                <option value="">Ke...</option>
                {years.map((y) => (
                  <option key={y.id} value={y.id}>{y.start_year}</option>
                ))}
              </select>
              <Button onClick={doPreview} disabled={!from || !to}>Preview</Button>
            </div>
          </div>
          {preview && (
            <div className="flex flex-col gap-2">
              <Label>Langkah 2 — Hasil preview</Label>
              <pre className="rounded bg-muted p-2 text-xs">{JSON.stringify(preview, null, 2)}</pre>
              <Button variant="outline" onClick={approve}>Setujui</Button>
              <Label>Langkah 3 — Ketik tahun tujuan untuk eksekusi</Label>
              <div className="flex gap-2">
                <Input value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder="cth. 2026" />
                <Button variant="destructive" onClick={execute}>Eksekusi</Button>
              </div>
            </div>
          )}
          <div>
            <Label>Riwayat jobs</Label>
            {jobs.map((j) => (
              <p key={j.id} className="text-sm">{j.id.slice(0, 8)} — {j.status}</p>
            ))}
          </div>
        </CardContent>
      </Card>
    </main>
  );
}
