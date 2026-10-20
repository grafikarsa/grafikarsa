"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { apiGet, apiSend } from "@/lib/api";

type Media = {
  id: string;
  object_key: string;
  purpose: string;
  mime: string;
  size_bytes: number;
  username?: string | null;
  created_at: string;
};

export default function MediaQuarantinePage() {
  const qc = useQueryClient();
  const list = useQuery({
    queryKey: ["media-quarantine"],
    queryFn: () => apiGet<Media[]>("v1/admin/media/quarantine?per_page=50"),
  });

  const restore = useMutation({
    mutationFn: (id: string) => apiSend(`v1/admin/media/${id}/unquarantine`, "POST"),
    onSuccess: () => {
      toast.success("Media dipulihkan.");
      qc.invalidateQueries({ queryKey: ["media-quarantine"] });
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : "Gagal: objek mungkin belum ada di storage."),
  });

  const items = Array.isArray(list.data) ? list.data : [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Media dalam karantina</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-muted-foreground text-xs">
          Media dikarantina otomatis oleh job <code>scan-media</code> ketika objeknya hilang atau
          ukurannya tidak lagi cocok. Pulihkan setelah objek dikembalikan dari backup.
        </p>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Object key</TableHead>
              <TableHead>Pemilik</TableHead>
              <TableHead>Purpose</TableHead>
              <TableHead>Ukuran</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((m) => (
              <TableRow key={m.id}>
                <TableCell className="max-w-[280px] truncate font-mono text-xs">
                  {m.object_key}
                </TableCell>
                <TableCell className="text-xs">@{m.username ?? "-"}</TableCell>
                <TableCell className="text-xs">
                  {m.purpose} · {m.mime}
                </TableCell>
                <TableCell className="text-xs">{(m.size_bytes / 1024).toFixed(1)} KB</TableCell>
                <TableCell className="text-right">
                  <Button
                    size="sm"
                    onClick={() => restore.mutate(m.id)}
                    disabled={restore.isPending}
                  >
                    Pulihkan
                  </Button>
                </TableCell>
              </TableRow>
            ))}
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
                  {list.isLoading ? "Memuat..." : "Tidak ada media dalam karantina."}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
