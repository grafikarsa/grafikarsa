"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select-field";
import { apiGet, apiSend } from "@/lib/api";

type Year = { id: string; start_year: number; is_active: boolean };
type Job = { id: string; status: string; executed_at: string };
type Exception = { user_id: string; action: "repeat" | "transfer" };

export default function PromotionPage() {
  const [years, setYears] = useState<Year[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [preview, setPreview] = useState<Record<string, unknown> | null>(null);
  const [jobId, setJobId] = useState("");
  const [confirm, setConfirm] = useState("");
  const [exceptions, setExceptions] = useState<Exception[]>([]);
  const [users, setUsers] = useState<{ id: string; username: string; name: string }[]>([]);
  const [excUser, setExcUser] = useState("");
  const [excAction, setExcAction] = useState<"repeat" | "transfer">("repeat");

  async function load() {
    const y = await apiGet<Year[]>("v1/admin/years");
    setYears(y);
    const j = await apiGet<Job[]>("v1/admin/promotion/jobs").catch(() => []);
    setJobs(j);
  }

  useEffect(() => {
    load().catch((err: unknown) => toast.error(err instanceof Error ? err.message : "Gagal."));
    // Daftar user untuk pengecualian (naik kelas / pindah).
    apiGet<{ id: string; username: string; name: string }[]>("v1/admin/users?per_page=200")
      .then((u) => setUsers(Array.isArray(u) ? u : []))
      .catch(() => setUsers([]));
  }, []);

  async function doPreview() {
    try {
      const r = await apiSend<{ id: string; preview: Record<string, unknown> }>(
        "v1/admin/promotion/jobs/preview",
        "POST",
        {
          from_year_id: from,
          to_year_id: to,
        },
      );
      setJobId(r.id);
      setPreview(r.preview);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  async function approve() {
    try {
      await apiSend(`v1/admin/promotion/jobs/${jobId}/approve`, "POST", { exceptions });
      toast.success(
        exceptions.length > 0
          ? `Disetujui dengan ${exceptions.length} pengecualian. Lanjut eksekusi.`
          : "Disetujui. Lanjut eksekusi.",
      );
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menyetujui.");
    }
  }

  async function execute() {
    try {
      await apiSend(`v1/admin/promotion/jobs/${jobId}/execute`, "POST", { confirm });
      toast.success("Eksekusi selesai.");
      setPreview(null);
      setExceptions([]);
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
              <select
                value={from}
                onChange={(e) => setFrom(e.target.value)}
                className="border rounded p-2"
              >
                <option value="">Dari...</option>
                {years.map((y) => (
                  <option key={y.id} value={y.id}>
                    {y.start_year} {y.is_active && "(aktif)"}
                  </option>
                ))}
              </select>
              <select
                value={to}
                onChange={(e) => setTo(e.target.value)}
                className="border rounded p-2"
              >
                <option value="">Ke...</option>
                {years.map((y) => (
                  <option key={y.id} value={y.id}>
                    {y.start_year}
                  </option>
                ))}
              </select>
              <Button onClick={doPreview} disabled={!from || !to}>
                Preview
              </Button>
            </div>
          </div>
          {preview && (
            <div className="flex flex-col gap-3">
              <Label>Langkah 2 — Hasil preview</Label>
              <div className="rounded border p-3 text-sm">
                <p>
                  Total siswa: <strong>{(preview as { total?: number }).total ?? 0}</strong> · naik:{" "}
                  <strong>
                    {Object.values(
                      (preview as { naik?: Record<string, number> }).naik ?? {},
                    ).reduce((a, b) => a + Number(b), 0)}
                  </strong>{" "}
                  · lulus: <strong>{(preview as { lulus?: number }).lulus ?? 0}</strong>
                </p>
                {Array.isArray((preview as { warnings?: string[] }).warnings) &&
                  ((preview as { warnings?: string[] }).warnings ?? []).length > 0 && (
                    <ul className="mt-2 list-inside list-disc text-xs text-amber-600">
                      {((preview as { warnings?: string[] }).warnings ?? []).map((w, i) => (
                        <li key={i}>{w}</li>
                      ))}
                    </ul>
                  )}
              </div>

              <div className="flex flex-col gap-2 rounded border p-3">
                <Label>Pengecualian (tinggal kelas / pindah)</Label>
                <p className="text-muted-foreground text-xs">
                  Siswa pada daftar ini tidak naik: <strong>tinggal kelas</strong> tetap di grade &
                  rombel yang sama,
                  <strong> pindah</strong> ditandai transferred (bukan alumni).
                </p>
                <div className="flex flex-wrap items-center gap-2">
                  <SelectField
                    value={excUser}
                    onValueChange={setExcUser}
                    placeholder="Pilih siswa..."
                    className="w-56"
                    options={users.map((u) => ({
                      value: u.id,
                      label: `@${u.username} — ${u.name}`,
                    }))}
                  />
                  <SelectField
                    value={excAction}
                    onValueChange={(v) => setExcAction(v as "repeat" | "transfer")}
                    className="w-40"
                    options={[
                      { value: "repeat", label: "tinggal kelas" },
                      { value: "transfer", label: "pindah" },
                    ]}
                  />
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      if (!excUser) return;
                      setExceptions((prev) => [
                        ...prev.filter((e) => e.user_id !== excUser),
                        { user_id: excUser, action: excAction },
                      ]);
                      setExcUser("");
                    }}
                  >
                    Tambah
                  </Button>
                </div>
                {exceptions.map((e) => {
                  const u = users.find((x) => x.id === e.user_id);
                  return (
                    <div key={e.user_id} className="flex items-center gap-2 text-xs">
                      <span>
                        @{u?.username ?? e.user_id.slice(0, 8)} — {e.action}
                      </span>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="ml-auto"
                        onClick={() =>
                          setExceptions((prev) => prev.filter((x) => x.user_id !== e.user_id))
                        }
                      >
                        Hapus
                      </Button>
                    </div>
                  );
                })}
                {exceptions.length === 0 && (
                  <p className="text-muted-foreground text-xs">Belum ada pengecualian.</p>
                )}
              </div>

              <Button variant="outline" onClick={approve}>
                Setujui{exceptions.length > 0 ? ` (${exceptions.length} pengecualian)` : ""}
              </Button>
              <Label>Langkah 3 — Ketik tahun tujuan untuk eksekusi</Label>
              <div className="flex gap-2">
                <Input
                  value={confirm}
                  onChange={(e) => setConfirm(e.target.value)}
                  placeholder="cth. 2026"
                />
                <Button variant="destructive" onClick={execute}>
                  Eksekusi
                </Button>
              </div>
            </div>
          )}
          <div>
            <Label>Riwayat jobs</Label>
            {jobs.map((j) => (
              <p key={j.id} className="text-sm">
                {j.id.slice(0, 8)} — {j.status}
              </p>
            ))}
          </div>
        </CardContent>
      </Card>
    </main>
  );
}
