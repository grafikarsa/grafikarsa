import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

async function getApiHealth(): Promise<string> {
  const base = process.env.INTERNAL_API_URL ?? process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:8080/v1";
  try {
    const res = await fetch(`${base.replace(/\/v1$/, "")}/health`, { next: { revalidate: 30 } });
    if (!res.ok) return "down";
    const json = await res.json();
    return json?.data?.status ?? "ok";
  } catch {
    return "unreachable";
  }
}

export default async function Home() {
  const api = await getApiHealth();
  return (
    <main className="mx-auto flex min-h-screen max-w-2xl flex-col items-center justify-center gap-6 p-8">
      <h1 className="text-4xl font-bold tracking-tight">Grafikarsa v2</h1>
      <p className="text-muted-foreground text-center">
        Katalog portfolio warga SMKN 4 Malang. Fondasi Fase 0 — feed, portfolio, dan admin menyusul Fase 1+.
      </p>
      <Card className="w-full">
        <CardHeader>
          <CardTitle>Status layanan</CardTitle>
        </CardHeader>
        <CardContent className="flex gap-6 text-sm">
          <span>
            Web: <strong>ok</strong>
          </span>
          <span>
            API: <strong>{api}</strong>
          </span>
        </CardContent>
      </Card>
    </main>
  );
}
