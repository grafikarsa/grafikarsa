"use client";

import { use, useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { apiGet, apiSend } from "@/lib/api";

export default function AssessmentPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [metrics, setMetrics] = useState<{ id: string; name: string }[]>([]);
  const [scores, setScores] = useState<Record<string, number>>({});

  useEffect(() => {
    apiGet<{ id: string; name: string }[]>("v1/admin/assessments/metrics")
      .then(setMetrics)
      .catch(() => null);
  }, []);

  async function submit() {
    try {
      await apiSend(`v1/admin/assessments/portfolio/${id}`, "POST", {
        scores: Object.entries(scores).map(([metric_id, score]) => ({ metric_id, score })),
      });
      toast.success("Nilai tersimpan.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  return (
    <main className="mx-auto max-w-xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Nilai portfolio</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          {metrics.map((m) => (
            <div key={m.id} className="flex items-center gap-2">
              <p className="flex-1 text-sm">{m.name}</p>
              <Input
                type="number"
                min={1}
                max={10}
                className="w-20"
                value={scores[m.id] ?? ""}
                onChange={(e) => setScores({ ...scores, [m.id]: Number(e.target.value) })}
              />
            </div>
          ))}
          <Button onClick={submit}>Simpan nilai</Button>
        </CardContent>
      </Card>
    </main>
  );
}
