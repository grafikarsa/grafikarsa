import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type CL = { id: string; version: string; title: string; summary: string };

async function getChangelogs(): Promise<CL[]> {
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8091/v1";
  try {
    const res = await fetch(`${base}/changelogs`, { next: { revalidate: 300 } });
    if (!res.ok) return [];
    const json = await res.json();
    return json?.data ?? [];
  } catch {
    return [];
  }
}

export const metadata = { title: "Changelog — Grafikarsa" };

export default async function ChangelogPage() {
  const items = await getChangelogs();
  return (
    <main className="mx-auto max-w-2xl p-4">
      <h1 className="mb-4 text-2xl font-bold">Changelog</h1>
      {items.length === 0 && <p className="text-muted-foreground">Belum ada rilis.</p>}
      {items.map((c) => (
        <Card key={c.id} className="mb-3">
          <CardHeader>
            <CardTitle>v{c.version} — {c.title}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm">{c.summary}</p>
          </CardContent>
        </Card>
      ))}
    </main>
  );
}
