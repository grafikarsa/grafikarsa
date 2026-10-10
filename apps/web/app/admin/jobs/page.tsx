"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { SelectField } from "@/components/ui/select-field";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { apiGet } from "@/lib/api";

type Job = {
  id: number;
  topic: string;
  status: string;
  attempts: number;
  next_retry_at: string | null;
  created_at: string;
};

const STATUSES = ["", "pending", "processing", "sent", "failed"];

export default function AdminJobsPage() {
  const [status, setStatus] = useState("");
  const q = useQuery({
    queryKey: ["admin-jobs", status],
    queryFn: () => apiGet<Job[]>(`v1/admin/jobs?per_page=50${status ? `&status=${status}` : ""}`),
    refetchInterval: 30_000,
  });
  const items = Array.isArray(q.data) ? q.data : [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Job queue (outbox)</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex items-center gap-2">
          <SelectField
            value={status}
            onValueChange={setStatus}
            placeholder="Semua status"
            className="w-44"
            options={STATUSES.filter(Boolean).map((s) => ({ value: s, label: s }))}
          />
          <p className="text-muted-foreground text-xs">
            Disegarkan otomatis tiap 30 detik. Status <code>processing</code> yang tertinggal akan
            dikembalikan ke <code>pending</code> oleh reaper.
          </p>
        </div>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Topic</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Attempts</TableHead>
              <TableHead>Dibuat</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((j) => (
              <TableRow key={j.id}>
                <TableCell className="font-mono text-xs">{j.id}</TableCell>
                <TableCell className="font-mono text-xs">{j.topic}</TableCell>
                <TableCell>
                  <Badge variant={j.status === "failed" ? "destructive" : j.status === "sent" ? "secondary" : "default"}>
                    {j.status}
                  </Badge>
                </TableCell>
                <TableCell>{j.attempts}</TableCell>
                <TableCell className="text-xs">
                  {j.created_at ? new Date(j.created_at).toLocaleString("id-ID") : "—"}
                </TableCell>
              </TableRow>
            ))}
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
                  {q.isLoading ? "Memuat..." : "Tidak ada job."}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
