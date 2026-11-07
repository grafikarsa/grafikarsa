"use client";

import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import {
  CalendarRange,
  GraduationCap,
  School,
  ScrollText,
  Users,
} from "lucide-react";

import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { adminNav } from "@/lib/auth/permissions";
import { useSession } from "@/lib/auth/session";

const SECTION_ICONS: Record<string, LucideIcon> = {
  "/admin/pengguna": Users,
  "/admin/akademik/jurusan": GraduationCap,
  "/admin/akademik/tahun-ajaran": CalendarRange,
  "/admin/akademik/kelas": School,
  "/admin/audit": ScrollText,
};

const SECTION_DESCRIPTIONS: Record<string, string> = {
  "/admin/pengguna": "Kelola akun, peran, dan status pengguna.",
  "/admin/akademik/jurusan": "Atur data jurusan dan status aktifnya.",
  "/admin/akademik/tahun-ajaran": "Kelola tahun ajaran dan tanggal berlakunya.",
  "/admin/akademik/kelas": "Kelola rombel pada setiap tahun ajaran.",
  "/admin/audit": "Telusuri jejak aktivitas administratif.",
};

export function AdminDashboard() {
  const { session, status } = useSession();

  if (status !== "authenticated") {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-80" />
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} className="h-28 w-full" />
          ))}
        </div>
      </div>
    );
  }

  const sections = adminNav(session).filter((item) => item.href !== "/admin");

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">
          Selamat datang, {session.user.displayName}
        </h2>
        <p className="text-sm text-muted-foreground">
          Pilih bagian yang ingin kamu kelola.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {sections.map((item) => {
          const Icon = SECTION_ICONS[item.href] ?? Users;

          return (
            <Link key={item.href} href={item.href} className="group">
              <Card className="h-full transition-colors group-hover:bg-muted/50">
                <CardHeader>
                  <span className="mb-1 flex size-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Icon aria-hidden="true" className="size-4" />
                  </span>
                  <CardTitle>{item.label}</CardTitle>
                  <CardDescription>
                    {SECTION_DESCRIPTIONS[item.href] ?? "Buka bagian admin."}
                  </CardDescription>
                </CardHeader>
              </Card>
            </Link>
          );
        })}
      </div>
    </div>
  );
}
