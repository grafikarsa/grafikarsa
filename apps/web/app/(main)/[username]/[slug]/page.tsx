import type { Metadata } from "next";
import { PortfolioDetail } from "@/components/portfolio/portfolio-detail";
import { creativeWorkJsonLd, getPortfolio } from "./data";
import { jsonLdScript } from "@/lib/json-ld";

const SITE_URL = process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:3000";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ username: string; slug: string }>;
}): Promise<Metadata> {
  const { username, slug } = await params;
  const p = await getPortfolio(username, slug);
  if (!p) return { title: "Tidak ditemukan — Grafikarsa" };

  const description = `Karya ${p.author_name ?? username} di Grafikarsa.`;

  // Gambar OG TIDAK ditulis manual: file opengraph-image.tsx di segmen ini
  // otomatis dipakai Next.js untuk openGraph.images dan twitter.images.
  // Kalau images diisi manual di sini, yang otomatis justru dikalahkan.
  return {
    title: `${p.title} — Grafikarsa`,
    description,
    openGraph: {
      type: "article",
      title: p.title,
      description,
      url: `${SITE_URL}/${username}/${slug}`,
      siteName: "Grafikarsa",
    },
    twitter: {
      card: "summary_large_image",
      title: p.title,
      description,
    },
  };
}

export default async function PortfolioPage({
  params,
}: {
  params: Promise<{ username: string; slug: string }>;
}) {
  const { username, slug } = await params;
  const p = await getPortfolio(username, slug);
  if (!p) return <main className="p-8">Portfolio tidak ditemukan.</main>;

  // JSON-LD CreativeWork (PRD §14.3).
  const jsonLd = creativeWorkJsonLd(p, username, SITE_URL);

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: jsonLdScript(jsonLd) }}
      />
      <PortfolioDetail initial={p} />
    </>
  );
}
