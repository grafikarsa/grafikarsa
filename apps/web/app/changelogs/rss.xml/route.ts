// RSS feed changelog (publik) — dihasilkan dari API internal.
export const revalidate = 300;

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

export async function GET() {
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8080/v1";
  const appUrl = process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:3020";
  let items: { id: string; version: string; title: string; summary: string }[] = [];
  try {
    const res = await fetch(`${base}/changelogs?per_page=20`, { next: { revalidate: 300 } });
    if (res.ok) {
      const json = await res.json();
      items = json?.data ?? [];
    }
  } catch {
    items = [];
  }
  const body = items
    .map(
      (c) => `    <item>
      <title>v${esc(c.version)} — ${esc(c.title)}</title>
      <link>${appUrl}/changelog/${esc(c.version)}</link>
      <guid>${esc(c.id)}</guid>
      <description>${esc(c.summary)}</description>
    </item>`,
    )
    .join("\n");
  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Grafikarsa Changelog</title>
    <link>${appUrl}/changelog</link>
    <description>Pembaruan platform Grafikarsa</description>
${body}
  </channel>
</rss>`;
  return new Response(xml, { headers: { "content-type": "application/rss+xml; charset=utf-8" } });
}
