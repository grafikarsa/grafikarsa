// Notifikasi: pemetaan payload ke halaman tujuan.
//
// PRD §10: notifikasi harus bisa diklik menuju konteksnya.

export type Notif = {
  id: string;
  type: string;
  title: string;
  body: string;
  created_at: string;
  is_read: boolean;
  data?: Record<string, unknown> | null;
};

/**
 * deepLink memetakan payload `data` notifikasi ke halaman yang relevan.
 *
 * Fungsi ini TIDAK diekspor dari page component: Next.js hanya mengizinkan
 * export tertentu (default, metadata, generateMetadata, dll). Export
 * `deepLink` dari sebuah page membuat build webpack gagal dengan
 * "deepLink is not a valid Page export field". Karena itu ia ditaruh di
 * lib/ supaya tetap bisa diuji (lihat notifications.test.ts).
 */
export function deepLink(n: Notif): string | null {
  const d = (n.data ?? {}) as Record<string, unknown>;
  const portfolioID = typeof d.portfolio_id === "string" ? d.portfolio_id : "";
  const username = typeof d.username === "string" ? d.username : "";
  const slug = typeof d.slug === "string" ? d.slug : "";
  const threadID = typeof d.thread_id === "string" ? d.thread_id : "";
  const version = typeof d.version === "string" ? d.version : "";

  if (username && slug) return `/${username}/${slug}`;
  if (portfolioID) return `/portfolios?highlight=${portfolioID}`;
  if (threadID) return `/feedback?thread=${threadID}`;
  if (version) return `/changelog/${version}`;
  return null;
}
