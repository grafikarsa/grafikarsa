"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { apiGet } from "@/lib/api";

type Entry = {
  rank: number;
  score: number;
  delta_rank: number;
  breakdown: Record<string, number>;
  subject_user_id?: string | null;
  subject_portfolio_id?: string | null;
  name?: string;
  username?: string;
  title?: string;
  slug?: string;
  owner_name?: string;
};

type Kind = "best_students" | "best_projects";
type Period = "weekly" | "monthly";

const SCOPES = [
  "global",
  "jurusan:rpl",
  "jurusan:tkj",
  "jurusan:mm",
  "jurusan:dkv",
  "jurusan:animasi",
];

function Delta({ n }: { n: number }) {
  if (!n) return <span className="text-muted-foreground text-xs">—</span>;
  return (
    <span className={n > 0 ? "text-xs text-emerald-600" : "text-xs text-red-600"}>
      {n > 0 ? `▲${n}` : `▼${Math.abs(n)}`}
    </span>
  );
}

export default function LeaderboardsPage() {
  const [kind, setKind] = useState<Kind>("best_students");
  const [period, setPeriod] = useState<Period>("weekly");
  const [scope, setScope] = useState("global");

  const q = useQuery({
    queryKey: ["leaderboard", kind, period, scope],
    queryFn: () =>
      apiGet<Entry[]>(
        `v1/leaderboards?kind=${kind}&period=${period}&scope=${encodeURIComponent(scope)}`,
      ),
  });

  const items = Array.isArray(q.data) ? q.data : [];

  return (
    <main className="mx-auto max-w-2xl p-4">
      <h1 className="mb-4 text-2xl font-bold">Peringkat</h1>

      <div className="mb-3 flex flex-wrap gap-2">
        <Tabs value={kind} onValueChange={(v) => setKind(v as Kind)}>
          <TabsList>
            <TabsTrigger value="best_students">Siswa</TabsTrigger>
            <TabsTrigger value="best_projects">Karya</TabsTrigger>
          </TabsList>
        </Tabs>
        <Tabs value={period} onValueChange={(v) => setPeriod(v as Period)}>
          <TabsList>
            <TabsTrigger value="weekly">Mingguan</TabsTrigger>
            <TabsTrigger value="monthly">Bulanan</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      <div className="mb-4 flex flex-wrap gap-1">
        {SCOPES.map((s) => (
          <button
            key={s}
            onClick={() => setScope(s)}
            className={`rounded border px-2 py-1 text-xs ${scope === s ? "bg-primary text-primary-foreground" : ""}`}
          >
            {s.replace("jurusan:", "").toUpperCase()}
          </button>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>
            {kind === "best_students" ? "Siswa terbaik" : "Karya terbaik"} · {scope}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          {q.isLoading && [0, 1, 2].map((i) => <Skeleton key={i} className="h-6 w-full" />)}
          {q.isError && <p className="text-sm">Gagal memuat peringkat.</p>}
          {!q.isLoading && items.length === 0 && (
            <p className="text-muted-foreground text-sm">
              Belum ada data untuk periode ini. Peringkat dihitung otomatis setiap hari.
            </p>
          )}
          {items.map((e) => (
            <div
              key={`${e.rank}-${e.subject_user_id ?? e.subject_portfolio_id}`}
              className="flex items-center gap-3 border-b py-2"
            >
              <span className="w-6 text-right font-semibold">{e.rank}</span>
              <Delta n={e.delta_rank} />
              <div className="flex-1">
                <p className="text-sm">
                  {e.title && e.username && e.slug ? (
                    <>
                      <Link
                        className="font-medium hover:underline"
                        href={`/${e.username}/${e.slug}`}
                      >
                        {e.title}
                      </Link>
                      <span className="text-muted-foreground"> · @{e.username}</span>
                    </>
                  ) : e.username ? (
                    <Link className="font-medium hover:underline" href={`/${e.username}`}>
                      {e.name ?? `@${e.username}`}
                    </Link>
                  ) : (
                    <span className="text-muted-foreground">#{e.rank}</span>
                  )}
                </p>
                <p className="text-muted-foreground text-xs">
                  skor {Number(e.score).toFixed(1)}
                  {e.breakdown?.portfolios !== undefined &&
                    ` · ${e.breakdown.portfolios} portfolio`}
                  {e.breakdown?.likes !== undefined && ` · ${e.breakdown.likes} suka`}
                  {e.breakdown?.assessment ? ` · nilai ${e.breakdown.assessment}` : ""}
                  {e.breakdown?.rising ? " · Rising" : ""}
                </p>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </main>
  );
}
