"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { apiGet } from "@/lib/api";

type AuditLog = {
  id: number;
  actor_id?: string | null;
  action: string;
  entity: string;
  entity_id: string;
  note?: string;
  ip?: string;
  created_at: string;
};

export default function AdminAuditLogsPage() {
  const [action, setAction] = useState("");
  const q = useQuery({
    queryKey: ["admin-audit-logs", action],
    queryFn: () =>
      apiGet<AuditLog[]>(
        `v1/admin/audit-logs?per_page=50${action ? `&action=${encodeURIComponent(action)}` : ""}`,
      ),
  });
  const items = Array.isArray(q.data) ? q.data : [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Audit log</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="grid gap-1 sm:w-72">
          <Label htmlFor="action">Filter aksi</Label>
          <Input
            id="action"
            value={action}
            onChange={(e) => setAction(e.target.value)}
            placeholder="mis. roles.assign"
          />
        </div>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Waktu</TableHead>
              <TableHead>Aksi</TableHead>
              <TableHead>Entitas</TableHead>
              <TableHead>Catatan</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((l) => (
              <TableRow key={l.id}>
                <TableCell className="text-xs">
                  {l.created_at ? new Date(l.created_at).toLocaleString("id-ID") : "—"}
                </TableCell>
                <TableCell>
                  <Badge variant="secondary">{l.action}</Badge>
                </TableCell>
                <TableCell className="font-mono text-xs">
                  {l.entity}
                  {l.entity_id ? ` · ${l.entity_id.slice(0, 8)}` : ""}
                </TableCell>
                <TableCell className="text-xs">{l.note || "—"}</TableCell>
              </TableRow>
            ))}
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="text-center text-sm text-muted-foreground">
                  {q.isLoading ? "Memuat..." : "Belum ada catatan."}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
