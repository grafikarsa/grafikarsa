"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api";

const NAV: { href: string; label: string }[] = [
  { href: "/admin/dashboard", label: "Dashboard" },
  { href: "/admin/users", label: "Pengguna" },
  { href: "/admin/moderation", label: "Moderasi" },
  { href: "/admin/portfolios", label: "Portfolio" },
  { href: "/admin/majors", label: "Jurusan" },
  { href: "/admin/years", label: "Tahun Ajaran" },
  { href: "/admin/classes", label: "Kelas" },
  { href: "/admin/series", label: "Series" },
  { href: "/admin/tags", label: "Tags" },
  { href: "/admin/roles", label: "Role" },
  { href: "/admin/assessments", label: "Penilaian" },
  { href: "/admin/feedback", label: "Feedback" },
  { href: "/admin/changelogs", label: "Changelog" },
  { href: "/admin/import", label: "Import" },
  { href: "/admin/promotion", label: "Kenaikan Kelas" },
  { href: "/admin/jobs", label: "Job Queue" },
  { href: "/admin/audit-logs", label: "Audit Log" },
  { href: "/admin/media-quarantine", label: "Media Karantina" },
  { href: "/admin/settings", label: "Pengaturan" },
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const me = useQuery({
    queryKey: ["me-permissions"],
    queryFn: () => apiGet<string[]>("v1/me/permissions"),
    retry: false,
  });
  const perms = new Set(me.data ?? []);
  const isAdmin = perms.has("system.settings"); // admin penuh

  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-4 p-4 md:flex-row">
      <nav className="flex flex-row gap-1 overflow-x-auto md:w-56 md:flex-col">
        {NAV.map((n) => (
          <Link
            key={n.href}
            href={n.href}
            className={`rounded px-3 py-2 text-sm whitespace-nowrap ${
              pathname === n.href ? "bg-primary text-primary-foreground" : "hover:bg-muted"
            }`}
          >
            {n.label}
          </Link>
        ))}
        {isAdmin && (
          <span className="px-3 py-2 text-xs text-muted-foreground">Mode admin penuh</span>
        )}
      </nav>
      <div className="flex-1">{children}</div>
    </div>
  );
}
