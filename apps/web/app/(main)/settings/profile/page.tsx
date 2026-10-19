"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { AvatarUploader } from "@/components/user/avatar-uploader";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { apiGet, apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";
import { firstIssueMessage, profileSchema } from "@/lib/validation";

type Me = {
  id: string;
  username: string;
  name: string;
  bio?: string;
  email: string;
  avatar_media_id?: string | null;
  banner_media_id?: string | null;
  show_email?: boolean;
};

// Validasi klien memakai skema zod di lib/validation.ts, yang mencerminkan
// validator backend. Server tetap menolak kalau aturan tidak cocok.
function validateProfile(name: string, bio: string, username: string): string | null {
  const parsed = profileSchema.safeParse({ name, bio, username });
  return parsed.success ? null : firstIssueMessage(parsed.error);
}

export default function ProfilePage() {
  const me = useAuthStore((s) => s.user);
  const qc = useQueryClient();
  const profile = useQuery({ queryKey: ["me-profile"], queryFn: () => apiGet<Me>("v1/me") });
  const [name, setName] = useState("");
  const [bio, setBio] = useState("");
  const [username, setUsername] = useState("");
  const [showEmail, setShowEmail] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!profile.data) return;
    setName(profile.data.name ?? "");
    setBio(profile.data.bio ?? "");
    setUsername(profile.data.username ?? "");
    setShowEmail(Boolean(profile.data.show_email));
  }, [profile.data]);

  const save = useMutation({
    mutationFn: () => {
      const body: Record<string, unknown> = { name, bio, show_email: showEmail };
      if (profile.data && username !== profile.data.username) body.username = username;
      return apiSend("v1/me", "PATCH", body);
    },
    onSuccess: () => {
      toast.success("Profil disimpan.");
      qc.invalidateQueries({ queryKey: ["me-profile"] });
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal menyimpan."),
  });

  function submit() {
    const vErr = validateProfile(name, bio, username);
    if (vErr) {
      setError(vErr);
      return;
    }
    setError(null);
    save.mutate();
  }

  return (
    <main className="mx-auto max-w-xl p-8">
      <Card>
        <CardHeader>
          <CardTitle>Profil — {me?.name ?? "tamu"}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="grid gap-2">
            <Label>Foto profil (potong 1:1 sebelum unggah)</Label>
            <AvatarUploader
              currentMediaID={profile.data?.avatar_media_id ?? null}
              onDone={() => {
                qc.invalidateQueries({ queryKey: ["me-profile"] });
                qc.invalidateQueries({ queryKey: ["profile"] });
              }}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="username">Username</Label>
            <Input id="username" value={username} onChange={(e) => setUsername(e.target.value.toLowerCase())} />
            <p className="text-muted-foreground text-xs">Mengubah username mengubah URL profilmu.</p>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="name">Nama</Label>
            <Input id="name" value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="bio">Bio (maks 500)</Label>
            <Textarea id="bio" rows={3} value={bio} onChange={(e) => setBio(e.target.value)} />
            <p className="text-muted-foreground text-xs">{bio.length}/500</p>
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="show-email">Tampilkan email di profil publik</Label>
            <Switch id="show-email" checked={showEmail} onCheckedChange={setShowEmail} />
          </div>

          <p className="text-muted-foreground text-xs">Email: {profile.data?.email}</p>
          {error && <p className="text-destructive text-xs">{error}</p>}

          <Button onClick={submit} disabled={save.isPending || profile.isLoading}>
            Simpan profil
          </Button>
        </CardContent>
      </Card>
    </main>
  );
}
