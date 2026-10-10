"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { SelectField } from "@/components/ui/select-field";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { apiGet, apiSend } from "@/lib/api";

type Item = {
  id: string;
  title: string;
  slug: string;
  status: string;
  owner_name: string;
  username: string;
};

const ACTIONS: [string, string][] = [
  ["approve", "Setujui"],
  ["request-changes", "Minta perbaikan"],
  ["reject", "Tolak"],
  ["restore", "Pulihkan"],
];

export default function AdminPortfoliosPage() {
  const qc = useQueryClient();
  const [status, setStatus] = useState("");
  const [note, setNote] = useState("");

  const list = useQuery({
    queryKey: ["admin-portfolios", status],
    queryFn: () => apiGet<Item[]>(`v1/admin/portfolios${status ? `?status=${status}` : ""}`),
  });

  const act = useMutation({
    mutationFn: ({ id, action }: { id: string; action: string }) =>
      apiSend(`v1/admin/portfolios/${id}/${action}`, "POST", { note }),
    onSuccess: () => {
      toast.success("Aksi diterapkan.");
      setNote("");
      qc.invalidateQueries({ queryKey: ["admin-portfolios"] });
      qc.invalidateQueries({ queryKey: ["admin-pending"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const items = Array.isArray(list.data) ? list.data : [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Semua portfolio</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <SelectField
            value={status}
            onValueChange={setStatus}
            placeholder="Semua status"
            className="w-52"
            options={["draft", "pending_review", "published", "rejected", "changes_requested", "archived"].map((s) => ({ value: s, label: s }))}
          />
          <input
            className="rounded border px-2 py-1 text-sm"
            placeholder="Catatan moderasi (untuk tolak/perbaikan)"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
        </div>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Karya</TableHead>
              <TableHead>Pemilik</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Aksi</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((p) => (
              <TableRow key={p.id}>
                <TableCell>{p.title}</TableCell>
                <TableCell className="text-xs">@{p.username}</TableCell>
                <TableCell>
                  <Badge variant={p.status === "published" ? "default" : "secondary"}>{p.status}</Badge>
                </TableCell>
                <TableCell className="flex flex-wrap gap-1">
                  {ACTIONS.filter(([a]) => {
                    if (p.status === "published") return a === "reject" || a === "restore";
                    if (p.status === "rejected") return a === "restore" || a === "approve";
                    return a !== "restore";
                  }).map(([a, label]) => (
                    <Button
                      key={a}
                      size="sm"
                      variant={a === "reject" ? "destructive" : "outline"}
                      disabled={(a === "reject" || a === "request-changes") && note.trim().length < 5}
                      onClick={() => act.mutate({ id: p.id, action: a })}
                    >
                      {label}
                    </Button>
                  ))}
                </TableCell>
              </TableRow>
            ))}
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} className="text-center text-sm text-muted-foreground">
                  {list.isLoading ? "Memuat..." : "Tidak ada portfolio."}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
