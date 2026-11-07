"use client";

import Link from "next/link";
import { LayoutDashboard, LogOut, Settings, UserRound } from "lucide-react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { useLogout, useSession } from "@/lib/auth/session";

function initials(displayName: string): string {
  const parts = displayName.trim().split(/\s+/).filter(Boolean).slice(0, 2);

  if (parts.length === 0) {
    return "?";
  }

  return parts
    .map((part) => part.charAt(0).toUpperCase())
    .join("");
}

export function UserMenu() {
  const { session, status } = useSession();
  const logout = useLogout();

  if (status === "loading") {
    return <Skeleton className="size-8 rounded-full" />;
  }

  if (status === "anonymous") {
    return (
      <Button asChild size="sm" className="ml-1">
        <Link href="/masuk">Masuk</Link>
      </Button>
    );
  }

  const { user } = session;
  const username = user.username;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="rounded-full"
          aria-label="Menu akun"
        >
          <Avatar>
            {user.avatarUrl ? (
              <AvatarImage src={user.avatarUrl} alt={user.displayName} />
            ) : null}
            <AvatarFallback>{initials(user.displayName)}</AvatarFallback>
          </Avatar>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-44">
        {username ? (
          <DropdownMenuItem asChild>
            <Link href={`/@${username}`}>
              <UserRound aria-hidden="true" />
              Profil saya
            </Link>
          </DropdownMenuItem>
        ) : null}
        <DropdownMenuItem asChild>
          <Link href="/studio">
            <LayoutDashboard aria-hidden="true" />
            Studio
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <Link href="/pengaturan/profil">
            <Settings aria-hidden="true" />
            Pengaturan
          </Link>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          variant="destructive"
          disabled={logout.isPending}
          onSelect={() => logout.mutate()}
        >
          <LogOut aria-hidden="true" />
          Keluar
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
