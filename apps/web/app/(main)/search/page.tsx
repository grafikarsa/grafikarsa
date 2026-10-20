"use client";

import { useState } from "react";
import Link from "next/link";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { apiGet } from "@/lib/api";

type Result = {
  portfolios: { id: string; title: string; slug: string; username: string }[];
  users: { id: string; username: string; name: string }[];
  suggest: { slug: string; name: string }[];
};

export default function SearchPage() {
  const [q, setQ] = useState("");
  const [res, setRes] = useState<Result | null>(null);

  async function search(v: string) {
    setQ(v);
    if (v.length < 2) {
      setRes(null);
      return;
    }
    try {
      const data = await apiGet<Result>(`v1/search?q=${encodeURIComponent(v)}`);
      setRes(data);
    } catch {
      /* abaikan */
    }
  }

  return (
    <main className="mx-auto max-w-2xl p-4">
      <h1 className="mb-4 text-2xl font-bold">Pencarian</h1>
      <Input
        value={q}
        onChange={(e) => search(e.target.value)}
        placeholder="Minimal 2 karakter..."
      />
      {res && (
        <div className="mt-4 flex flex-col gap-4">
          {res.portfolios.map((p) => (
            <Link key={p.id} href={`/${p.username}/${p.slug}`}>
              <Card>
                <CardContent className="p-3">
                  <p className="font-semibold">{p.title}</p>
                  <p className="text-muted-foreground text-xs">@{p.username}</p>
                </CardContent>
              </Card>
            </Link>
          ))}
          {res.users.map((u) => (
            <Link key={u.id} href={`/${u.username}`}>
              <Card>
                <CardContent className="p-3">
                  <p className="font-semibold">
                    {u.name} <span className="text-muted-foreground">@{u.username}</span>
                  </p>
                </CardContent>
              </Card>
            </Link>
          ))}
          {res.portfolios.length === 0 && res.users.length === 0 && (
            <p className="text-muted-foreground text-sm">
              Tidak ada hasil.
              {res.suggest.map((s) => ` Coba: ${s.name}`).join("")}
            </p>
          )}
        </div>
      )}
    </main>
  );
}
