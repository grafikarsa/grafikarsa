"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { apiGet } from "@/lib/api";

// /me: arahkan ke profil sendiri (deep-link notifikasi promotion -> /me).
//
// Halaman ini HARUS client component. API hanya menerima access token lewat
// header Authorization — bukan cookie sesi. Versi server component yang
// meneruskan cookie selalu berakhir 401 lalu redirect ke /login, padahal
// session-nya sah.
export default function MePage() {
  const router = useRouter();

  useEffect(() => {
    let dibatalkan = false;
    apiGet<{ username?: string }>("v1/me")
      .then((me) => {
        if (dibatalkan) return;
        router.replace(me?.username ? `/${me.username}` : "/login?next=/me");
      })
      .catch(() => {
        if (!dibatalkan) router.replace("/login?next=/me");
      });
    return () => {
      dibatalkan = true;
    };
  }, [router]);

  return <p className="p-8 text-sm text-muted-foreground">Memuat...</p>;
}
