"use client";

import { useQuery } from "@tanstack/react-query";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiSend } from "@/lib/api";
import { useEffect, useState } from "react";
import { toast } from "sonner";

type Settings = {
  feed_weights?: Record<string, number>;
  upload_limits?: Record<string, number>;
};

export default function AdminSettingsPage() {
  const q = useQuery({ queryKey: ["admin-settings"], queryFn: () => apiGet<Settings>("v1/admin/settings") });
  const [weights, setWeights] = useState<Record<string, string>>({});
  const [limits, setLimits] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!q.data) return;
    const w: Record<string, string> = {};
    for (const [k, v] of Object.entries(q.data.feed_weights ?? {})) w[k] = String(v);
    setWeights(w);
    const l: Record<string, string> = {};
    for (const [k, v] of Object.entries(q.data.upload_limits ?? {})) l[k] = String(v);
    setLimits(l);
  }, [q.data]);

  const total = Object.values(weights).reduce((a, v) => a + (Number(v) || 0), 0);

  async function save(key: "feed_weights" | "upload_limits") {
    setBusy(true);
    try {
      const src = key === "feed_weights" ? weights : limits;
      const value: Record<string, number> = {};
      for (const [k, v] of Object.entries(src)) value[k] = Number(v);
      await apiSend(`v1/admin/settings/${key}`, "PATCH", { value });
      toast.success("Pengaturan disimpan.");
      q.refetch();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menyimpan.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Bobot algoritma feed</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {q.isLoading && <Skeleton className="h-32 w-full" />}
          <p className="text-muted-foreground text-xs">
            Total bobot harus 1.0 (server menolak selain itu). Perubahan langsung dipakai feed
            smart tanpa deploy.
          </p>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {Object.keys(weights).map((k) => (
              <div key={k} className="grid gap-1">
                <Label htmlFor={`w-${k}`}>{k}</Label>
                <Input
                  id={`w-${k}`}
                  type="number"
                  step="0.01"
                  min="0"
                  max="1"
                  value={weights[k]}
                  onChange={(e) => setWeights({ ...weights, [k]: e.target.value })}
                />
              </div>
            ))}
          </div>
          <div className="flex items-center gap-3">
            <Badge variant={Math.abs(total - 1) < 0.011 ? "secondary" : "destructive"}>
              total {total.toFixed(2)}
            </Badge>
            <Button onClick={() => save("feed_weights")} disabled={busy || Math.abs(total - 1) >= 0.011}>
              Simpan bobot
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Batas ukuran upload (MB)</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-muted-foreground text-xs">
            Dibatasi keras 52MB oleh server (batas body). Nilai dipakai langsung oleh endpoint
            upload.
          </p>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {Object.keys(limits).map((k) => (
              <div key={k} className="grid gap-1">
                <Label htmlFor={`l-${k}`}>{k}</Label>
                <Input
                  id={`l-${k}`}
                  type="number"
                  step="0.5"
                  min="0"
                  value={limits[k]}
                  onChange={(e) => setLimits({ ...limits, [k]: e.target.value })}
                />
              </div>
            ))}
          </div>
          <Button className="self-start" onClick={() => save("upload_limits")} disabled={busy}>
            Simpan batas upload
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
