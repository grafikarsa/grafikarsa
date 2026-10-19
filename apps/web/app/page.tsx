import type { Metadata } from "next";
import Link from "next/link";
import { Badge } from "@/components/ui/badge";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export const metadata: Metadata = {
  title: "Grafikarsa — Katalog Portfolio SMKN 4 Malang",
  description:
    "Katalog portfolio siswa dan alumni SMKN 4 Malang: karya terbaik, peringkat mingguan, dan changelog platform.",
  openGraph: {
    title: "Grafikarsa",
    description: "Katalog portfolio warga SMKN 4 Malang.",
    type: "website",
  },
};

type LeaderItem = {
  rank: number;
  score: number;
  name?: string;
  username?: string;
  title?: string;
  slug?: string;
  owner_name?: string;
  subject_user_id?: string;
  subject_portfolio_id?: string;
};

type Changelog = { id: string; version: string; title: string; summary: string };

async function api<T>(path: string): Promise<T | null> {
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8080/v1";
  try {
    const res = await fetch(`${base}${path}`, { next: { revalidate: 600 } });
    if (!res.ok) return null;
    const json = await res.json();
    return (json?.data ?? null) as T;
  } catch {
    return null;
  }
}

export default async function LandingPage() {
  const [studentsRaw, projectsRaw, changelogs] = await Promise.all([
    api<LeaderItem[]>("/leaderboards?period=weekly&scope=global&kind=best_students"),
    api<LeaderItem[]>("/leaderboards?period=weekly&scope=global&kind=best_projects"),
    api<Changelog[]>("/changelogs?per_page=3"),
  ]);
  const students = (studentsRaw ?? []).slice(0, 5);
  const projects = (projectsRaw ?? []).slice(0, 5);
  const releases = (changelogs ?? []).slice(0, 3);

  const jsonLd = {
    "@context": "https://schema.org",
    "@type": "WebSite",
    name: "Grafikarsa",
    about: "Katalog portfolio SMKN 4 Malang",
    publisher: { "@type": "Organization", name: "SMKN 4 Malang" },
  };

  const faq = [
    {
      q: "Siapa yang bisa punya portfolio di Grafikarsa?",
      a: "Siswa dan alumni SMKN 4 Malang. Akun dibuat oleh admin sekolah, lalu kamu masuk dan mengubah password.",
    },
    {
      q: "Bagaimana karya saya dinilai?",
      a: "Guru pembimbing memberi penilaian per aspek. Portfolio yang lolos moderasi masuk katalog, dan peringkat dihitung dari kualitas, interaksi, dan tren.",
    },
    {
      q: "Apakah karya saya bisa dilihat publik?",
      a: "Ya, setelah kamu kirim untuk review dan moderator menyetujuinya. Draft dan karya yang ditolak hanya terlihat olehmu.",
    },
  ];

  return (
    <main className="mx-auto flex max-w-3xl flex-col gap-8 p-8">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />

      <section className="text-center">
        <h1 className="text-4xl font-bold tracking-tight">Grafikarsa</h1>
        <p className="text-muted-foreground mt-2">Katalog portfolio warga SMKN 4 Malang.</p>
        <div className="mt-4 flex flex-wrap justify-center gap-2">
          <Link className={buttonVariants()} href="/feed">
            Lihat feed
          </Link>
          <Link className={buttonVariants({ variant: "outline" })} href="/portfolios">
            Jelajahi karya
          </Link>
          <Link className={buttonVariants({ variant: "ghost" })} href="/leaderboards">
            Peringkat
          </Link>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Siswa terbaik minggu ini</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-1">
            {students.length === 0 && (
              <p className="text-muted-foreground text-sm">Belum ada data.</p>
            )}
            {students.map((s) => (
              <p key={s.rank} className="text-sm">
                <span className="mr-2 font-semibold">#{s.rank}</span>
                {s.username ? (
                  <Link className="hover:underline" href={`/${s.username}`}>
                    {s.name ?? `@${s.username}`}
                  </Link>
                ) : (
                  <span className="text-muted-foreground">(data belum lengkap)</span>
                )}
                <span className="text-muted-foreground ml-2 text-xs">
                  skor {Number(s.score).toFixed(1)}
                </span>
              </p>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Karya terbaik minggu ini</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-1">
            {projects.length === 0 && (
              <p className="text-muted-foreground text-sm">Belum ada data.</p>
            )}
            {projects.map((p) => (
              <p key={p.rank} className="text-sm">
                <span className="mr-2 font-semibold">#{p.rank}</span>
                {p.username && p.slug ? (
                  <Link className="hover:underline" href={`/${p.username}/${p.slug}`}>
                    {p.title}
                  </Link>
                ) : (
                  <span className="text-muted-foreground">(data belum lengkap)</span>
                )}
                <span className="text-muted-foreground ml-2 text-xs">
                  skor {Number(p.score).toFixed(1)}
                </span>
              </p>
            ))}
          </CardContent>
        </Card>
      </section>

      {releases.length > 0 && (
        <section>
          <h2 className="mb-2 text-lg font-semibold">Terbaru di platform</h2>
          <div className="flex flex-col gap-2">
            {releases.map((c) => (
              <div key={c.id} className="flex items-start gap-2">
                <Badge variant="secondary">v{c.version}</Badge>
                <div>
                  <p className="text-sm font-medium">{c.title}</p>
                  <p className="text-muted-foreground text-xs">{c.summary}</p>
                </div>
              </div>
            ))}
          </div>
          <Link href="/changelog" className="mt-2 inline-block text-sm underline">
            Semua changelog
          </Link>
        </section>
      )}

      <section>
        <h2 className="mb-2 text-lg font-semibold">Pertanyaan umum</h2>
        <div className="flex flex-col gap-3">
          {faq.map((f) => (
            <div key={f.q}>
              <p className="text-sm font-medium">{f.q}</p>
              <p className="text-muted-foreground text-sm">{f.a}</p>
            </div>
          ))}
        </div>
      </section>

      <footer className="text-muted-foreground flex flex-wrap gap-3 border-t pt-4 text-xs">
        <Link href="/feed" className="hover:underline">
          Feed
        </Link>
        <Link href="/leaderboards" className="hover:underline">
          Peringkat
        </Link>
        <Link href="/portfolios" className="hover:underline">
          Karya
        </Link>
        <Link href="/users" className="hover:underline">
          Warga
        </Link>
        <Link href="/changelog" className="hover:underline">
          Changelog
        </Link>
        <Link href="/feedback" className="hover:underline">
          Masukan
        </Link>
        <a href="/changelogs/rss.xml" className="hover:underline">
          RSS
        </a>
      </footer>
    </main>
  );
}
