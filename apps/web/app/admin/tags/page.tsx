"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { apiGet, apiSend } from "@/lib/api";

type Tag = { id: string; slug: string; name: string };

export default function AdminTagsPage() {
  const [tags, setTags] = useState<Tag[]>([]);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");

  async function load() {
    const data = await apiGet<Tag[]>("v1/tags");
    setTags(data);
  }

  useEffect(() => {
    load().catch(() => null);
  }, []);

  async function create() {
    try {
      await apiSend("v1/admin/tags", "POST", { name, slug });
      setName("");
      setSlug("");
      load();
      toast.success("Tag dibuat.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  return (
    <main className="mx-auto max-w-2xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Tags</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <div className="flex gap-2">
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Nama" />
            <Input value={slug} onChange={(e) => setSlug(e.target.value)} placeholder="slug" />
            <Button onClick={create}>Tambah</Button>
          </div>
          {tags.map((t) => (
            <p key={t.id} className="border-b py-1 text-sm">{t.name} <span className="text-muted-foreground">#{t.slug}</span></p>
          ))}
        </CardContent>
      </Card>
    </main>
  );
}
