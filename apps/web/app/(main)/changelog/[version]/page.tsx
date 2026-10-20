import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type Section = { kind: string; title: string; body_md: string };
type Detail = {
  id: string;
  version: string;
  title: string;
  summary: string;
  status: string;
  sections: Section[];
};

const KIND_LABEL: Record<string, string> = {
  added: "Baru",
  improved: "Peningkatan",
  fixed: "Perbaikan",
  removed: "Dihapus",
};

// Halaman SEO per rilis: /changelog/{version} (PRD §11.2).
export async function generateMetadata({
  params,
}: {
  params: Promise<{ version: string }>;
}): Promise<Metadata> {
  const { version } = await params;
  const data = await fetchDetail(version);
  if (!data) return { title: "Changelog — Grafikarsa" };
  return {
    title: `Grafikarsa v${data.version} — ${data.title}`,
    description: data.summary || undefined,
    alternates: { types: { "application/rss+xml": "/changelogs/rss.xml" } },
  };
}

async function fetchDetail(version: string): Promise<Detail | null> {
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8080/v1";
  try {
    const res = await fetch(`${base}/changelogs/version/${encodeURIComponent(version)}`, {
      next: { revalidate: 300 },
    });
    if (!res.ok) return null;
    const json = await res.json();
    return (json?.data ?? null) as Detail | null;
  } catch {
    return null;
  }
}

export default async function ChangelogVersionPage({
  params,
}: {
  params: Promise<{ version: string }>;
}) {
  const { version } = await params;
  const data = await fetchDetail(version);
  if (!data) notFound();

  const jsonLd = {
    "@context": "https://schema.org",
    "@type": "TechArticle",
    headline: `Grafikarsa v${data.version} — ${data.title}`,
    description: data.summary,
    datePublished: undefined,
  };

  return (
    <main className="mx-auto flex max-w-2xl flex-col gap-4 p-4">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />
      <Link href="/changelog" className="text-sm underline">
        ← Semua changelog
      </Link>
      <Card>
        <CardHeader>
          <CardTitle>
            v{data.version} — {data.title}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {data.summary && <p className="text-sm">{data.summary}</p>}
          {(data.sections ?? []).map((s, i) => (
            <div key={i} className="text-sm">
              <Badge variant="secondary" className="mr-2">
                {KIND_LABEL[s.kind] ?? s.kind}
              </Badge>
              <span className="font-medium">{s.title}</span>
              {s.body_md && (
                <p className="text-muted-foreground mt-1 whitespace-pre-wrap">{s.body_md}</p>
              )}
            </div>
          ))}
          {(data.sections ?? []).length === 0 && (
            <p className="text-muted-foreground text-sm">Rilis ini belum punya detail section.</p>
          )}
        </CardContent>
      </Card>
      <a href="/changelogs/rss.xml" className="text-sm underline">
        RSS
      </a>
    </main>
  );
}
