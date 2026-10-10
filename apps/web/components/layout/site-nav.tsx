"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { apiGet, logout } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

const LINKS: { href: string; label: string }[] = [
  { href: "/feed", label: "Feed" },
  { href: "/portfolios", label: "Karya" },
  { href: "/leaderboards", label: "Peringkat" },
  { href: "/users", label: "Warga" },
  { href: "/changelog", label: "Changelog" },
];

export function SiteNav() {
  const user = useAuthStore((s) => s.user);
  const pathname = usePathname();
  const router = useRouter();

  const notif = useQuery({
    queryKey: ["notif-count"],
    queryFn: () => apiGet<{ unread: number }>("v1/notifications/unread-count"),
    enabled: Boolean(user),
    refetchInterval: 60_000,
  });

  async function doLogout() {
    await logout();
    router.push("/login");
  }

  return (
    <header className="border-b">
      <nav className="mx-auto flex max-w-4xl flex-wrap items-center gap-3 p-3">
        <Link href="/" className="font-bold">
          Grafikarsa
        </Link>
        {LINKS.map((l) => (
          <Link
            key={l.href}
            href={l.href}
            className={`text-sm ${pathname === l.href ? "font-semibold" : "text-muted-foreground hover:underline"}`}
          >
            {l.label}
          </Link>
        ))}

        <div className="ml-auto flex items-center gap-2">
          {user ? (
            <>
              <Link href="/notifications" className="relative">
                <Button variant="ghost" size="sm">
                  Notifikasi
                </Button>
                {(notif.data?.unread ?? 0) > 0 && (
                  <Badge className="absolute -top-1 -right-1 px-1 text-[10px]">{notif.data?.unread}</Badge>
                )}
              </Link>
              <Link href="/me/portfolios" className="text-sm">
                Karya saya
              </Link>
              <Link href="/settings" className="text-sm">
                Pengaturan
              </Link>
              <Link href={`/${user.username}`}>
                <Avatar className="size-7">
                  <AvatarImage src={`/api/v1/media/avatar/${user.id}`} alt="" />
                  <AvatarFallback>{user.name?.slice(0, 1).toUpperCase()}</AvatarFallback>
                </Avatar>
              </Link>
              <Button variant="outline" size="sm" onClick={doLogout}>
                Keluar
              </Button>
            </>
          ) : (
            <Link href="/login">
              <Button size="sm">Masuk</Button>
            </Link>
          )}
        </div>
      </nav>
    </header>
  );
}
