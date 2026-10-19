"use client";

import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

type Section = { kind: string; title: string; body_md: string };
type Item = { id: string; version: string; title: string; summary: string };

const KIND_LABEL: Record<string, string> = {
  added: "Baru",
  improved: "Peningkatan",
  fixed: "Perbaikan",
  removed: "Dihapus",
};

export default function ChangelogPage() {
  const user = useAuthStore((s) => s.user);
  const [detailed, setDetailed] = useState<Record<string, Section[]>>({});

  const list = useQuery({
    queryKey: ["changelogs"],
    queryFn: () => apiGet<Item[]>("v1/changelogs"),
  });

  const unread = useQuery({
    queryKey: ["changelogs-unread"],
    queryFn: () => apiGet<{ unread: number }>("v1/changelogs/unread-count"),
    enabled: Boolean(user),
  });

  // Tandai sudah dibaca saat halaman dibuka (sekali per kunjungan).
  useEffect(() => {
    if (!user) return;
    apiSend("v1/changelogs/mark-all-read", "POST")
      .then(() => unread.refetch())
      .catch(() => null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user]);

  async function loadSections(id: string) {
    if (detailed[id]) return;
    try {
      const d = await apiGet<{ sections: Section[] }>(`v1/changelogs/${id}`);
      setDetailed((prev) => ({ ...prev, [id]: d.sections }));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat detail.");
    }
  }

  const items = list.data ?? [];

  return (
    <main className="mx-auto max-w-2xl p-4">
      <div className="mb-4 flex items-center gap-3">
        <h1 className="text-2xl font-bold">Changelog</h1>
        {user && (unread.data?.unread ?? 0) > 0 && <Badge>{unread.data?.unread} baru</Badge>}
        <a href="/changelogs/rss.xml" className="ml-auto text-sm underline">
          RSS
        </a>
      </div>

      {list.isLoading &&
        [0, 1].map((i) => (
          <Card key={i} className="mb-3">
            <CardContent className="p-4">
              <Skeleton className="h-5 w-1/2" />
            </CardContent>
          </Card>
        ))}

      {list.isError && (
        <Card>
          <CardContent className="p-4 text-sm">
            Gagal memuat changelog.{" "}
            <Button variant="link" className="p-0" onClick={() => list.refetch()}>
              Coba lagi
            </Button>
          </CardContent>
        </Card>
      )}

      {!list.isLoading && items.length === 0 && (
        <p className="text-muted-foreground">Belum ada rilis.</p>
      )}

      {items.map((c) => (
        <Card key={c.id} className="mb-3">
          <CardHeader>
            <CardTitle>
              <a className="hover:underline" href={`/changelog/${c.version}`}>
                v{c.version} — {c.title}
              </a>
            </CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-2">
            {c.summary && <p className="text-sm">{c.summary}</p>}
            {detailed[c.id] ? (
              detailed[c.id].length > 0 ? (
                detailed[c.id].map((s, i) => (
                  <div key={i} className="text-sm">
                    <Badge variant="secondary" className="mr-2">
                      {KIND_LABEL[s.kind] ?? s.kind}
                    </Badge>
                    <span className="font-medium">{s.title}</span>
                    {s.body_md && (
                      <p className="text-muted-foreground mt-1 whitespace-pre-wrap">{s.body_md}</p>
                    )}
                  </div>
                ))
              ) : (
                <p className="text-muted-foreground text-sm">Tanpa detail section.</p>
              )
            ) : (
              <Button
                variant="outline"
                size="sm"
                className="self-start"
                onClick={() => loadSections(c.id)}
              >
                Lihat detail
              </Button>
            )}
          </CardContent>
        </Card>
      ))}
    </main>
  );
}
