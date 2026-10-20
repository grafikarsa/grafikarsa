"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { login } from "@/lib/api";
import { firstIssueMessage, loginSchema } from "@/lib/validation";
import { Turnstile } from "@/components/security/turnstile";

function LoginInner() {
  const router = useRouter();
  const params = useSearchParams();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [captcha, setCaptcha] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    // Validasi klien (zod) sebelum ke server — server tetap sumber kebenaran.
    const parsed = loginSchema.safeParse({ username, password });
    if (!parsed.success) {
      toast.error(firstIssueMessage(parsed.error));
      return;
    }
    setLoading(true);
    try {
      await login(username, password, captcha || undefined);
      toast.success("Login berhasil.");
      // Hanya izinkan path internal (cegah open redirect via ?next=absolut).
      const next = params.get("next") ?? "/";
      router.push(next.startsWith("/") && !next.startsWith("//") ? next : "/");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Login gagal.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-sm flex-col justify-center p-8">
      <Card>
        <CardHeader>
          {/* CardTitle dirender sebagai <div>, jadi halaman ini perlu <h1>
              sendiri — selain struktur dokumen yang benar, ini yang
              membuat test E2E/a11y bisa menemukan judul halaman. */}
          <h1 className="text-lg font-semibold">Masuk Grafikarsa</h1>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="flex flex-col gap-4">
            <div className="grid gap-2">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
              />
            </div>
            <Turnstile onToken={setCaptcha} />
            <Button type="submit" disabled={loading}>
              {loading ? "Masuk..." : "Masuk"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}

export default function LoginPage() {
  // useSearchParams (?next=) butuh Suspense boundary saat prerender.
  return (
    <Suspense fallback={<main className="mx-auto max-w-sm p-8">Memuat...</main>}>
      <LoginInner />
    </Suspense>
  );
}
