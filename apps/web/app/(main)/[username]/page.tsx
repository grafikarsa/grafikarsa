"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useParams } from "next/navigation";
import { toast } from "sonner";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

type Profile = {
  id: string;
  username: string;
  name: string;
  bio?: string;
  role_primary?: string;
  status?: string;
  major?: string;
  avatar_media_id?: string | null;
  is_following?: boolean;
  followers?: number;
  following?: number;
};

type Portfolio = { id: string; title: string; slug: string; username: string };

export default function ProfileByUsername() {
  const params = useParams<{ username: string }>();
  const username = params?.username ?? "";
  const me = useAuthStore((s) => s.user);
  const qc = useQueryClient();

  const profile = useQuery({
    queryKey: ["profile", username],
    queryFn: () => apiGet<Profile>(`v1/users/${username}`),
    enabled: Boolean(username),
  });

  const works = useQuery({
    queryKey: ["profile-works", username],
    queryFn: () =>
      apiGet<Portfolio[]>(`v1/portfolios?user=${encodeURIComponent(username)}&per_page=12`),
    enabled: Boolean(username),
  });

  const follow = useMutation({
    mutationFn: (isFollowing: boolean) =>
      apiSend(`v1/users/${username}/follow`, isFollowing ? "DELETE" : "POST"),
    onSuccess: () => {
      toast.success("Diperbarui.");
      qc.invalidateQueries({ queryKey: ["profile", username] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal."),
  });

  const p = profile.data;
  const isSelf = me?.username === username;

  return (
    <main className="mx-auto flex max-w-2xl flex-col gap-4 p-4">
      {profile.isLoading && <Skeleton className="h-40 w-full" />}
      {profile.isError && (
        <Card>
          <CardContent className="p-4 text-sm">Profil tidak ditemukan.</CardContent>
        </Card>
      )}

      {p && (
        <Card>
          <CardHeader className="flex-row items-center gap-3">
            <Avatar className="size-14">
              <AvatarImage
                src={p.avatar_media_id ? `/api/v1/media/${p.avatar_media_id}?w=320` : ""}
                alt=""
              />
              <AvatarFallback>{p.name?.slice(0, 1).toUpperCase()}</AvatarFallback>
            </Avatar>
            <div className="flex flex-col gap-1">
              <CardTitle>
                {p.name} <span className="text-muted-foreground font-normal">@{p.username}</span>
              </CardTitle>
              <div className="flex flex-wrap items-center gap-2">
                {p.role_primary && <Badge variant="secondary">{p.role_primary}</Badge>}
                {p.status === "lulus" && <Badge>Alumni</Badge>}
                {typeof p.followers === "number" && (
                  <span className="text-muted-foreground text-xs">
                    {p.followers} pengikut · {p.following ?? 0} mengikuti
                  </span>
                )}
              </div>
            </div>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {p.bio && <p className="text-sm">{p.bio}</p>}
            <div className="flex flex-wrap gap-2">
              {!isSelf && me && (
                <Button
                  variant={p.is_following ? "outline" : "default"}
                  size="sm"
                  onClick={() => follow.mutate(Boolean(p.is_following))}
                  disabled={follow.isPending}
                >
                  {p.is_following ? "Batal ikuti" : "Ikuti"}
                </Button>
              )}
              {!me && (
                <Link href={`/login?next=/${p.username}`} className="text-sm underline">
                  Masuk untuk mengikuti
                </Link>
              )}
              <Link className="text-sm underline" href={`/users/${p.username}/followers`}>
                Pengikut
              </Link>
              <Link className="text-sm underline" href={`/users/${p.username}/following`}>
                Mengikuti
              </Link>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Karya terbit</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          {works.isLoading && <Skeleton className="h-20 w-full" />}
          {(works.data ?? []).map((w) => (
            <Link key={w.id} href={`/${w.username}/${w.slug}`} className="text-sm hover:underline">
              {w.title}
            </Link>
          ))}
          {!works.isLoading && (works.data ?? []).length === 0 && (
            <p className="text-muted-foreground text-sm">Belum ada karya yang terbit.</p>
          )}
        </CardContent>
      </Card>
    </main>
  );
}
