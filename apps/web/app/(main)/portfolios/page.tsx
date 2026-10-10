"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { apiGet } from "@/lib/api";

type Item = { id: string; title: string; slug: string; username: string; owner_name: string };

export default function CatalogPage() {
  const params = useSearchParams();
  const search = params.get("search") ?? "";
  const { data, isLoading } = useQuery({
    queryKey: ["portfolios", search],
    queryFn: () => apiGet<{ data: Item[] }>(`v1/portfolios?page=1&per_page=20`),
  });
  const items = data?.data ?? [];

  return (
    <main className="mx-auto max-w-4xl p-4">
      <h1 className="mb-4 text-2xl font-bold">Katalog Portfolio</h1>
      <form action="/portfolios" className="mb-4 flex gap-2">
        <Input name="search" placeholder="Cari judul..." defaultValue={search} />
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
