"use client";

import { useInfiniteQuery, useMutation, useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { apiGet, apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

type FeedItem = { id: string; title: string; slug: string; username: string; owner_name: string; score?: number };
type Algorithm = "smart" | "recent" | "following";

const EMPTY: Record<Algorithm, string> = {
  smart: "Belum ada rekomendasi. Ikuti beberapa teman atau buat portfolio pertamamu.",
  recent: "Belum ada karya yang diterbitkan.",
  following: "Kamu belum mengikuti siapa pun. Cari teman di direktori pengguna.",
};

export function FeedList() {
  const user = useAuthStore((s) => s.user);
  const [algo, setAlgo] = useState<Algorithm>("recent");
  const sentinel = useRef<HTMLDivElement | null>(null);

  // Preferensi algoritma diambil dari server (dihormati lintas device).
  const prefs = useQuery({
    queryKey: ["feed-preferences"],
    queryFn: () => apiGet<{ algorithm: Algorithm }>("v1/feed/preferences"),
    enabled: Boolean(user),
  });
  useEffect(() => {
    if (prefs.data?.algorithm) setAlgo(prefs.data.algorithm);
  }, [prefs.data]);

  const savePrefs = useMutation({
    mutationFn: (algorithm: Algorithm) => apiSend("v1/feed/preferences", "PUT", { algorithm }),
    onError: () => toast.error("Gagal menyimpan preferensi feed."),
  });

  function changeAlgo(v: string) {
    const next = v as Algorithm;
    setAlgo(next);
    if (user) savePrefs.mutate(next);
  }

  const q = useInfiniteQuery({
    queryKey: ["feed", algo, user?.username ?? "guest"],
    queryFn: ({ pageParam }) => apiGet<FeedItem[]>(`v1/feed?algorithm=${algo}&page=${pageParam}&per_page=15`),
    getNextPageParam: (last, all) => (last.length === 15 ? all.length + 1 : undefined),
    initialPageParam: 1,
    enabled: algo !== "following" || Boolean(user),
  });

  const items = q.data?.pages.flatMap((p) => p) ?? [];

  useEffect(() => {
    const el = sentinel.current;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && q.hasNextPage && !q.isFetchingNextPage) {
          q.fetchNextPage();
        }
      },
      { rootMargin: "200px" },
    );
    io.observe(el);
    return () => io.disconnect();
  }, [q]);

  async function like(id: string) {
    try {
      await apiSend(`v1/portfolios/id/${id}/like`, "POST");
      toast.success("Disukai.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal menyukai.");
    }
  }

  async function share(item: FeedItem) {
    const url = `${window.location.origin}/${item.username}/${item.slug}`;
    if (navigator.share) {
      await navigator.share({ title: item.title, url }).catch(() => null);
      return;
    }
    await navigator.clipboard.writeText(url);
    toast.success("Tautan disalin.");
  }

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-4 p-4">
      <Tabs value={algo} onValueChange={changeAlgo}>
        <TabsList>
          <TabsTrigger value="smart" disabled={!user}>
            Untukmu
          </TabsTrigger>
          <TabsTrigger value="recent">Terbaru</TabsTrigger>
          <TabsTrigger value="following" disabled={!user}>
            Mengikuti
          </TabsTrigger>
        </TabsList>
      </Tabs>

      {q.isLoading &&
        [0, 1, 2].map((i) => (
          <Card key={i}>
            <CardContent className="flex flex-col gap-2 p-4">
              <Skeleton className="h-5 w-2/3" />
              <Skeleton className="h-4 w-1/3" />
            </CardContent>
          </Card>
        ))}

      {q.isError && (
        <Card>
          <CardContent className="p-4 text-sm">
            Gagal memuat feed.{" "}
            <Button variant="link" className="p-0" onClick={() => q.refetch()}>
              Coba lagi
            </Button>
          </CardContent>
        </Card>
      )}

      {items.map((it) => (
        <Card key={it.id}>
          <CardContent className="flex flex-col gap-2 p-4">
            <Link href={`/${it.username}/${it.slug}`} className="text-lg font-semibold hover:underline">
              {it.title}
            </Link>
            <p className="text-muted-foreground text-sm">
              oleh{" "}
              <Link href={`/${it.username}`} className="hover:underline">
                {it.owner_name}
              </Link>{" "}
              (@{it.username})
              {typeof it.score === "number" && <span className="ml-2 text-xs">skor {it.score.toFixed(2)}</span>}
            </p>
            <div className="flex gap-2">
              {user && (
                <Button variant="outline" size="sm" onClick={() => like(it.id)}>
                  Suka
                </Button>
              )}
              <Button variant="ghost" size="sm" onClick={() => share(it)}>
                Bagikan
              </Button>
            </div>
          </CardContent>
        </Card>
      ))}

      {!q.isLoading && items.length === 0 && (
        <Card>
          <CardContent className="flex flex-col items-center gap-3 p-6 text-center">
            <p className="text-muted-foreground text-sm">{EMPTY[algo]}</p>
            <div className="flex gap-2">
              <Link className={buttonVariants({ variant: "outline" })} href="/users">
                Cari teman
              </Link>
              <Link className={buttonVariants()} href="/me/portfolios">
                Buat portfolio
              </Link>
            </div>
          </CardContent>
        </Card>
      )}

      <div ref={sentinel} className="h-4" />
      {q.isFetchingNextPage && <p className="text-muted-foreground text-center text-sm">Memuat...</p>}
    </div>
  );
}
