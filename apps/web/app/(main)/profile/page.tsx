"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { apiGet, apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

export default function ProfilePage() {
  const user = useAuthStore((s) => s.user);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [progress, setProgress] = useState<number | null>(null);

  useEffect(() => {
    apiGet<{ avatar_url?: string }>("v1/me").catch(() => null);
  }, []);

  async function upload() {
    if (!file) return;
    setProgress(0);
    const fd = new FormData();
    fd.append("purpose", "avatar");
    fd.append("file", file);
    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/v1/media/upload");
    const { accessToken, csrfToken } = useAuthStore.getState();
    if (accessToken) xhr.setRequestHeader("Authorization", `Bearer ${accessToken}`);
    if (csrfToken) xhr.setRequestHeader("X-CSRF-Token", csrfToken);
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) setProgress(Math.round((e.loaded / e.total) * 100));
    };
    xhr.onload = async () => {
      try {
        const json = JSON.parse(xhr.responseText);
        if (!json.success) throw new Error(json.error.message);
        await apiSend(`v1/media/${json.data.media_id}/confirm`, "POST", { link_to: "avatar" });
        setPreview(`/api/v1/media/${json.data.media_id}`);
        toast.success("Avatar diperbarui.");
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Upload gagal.");
      } finally {
        setProgress(null);
      }
    };
    xhr.onerror = () => {
      toast.error("Upload gagal.");
      setProgress(null);
    };
    xhr.send(fd);
  }

  return (
    <main className="mx-auto max-w-xl p-8">
      <Card>
        <CardHeader>
          <CardTitle>Profil — {user?.name ?? "tamu"}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {preview && <img src={preview} alt="Avatar" className="h-24 w-24 rounded-full object-cover" />}
          <input
            type="file"
            accept="image/jpeg,image/png,image/webp"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
          {progress !== null && <p className="text-sm">Mengunggah... {progress}%</p>}
          <Button onClick={upload} disabled={!file || progress !== null}>
            Unggah avatar
          </Button>
          <p className="text-muted-foreground text-xs">Maks 2MB, JPG/PNG/WebP. SVG/HTML ditolak.</p>
        </CardContent>
      </Card>
    </main>
  );
}
