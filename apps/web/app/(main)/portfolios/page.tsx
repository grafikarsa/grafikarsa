"use client";

import { useQuery } from "@tanstack/react-query";
import { Suspense } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { apiGet } from "@/lib/api";

type Item = {
  id: string;
  title: string;
  slug: string;
  username: string;
  owner_name: string;
  published_at?: string;
};

const SORTS = [
  { value: "latest", label: "Terbaru" },
  { value: "top", label: "Terpopuler" },
  { value: "relevance", label: "Relevansi" },
];

function CatalogInner() {
  const params = useSearchParams();
  const search = params.get("search") ?? "";
  const user = params.get("user") ?? "";
  const tag = params.get("tag") ?? "";
  const sort = params.get("sort") ?? "latest";
  const qs = new URLSearchParams({ page: "1", per_page: "20", sort });
  if (search) qs.set("search", search);
  if (user) qs.set("user", user);
  if (tag) qs.set("tag_id", tag);
  const { data, isLoading } = useQuery({
    queryKey: ["portfolios", search, user, tag, sort],
    queryFn: () => apiGet<Item[]>(`v1/portfolios?${qs.toString()}`),
  });
  const items = data ?? [];

  return (
    <main className="mx-auto max-w-4xl p-4">
      <h1 className="mb-4 text-2xl font-bold">Katalog Portfolio</h1>
      {user && (
        <p className="text-muted-foreground mb-2 text-sm">
          Menampilkan karya <span className="font-medium">@{user}</span> ·{" "}
          <Link href="/portfolios" className="underline">
            tampilkan semua
          </Link>
        </p>
      )}
      <form action="/portfolios" className="mb-4 flex flex-wrap gap-2">
        <Input name="search" placeholder="Cari judul atau pembuat..." defaultValue={search} aria-label="Cari portfolio" />
        {user && <input type="hidden" name="user" value={user} />}
        {tag && <input type="hidden" name="tag" value={tag} />}
        <select
          name="sort"
          defaultValue={sort}
          aria-label="Urutkan"
          className="rounded border bg-transparent px-2 text-sm"
        >
          {SORTS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </form>
      {isLoading && <p>Memuat...</p>}
      <div className="grid gap-4 sm:grid-cols-2">
        {items.map((it) => (
          <Link key={it.id} href={`/${it.username}/${it.slug}`}>
            <Card>
              <CardContent className="p-4">
                <p className="font-semibold">{it.title}</p>
                <p className="text-muted-foreground text-sm">{it.owner_name}</p>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </main>
  );
}

export default function CatalogPage() {
  // useSearchParams butuh Suspense boundary saat prerender.
  return (
    <Suspense fallback={<p className="p-4 text-sm">Memuat katalog...</p>}>
      <CatalogInner />
    </Suspense>
  );
}
