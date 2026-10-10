"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet } from "@/lib/api";

type Person = { id: string; username: string; name: string };

export default function FollowListPage({ kind }: { kind: "followers" | "following" }) {
  const params = useParams<{ username: string }>();
  const username = params?.username ?? "";
  const q = useQuery({
    queryKey: ["follow-list", kind, username],
    queryFn: () => apiGet<Person[]>(`v1/users/${username}/${kind}?per_page=50`),
    enabled: Boolean(username),
  });
  const items = Array.isArray(q.data) ? q.data : [];

  return (
    <main className="mx-auto max-w-xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>
            {kind === "followers" ? "Pengikut" : "Mengikuti"} · @{username}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          {q.isLoading && <Skeleton className="h-16 w-full" />}
          {items.map((u) => (
            <Link key={u.id} href={`/${u.username}`} className="text-sm hover:underline">
              {u.name} <span className="text-muted-foreground">@{u.username}</span>
            </Link>
          ))}
          {!q.isLoading && items.length === 0 && (
            <p className="text-muted-foreground text-sm">Belum ada data.</p>
          )}
          <Link href={`/${username}`} className="text-sm underline">
            ← Kembali ke profil
          </Link>
        </CardContent>
      </Card>
    </main>
  );
}
