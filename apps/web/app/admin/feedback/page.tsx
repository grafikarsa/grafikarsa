"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { SelectField } from "@/components/ui/select-field";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { apiGet, apiSend } from "@/lib/api";

type Thread = { id: string; category: string; title: string; status: string; upvotes: number; priority?: string };
type Detail = {
  id: string;
  title: string;
  body: string;
  status: string;
  priority: string;
  sla_hours: number;
  age_hours: number;
  sla_breached: boolean;
  messages: { body: string; is_internal: boolean }[];
};

const STATUSES = ["open", "triaged", "in_progress", "resolved", "closed", "duplicate"];
const PRIORITIES = ["low", "medium", "high", "critical"];

export default function AdminFeedbackPage() {
  const qc = useQueryClient();
  const [status, setStatus] = useState("");
  const [selected, setSelected] = useState<string>("");
  const [reply, setReply] = useState("");
  const [nextStatus, setNextStatus] = useState("");
  const [nextPriority, setNextPriority] = useState("");
  const [internal, setInternal] = useState(false);

  const list = useQuery({
    queryKey: ["admin-feedback", status],
    queryFn: () => apiGet<Thread[]>(`v1/admin/feedback${status ? `?status=${status}` : ""}`),
  });

  const detail = useQuery({
    queryKey: ["admin-feedback-detail", selected],
    queryFn: () => apiGet<Detail>(`v1/admin/feedback/${selected}`),
    enabled: Boolean(selected),
  });

  const send = useMutation({
    mutationFn: () =>
      apiSend(`v1/admin/feedback/${selected}/reply`, "POST", {
        body: reply,
        is_internal: internal,
        status: nextStatus,
      }),
    onSuccess: () => {
      toast.success("Balasan terkirim.");
      setReply("");
      qc.invalidateQueries({ queryKey: ["admin-feedback-detail", selected] });
      qc.invalidateQueries({ queryKey: ["admin-feedback"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const triage = useMutation({
    mutationFn: () =>
      apiSend(`v1/admin/feedback/${selected}/triage`, "PATCH", {
        priority: nextPriority || undefined,
      }),
    onSuccess: () => {
      toast.success("Triage tersimpan.");
      qc.invalidateQueries({ queryKey: ["admin-feedback-detail", selected] });
      qc.invalidateQueries({ queryKey: ["admin-feedback"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const items = Array.isArray(list.data) ? list.data : [];

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Feedback</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="flex items-end gap-2">
            <SelectField
              value={status}
              onValueChange={setStatus}
              placeholder="Semua status"
              className="w-44"
              options={STATUSES.map((s) => ({ value: s, label: s }))}
            />
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Judul</TableHead>
                <TableHead>Kategori</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Vote</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((t) => (
                <TableRow key={t.id}>
                  <TableCell>{t.title}</TableCell>
                  <TableCell>{t.category}</TableCell>
                  <TableCell>
                    <Badge variant={t.status === "resolved" ? "secondary" : "default"}>{t.status}</Badge>
                  </TableCell>
                  <TableCell>{t.upvotes}</TableCell>
                  <TableCell className="text-right">
                    <Button variant="outline" size="sm" onClick={() => setSelected(t.id)}>
                      Buka
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
              {items.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
                    {list.isLoading ? "Memuat..." : "Belum ada feedback."}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {selected && detail.data && (
        <Card>
          <CardHeader>
            <CardTitle>
              {detail.data.title}{" "}
              {detail.data.sla_breached && <Badge variant="destructive">Lewat SLA</Badge>}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <p className="text-sm">{detail.data.body}</p>
            <p className="text-muted-foreground text-xs">
              Status {detail.data.status} · prioritas {detail.data.priority} · umur {Math.round(detail.data.age_hours)} jam ·
              target {detail.data.sla_hours} jam
            </p>

            <div className="flex flex-wrap gap-2">
              <SelectField
                value={nextPriority}
                onValueChange={setNextPriority}
                placeholder="Ubah prioritas..."
                className="w-48"
                options={PRIORITIES.map((p) => ({ value: p, label: p }))}
              />
              <Button
                variant="outline"
                size="sm"
                onClick={() => triage.mutate()}
                disabled={!nextPriority || triage.isPending}
              >
                Simpan prioritas
              </Button>
            </div>

            <div className="flex flex-col gap-2 rounded border p-3">
              {(detail.data.messages ?? []).map((m, i) => (
                <p key={i} className={`text-sm ${m.is_internal ? "italic text-amber-600" : ""}`}>
                  {m.is_internal ? "[internal] " : ""}
                  {m.body}
                </p>
              ))}
            </div>

            <Textarea value={reply} onChange={(e) => setReply(e.target.value)} placeholder="Balasan..." />
            <div className="flex flex-wrap items-center gap-2">
              <SelectField
                value={nextStatus}
                onValueChange={setNextStatus}
                placeholder="(status tetap)"
                className="w-44"
                options={STATUSES.map((s) => ({ value: s, label: `→ ${s}` }))}
              />
              <label className="flex items-center gap-1 text-xs">
                <input type="checkbox" checked={internal} onChange={(e) => setInternal(e.target.checked)} />
                catatan internal
              </label>
              <Button onClick={() => send.mutate()} disabled={reply.length < 2 || send.isPending}>
                Kirim balasan
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
