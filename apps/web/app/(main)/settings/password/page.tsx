"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

// Halaman wajib untuk user hasil import/admin-create: password awal harus
// diganti sebelum endpoint lain bisa dipakai (gate PASSWORD_CHANGE_REQUIRED).
export default function ChangePasswordPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (next !== confirm) {
      toast.error("Konfirmasi password tidak sama.");
      return;
    }
    if (next.length < 10) {
      toast.error("Password minimal 10 karakter.");
      return;
    }
    setBusy(true);
    try {
      await apiSend("v1/me/password", "PATCH", {
        current_password: current,
        new_password: next,
      });
      toast.success("Password diganti. Silakan masuk ulang.");
      // Semua sesi dicabut oleh server; arahkan ke login.
      useAuthStore.getState().logout();
      router.push("/login");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal mengganti password.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto max-w-sm p-8">
      <Card>
        <CardHeader>
          <CardTitle>Ganti password</CardTitle>
        </CardHeader>
        <CardContent>
          {user?.must_change_password && (
            <p className="mb-3 rounded border border-amber-500/40 bg-amber-500/5 p-2 text-xs">
              Kamu memakai password sementara. Ganti dulu untuk membuka fitur lain.
            </p>
          )}
          <form onSubmit={submit} className="flex flex-col gap-3">
            <div className="grid gap-1">
              <Label htmlFor="current">Password saat ini</Label>
              <Input id="current" type="password" value={current} onChange={(e) => setCurrent(e.target.value)} required />
            </div>
            <div className="grid gap-1">
              <Label htmlFor="next">Password baru</Label>
              <Input id="next" type="password" value={next} onChange={(e) => setNext(e.target.value)} required minLength={10} />
              <p className="text-muted-foreground text-xs">Minimal 10 karakter, kombinasi huruf besar/kecil dan angka.</p>
            </div>
            <div className="grid gap-1">
              <Label htmlFor="confirm">Ulangi password baru</Label>
              <Input id="confirm" type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} required />
            </div>
            <Button type="submit" disabled={busy || !current || !next}>
              {busy ? "Menyimpan..." : "Ganti password"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
