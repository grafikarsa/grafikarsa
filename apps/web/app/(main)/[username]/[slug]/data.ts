import type { PortfolioDetail } from "@/lib/portfolio-types";

/**
 * Ambil data karya untuk SSR (metadata + JSON-LD).
 *
 * Dipisah dari komponen client supaya tidak jadi bundel browser.
 */
export async function getPortfolio(
  username: string,
  slug: string,
): Promise<PortfolioDetail | null> {
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8091/v1";
  const res = await fetch(`${base}/portfolios/by-slug?username=${username}&slug=${slug}`, {
    next: { revalidate: 60 },
  });
  if (!res.ok) return null;
  const json = await res.json();
  return (json.data as PortfolioDetail) ?? null;
}

/**
 * Bangun JSON-LD schema.org/CreativeWork untuk sebuah karya.
 *
 * PRD §14.3: karya perlu JSON-LD `CreativeWork` supaya mesin pencari bisa
 * membaca karya sebagai karya (bukan sekadar halaman web), lengkap dengan
 * penulis, tanggal terbit, dan interaksi.
 */
export function creativeWorkJsonLd(
  p: PortfolioDetail,
  username: string,
  siteUrl: string,
): Record<string, unknown> {
  const url = `${siteUrl}/${username}/${p.slug}`;
  const ld: Record<string, unknown> = {
    "@context": "https://schema.org",
    "@type": "CreativeWork",
    name: p.title,
    url,
    author: {
      "@type": "Person",
      name: p.author_name ?? username,
      url: `${siteUrl}/${username}`,
    },
    publisher: {
      "@type": "Organization",
      name: "Grafikarsa",
      url: siteUrl,
    },
  };
  if (p.description) ld.description = p.description;
  if (p.published_at) ld.datePublished = p.published_at;
  if (p.updated_at) ld.dateModified = p.updated_at;
  if (p.like_count !== undefined) {
    ld.interactionStatistic = {
      "@type": "InteractionCounter",
      interactionType: "https://schema.org/LikeAction",
      userInteractionCount: p.like_count,
    };
  }
  if (p.comment_count !== undefined) {
    ld.commentCount = p.comment_count;
  }
  return ld;
}

/**
 * Ambil data karya untuk OG image.
 *
 * Sengaja terpisah dari getPortfolio: halaman memakai cache 60 detik,
 * sedangkan OG image tidak boleh ikut mewarisi cache halaman.
 */
export async function getPortfolioForOg(
  username: string,
  slug: string,
): Promise<PortfolioDetail | null> {
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8091/v1";
  try {
    const res = await fetch(`${base}/portfolios/by-slug?username=${username}&slug=${slug}`, {
      cache: "no-store",
    });
    if (!res.ok) return null;
    const json = await res.json();
    return (json.data as PortfolioDetail) ?? null;
  } catch {
    return null;
  }
}
