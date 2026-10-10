import type { Metadata } from "next";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export const metadata: Metadata = {
  title: "Grafikarsa — Katalog Portfolio SMKN 4 Malang",
  description: "Jelajahi karya siswa dan alumni SMKN 4 Malang.",
};

type LeaderItem = { rank: number; score: number; subject_user_id?: string; subject_portfolio_id?: string };

async function getTop(): Promise<{ students: LeaderItem[]; projects: LeaderItem[] }> {
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8091/v1";
  async function one(kind: string): Promise<LeaderItem[]> {
    try {
      const res = await fetch(`${base}/leaderboards?period=weekly&scope=global&kind=${kind}`, { next: { revalidate: 3600 } });
      if (!res.ok) return [];
      const json = await res.json();
      return json?.data?.slice(0, 6) ?? [];
    } catch {
      return [];
    }
  }
  const [students, projects] = await Promise.all([one("best_students"), one("best_projects")]);
  return { students, projects };
}

export default async function LandingPage() {
  const { students, projects } = await getTop();
  return (
    <main className="mx-auto flex max-w-3xl flex-col items-center gap-8 p-8">
      <div className="text-center">
        <h1 className="text-4xl font-bold tracking-tight">Grafikarsa</h1>
        <p className="text-muted-foreground mt-2">Katalog portfolio warga SMKN 4 Malang.</p>
        <div className="mt-4 flex justify-center gap-2">
          <Link href="/login">
            <Button>Masuk</Button>
          </Link>
          <Link href="/portfolios">
            <Button variant="outline">Jelajahi karya</Button>
          </Link>
        </div>
      </div>
      <div className="grid w-full gap-4 sm:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Siswa terbaik minggu ini</CardTitle>
          </CardHeader>
          <CardContent>
            {students.length === 0 && <p className="text-muted-foreground text-sm">Belum ada data.</p>}
            {students.map((s) => (
              <p key={s.rank} className="text-sm">
                #{s.rank} — skor {Number(s.score).toFixed(1)}
              </p>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Karya terbaik minggu ini</CardTitle>
          </CardHeader>
          <CardContent>
            {projects.length === 0 && <p className="text-muted-foreground text-sm">Belum ada data.</p>}
            {projects.map((s) => (
              <p key={s.rank} className="text-sm">
                #{s.rank} — skor {Number(s.score).toFixed(1)}
              </p>
            ))}
          </CardContent>
        </Card>
      </div>
    </main>
  );
}
