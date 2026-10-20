import type { MetadataRoute } from "next";

// Sitemap dinamis: halaman statis + portfolio published + profil user.
// Angka dibatasi (500 terbaru) supaya build tetap cepat dan tidak membebani API.
export const revalidate = 3600;

type Item = { username: string; slug?: string };

async function api<T>(path: string): Promise<T | null> {
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8080/v1";
  try {
    const res = await fetch(`${base}${path}`, { next: { revalidate: 3600 } });
    if (!res.ok) return null;
    const json = await res.json();
    return (json?.data ?? null) as T;
  } catch {
    return null;
  }
}

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const appUrl = process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:3020";
  const staticRoutes = [
    "",
    "/feed",
    "/portfolios",
    "/leaderboards",
    "/users",
    "/changelog",
    "/feedback",
    "/search",
  ];
  const entries: MetadataRoute.Sitemap = staticRoutes.map((path) => ({
    url: `${appUrl}${path}`,
    lastModified: new Date(),
    changeFrequency: path === "" || path === "/feed" ? "daily" : "weekly",
    priority: path === "" ? 1 : 0.7,
  }));

  const [works, users, changelogs] = await Promise.all([
    api<Item[]>("/portfolios?per_page=500&sort=latest"),
    api<Item[]>("/users?per_page=200"),
    api<{ version: string }[]>("/changelogs?per_page=100"),
  ]);

  for (const w of works ?? []) {
    if (w.username && w.slug) {
      entries.push({
        url: `${appUrl}/${w.username}/${w.slug}`,
        changeFrequency: "monthly",
        priority: 0.6,
      });
    }
  }
  for (const u of users ?? []) {
    if (u.username)
      entries.push({ url: `${appUrl}/${u.username}`, changeFrequency: "weekly", priority: 0.5 });
  }
  for (const c of changelogs ?? []) {
    if (c.version)
      entries.push({
        url: `${appUrl}/changelog/${c.version}`,
        changeFrequency: "monthly",
        priority: 0.5,
      });
  }

  return entries;
}
