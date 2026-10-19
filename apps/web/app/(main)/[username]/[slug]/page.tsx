import type { Metadata } from "next";
import { PortfolioDetail } from "@/components/portfolio/portfolio-detail";

async function getPortfolio(username: string, slug: string) {
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8091/v1";
  const res = await fetch(`${base}/portfolios/by-slug?username=${username}&slug=${slug}`, { next: { revalidate: 60 } });
  if (!res.ok) return null;
  const json = await res.json();
  return json.data;
}

export async function generateMetadata({ params }: { params: Promise<{ username: string; slug: string }> }): Promise<Metadata> {
  const { username, slug } = await params;
  const p = await getPortfolio(username, slug);
  if (!p) return { title: "Tidak ditemukan — Grafikarsa" };
  return { title: `${p.title} — Grafikarsa`, description: `Karya ${username} di Grafikarsa.` };
}

export default async function PortfolioPage({ params }: { params: Promise<{ username: string; slug: string }> }) {
  const { username, slug } = await params;
  const p = await getPortfolio(username, slug);
  if (!p) return <main className="p-8">Portfolio tidak ditemukan.</main>;
  return <PortfolioDetail initial={p} />;
}
