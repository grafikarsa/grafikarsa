"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { apiGet, apiSend } from "@/lib/api";

type Notif = {
  id: string;
  type: string;
  title: string;
  body: string;
  created_at: string;
  is_read: boolean;
  data?: Record<string, unknown> | null;
};

// deepLink memetakan payload `data` notifikasi ke halaman yang relevan
// (PRD §10: notifikasi harus bisa diklik menuju konteksnya).
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

export default function NotificationsPage() {
  const [items, setItems] = useState<Notif[]>([]);
  const [loading, setLoading] = useState(true);

  async function load() {
    try {
      const data = await apiGet<Notif[]>("v1/notifications");
      setItems(Array.isArray(data) ? data : []);
      return true;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat.");
      throw err;
    } finally {
      setLoading(false);
    }
  }

  // Load dulu; tandai terbaca HANYA bila pemuatan berhasil (jangan
  // menghilangkan badge saat request gagal).
  useEffect(() => {
    load()
      .then(() => apiSend("v1/notifications/read-all", "POST").catch(() => null))
      .catch(() => null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function markRead(id: string) {
    try {
      await apiSend(`v1/notifications/${id}/read`, "PATCH");
      setItems((prev) => prev.map((n) => (n.id === id ? { ...n, is_read: true } : n)));
    } catch {
      /* badge akan tersinkron saat halaman dimuat ulang */
    }
  }

  return (
    <main className="mx-auto max-w-2xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Notifikasi</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          {loading && <p className="text-muted-foreground text-sm">Memuat...</p>}
          {!loading && items.length === 0 && <p className="text-muted-foreground text-sm">Tidak ada notifikasi.</p>}
          {items.map((n) => {
            const href = deepLink(n);
            return (
              <div key={n.id} className="flex flex-col gap-1 border-b py-2">
                <div className="flex items-center gap-2">
                  {!n.is_read && (
                    <span className="bg-primary inline-block size-2 rounded-full" aria-label="Belum dibaca" />
                  )}
                  <p className="font-semibold">{n.title}</p>
                  <Badge variant="secondary">{n.type}</Badge>
                </div>
                <p className="text-sm">{n.body}</p>
                <div className="flex items-center gap-2">
                  {href && (
                    <Link
                      href={href}
                      className="text-sm underline"
                      onClick={() => {
                        if (!n.is_read) markRead(n.id).catch(() => null);
                      }}
                    >
                      Buka
                    </Link>
                  )}
                  {!n.is_read && (
                    <Button variant="ghost" size="sm" onClick={() => markRead(n.id)}>
                      Tandai dibaca
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={async () => {
                      await apiSend(`v1/notifications/${n.id}`, "DELETE");
                      load().catch(() => null);
                    }}
                  >
                    Hapus
                  </Button>
                  <span className="text-muted-foreground ml-auto text-xs">
                    {n.created_at ? new Date(n.created_at).toLocaleString("id-ID") : ""}
                  </span>
                </div>
              </div>
            );
          })}
        </CardContent>
      </Card>
    </main>
  );
}
