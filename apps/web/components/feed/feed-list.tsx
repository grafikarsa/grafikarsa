"use client";

import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { apiGet, apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

type Item = { id: string; title: string; slug: string; username: string; owner_name: string; score?: number };

export function FeedList() {
  const user = useAuthStore((s) => s.user);
  const [algo, setAlgo] = useState(user ? "smart" : "recent");
  const qc = useQueryClient();

  const q = useInfiniteQuery({
    queryKey: ["feed", algo],
    queryFn: ({ pageParam }) => apiGet<{ data: Item[] }>(`v1/feed?algorithm=${algo}&page=${pageParam ?? 1}&per_page=15`),
    getNextPageParam: (last, all) => (last.data.length === 15 ? all.length + 1 : undefined),
    initialPageParam: 1,
  });

  const items = q.data?.pages.flatMap((p) => p.data) ?? [];

  async function like(id: string) {
    try {
      await apiSend(`v1/portfolios/id/${id}/like`, "POST");
      toast.success("Disukai.");
      qc.invalidateQueries({ queryKey: ["feed"] });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-4 p-4">
      <Tabs value={algo} onValueChange={setAlgo}>
        <TabsList>
          <TabsTrigger value="smart">Untukmu</TabsTrigger>
          <TabsTrigger value="recent">Terbaru</TabsTrigger>
          <TabsTrigger value="following">Mengikuti</TabsTrigger>
        </TabsList>
      </Tabs>
      {q.isLoading && <p>Memuat...</p>}
      {items.map((it) => (
        <Card key={it.id}>
          <CardContent className="flex flex-col gap-2 p-4">
            <Link href={`/${it.username}/${it.slug}`} className="text-lg font-semibold hover:underline">
              {it.title}
            </Link>
            <p className="text-muted-foreground text-sm">
              oleh {it.owner_name} (@{it.username})
            </p>
            {user && <Button variant="outline" size="sm" onClick={() => like(it.id)}>Suka</Button>}
          </CardContent>
        </Card>
      ))}
      {!q.isLoading && items.length === 0 && (
        <div className="text-center">
          <p className="text-muted-foreground">Belum ada portfolio. Jadilah yang pertama!</p>
          <Link href="/me/portfolios/new">
            <Button className="mt-2">Buat portfolio</Button>
          </Link>
        </div>
      )}
      {q.hasNextPage && <Button variant="ghost" onClick={() => q.fetchNextPage()}>Muat lagi</Button>}
    </div>
  );
}

export function useFeedPrefs() {
  return useMutation({
    mutationFn: (algorithm: string) => apiSend("v1/feed/preferences", "PUT", { algorithm }),
  });
}
