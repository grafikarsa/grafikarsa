"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useAuthStore } from "@/lib/auth-store";
import { apiGet, apiSend } from "@/lib/api";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

type Prefs = {
  portfolio_liked: boolean;
  new_comment: boolean;
  new_follower: boolean;
  digest_daily: boolean;
};

const LINKS = [
  { href: "/settings/profile", label: "Profil & foto" },
  { href: "/settings/password", label: "Ganti password" },
  { href: "/me/portfolios", label: "Portfolio saya" },
  { href: "/settings/sessions", label: "Perangkat & sesi" },
  { href: "/notifications", label: "Notifikasi" },
  { href: "/feedback", label: "Masukan & laporan" },
];

export default function SettingsPage() {
  const user = useAuthStore((s) => s.user);
  const qc = useQueryClient();
  const pathname = usePathname();

  const prefs = useQuery({
    queryKey: ["notif-prefs"],
    queryFn: () => apiGet<Prefs>("v1/me/notification-prefs"),
    enabled: Boolean(user),
  });

  const save = useMutation({
    mutationFn: (next: Prefs) => apiSend("v1/me/notification-prefs", "PUT", next),
    onSuccess: () => {
      toast.success("Preferensi disimpan.");
      qc.invalidateQueries({ queryKey: ["notif-prefs"] });
    },
    onError: () => toast.error("Gagal menyimpan preferensi."),
  });

  function toggle(key: keyof Prefs, value: boolean) {
    if (!prefs.data) return;
    save.mutate({ ...prefs.data, [key]: value });
  }

  return (
    <main className="mx-auto flex max-w-xl flex-col gap-4 p-4">
      <Card>
        <CardHeader>
          <CardTitle>Pengaturan</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <p className="text-sm">
            Masuk sebagai <span className="font-semibold">{user?.name ?? "tamu"}</span>
            {user && <span className="text-muted-foreground"> (@{user.username})</span>}
          </p>
          {LINKS.map((l) => (
            <Link
              key={l.href}
              href={l.href}
              className={`text-sm hover:underline ${pathname === l.href ? "font-semibold" : ""}`}
            >
              {l.label}
            </Link>
          ))}
        </CardContent>
      </Card>

      {user && (
        <Card>
          <CardHeader>
            <CardTitle>Preferensi notifikasi</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {(
              [
                ["portfolio_liked", "Saat karya disukai"],
                ["new_comment", "Saat ada komentar"],
                ["new_follower", "Saat ada pengikut baru"],
                ["digest_daily", "Ringkasan harian"],
              ] as [keyof Prefs, string][]
            ).map(([key, label]) => (
              <div key={key} className="flex items-center justify-between">
                <Label htmlFor={key}>{label}</Label>
                <Switch
                  id={key}
                  checked={prefs.data?.[key] ?? false}
                  onCheckedChange={(v) => toggle(key, Boolean(v))}
                  disabled={prefs.isLoading || save.isPending}
                />
              </div>
            ))}
          </CardContent>
        </Card>
      )}
    </main>
  );
}
