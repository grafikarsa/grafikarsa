"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { SelectField } from "@/components/ui/select-field";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { apiGet } from "@/lib/api";

type Row = {
  id: string;
  title: string;
  slug: string;
  username: string;
  owner_name: string;
  assessed: boolean;
  total_score: number;
};

export default function AssessmentQueuePage() {
  const [filter, setFilter] = useState("");
  const q = useQuery({
    queryKey: ["assessment-queue", filter],
    queryFn: () => apiGet<Row[]>(`v1/admin/assessments?per_page=50${filter ? `&filter=${filter}` : ""}`),
  });
  const items = Array.isArray(q.data) ? q.data : [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Antrean penilaian</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <SelectField
            value={filter}
            onValueChange={setFilter}
            placeholder="Semua karya terbit"
            className="w-52"
            options={[
              { value: "unassessed", label: "Belum dinilai" },
              { value: "assessed", label: "Sudah dinilai" },
            ]}
          />
          <p className="text-muted-foreground text-xs">
            Hanya karya yang sudah terbit. Penilaian dibatasi ke jurusanmu (admin global melihat semua).
          </p>
        </div>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Karya</TableHead>
              <TableHead>Pemilik</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Skor</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((r) => (
              <TableRow key={r.id}>
                <TableCell>
                  <Link className="hover:underline" href={`/${r.username}/${r.slug}`}>
                    {r.title}
                  </Link>
                </TableCell>
                <TableCell className="text-xs">
                  {r.owner_name} <span className="text-muted-foreground">@{r.username}</span>
                </TableCell>
                <TableCell>
                  <Badge variant={r.assessed ? "secondary" : "default"}>
                    {r.assessed ? "sudah dinilai" : "belum dinilai"}
                  </Badge>
                </TableCell>
                <TableCell>{Number(r.total_score ?? 0).toFixed(1)}</TableCell>
                <TableCell className="text-right">
                  <Link className={buttonVariants({ size: "sm", variant: "outline" })} href={`/admin/assessments/${r.id}`}>
                    {r.assessed ? "Ubah nilai" : "Nilai"}
                  </Link>
                </TableCell>
              </TableRow>
            ))}
            {items.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-sm text-muted-foreground">
                  {q.isLoading ? "Memuat..." : "Tidak ada karya pada filter ini."}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
