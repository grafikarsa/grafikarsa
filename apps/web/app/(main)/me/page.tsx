import { redirect } from "next/navigation";
import { cookies } from "next/headers";

// /me: arahkan ke profil sendiri (deep-link notifikasi promotion->/me).
export default async function MePage() {
  const jar = await cookies();
  const base = process.env.INTERNAL_API_URL ?? "http://localhost:8080/v1";
  try {
    const res = await fetch(`${base}/me`, {
      headers: { cookie: jar.toString() },
      cache: "no-store",
    });
    if (res.ok) {
      const json = await res.json();
      if (json?.data?.username) redirect(`/${json.data.username}`);
    }
  } catch {
    /* jatuh ke login */
  }
  redirect("/login?next=/me");
}
