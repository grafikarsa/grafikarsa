"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet } from "@/lib/api";

type Stats = {
  scope: string;
  users: { total: number; active: number; alumni: number };
  portfolios: { total: number; published: number; pending: number };
  feedback_open: number;
  jobs_queued: number;
  jobs_failed: number;
  blocks_quarantined: number;
};

function Tile({ label, value, href }: { label: string; value: number; href?: string }) {
  const body = (
    <Card className="h-full">
      <CardContent className="flex flex-col gap-1 p-4">
        <span className="text-muted-foreground text-xs">{label}</span>
        <span className="text-2xl font-semibold">{value}</span>
      </CardContent>
    </Card>
  );
  return href ? (
    <Link href={href} className="block">
      {body}
    </Link>
  ) : (
    body
  );
}

export default function AdminDashboardPage() {
  const q = useQuery({
    queryKey: ["admin-dashboard"],
    queryFn: () => apiGet<Stats>("v1/admin/dashboard/stats"),
  });
  const s = q.data;

  return (
    <div className="flex flex-col gap-4">
      {/* CardTitle dirender sebagai <div>; halaman ini butuh <h1> sendiri
          supaya struktur dokumennya benar. */}
      <h1 className="text-2xl font-bold">Dashboard Admin</h1>
      <Card>
        <CardHeader>
          <CardTitle>Ringkasan</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <p className="text-muted-foreground text-sm">
            {s?.scope
              ? "Angka dibatasi ke jurusanmu (scope admin jurusan)."
              : "Angka seluruh sekolah."}
          </p>
          {q.isLoading && <Skeleton className="h-24 w-full" />}
          {q.isError && (
            <p className="text-sm">
              Gagal memuat statistik. Pastikan kamu punya permission dashboard.
            </p>
          )}
          {s && (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <Tile label="Pengguna" value={s.users.total} href="/admin/users" />
              <Tile label="Aktif" value={s.users.active} href="/admin/users" />
              <Tile label="Alumni" value={s.users.alumni} href="/admin/users" />
              <Tile label="Portfolio" value={s.portfolios.total} href="/admin/portfolios" />
              <Tile label="Terbit" value={s.portfolios.published} href="/admin/portfolios" />
              <Tile label="Menunggu review" value={s.portfolios.pending} href="/admin/moderation" />
              <Tile label="Feedback terbuka" value={s.feedback_open} href="/admin/feedback" />
              <Tile label="Job antre" value={s.jobs_queued} href="/admin/jobs" />
              <Tile label="Job gagal" value={s.jobs_failed} href="/admin/jobs" />
              <Tile label="Block karantina" value={s.blocks_quarantined} />
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Pintasan</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-2">
          <Link className={buttonVariants({ variant: "outline" })} href="/admin/moderation">
            Moderasi portfolio
          </Link>
          <Link className={buttonVariants({ variant: "outline" })} href="/admin/promotion">
            Kenaikan kelas
          </Link>
          <Link className={buttonVariants({ variant: "outline" })} href="/admin/settings">
            Pengaturan sistem
          </Link>
          <Link className={buttonVariants({ variant: "outline" })} href="/admin/audit-logs">
            Audit log
          </Link>
        </CardContent>
      </Card>
    </div>
  );
}
