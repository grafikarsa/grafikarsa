"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { apiGet } from "@/lib/api";

type User = { id: string; username: string; name: string };

async function getUser(username: string): Promise<User> {
  return apiGet<User>(`v1/users/${username}`);
}

export default function ProfileByUsername({ params }: { params: Promise<{ username: string }> }) {
  const { username } = use(params);
  const [data, setData] = useState<User | null>(null);
  useEffect(() => {
    getUser(username).then((u) => setData(u)).catch(() => setData(null));
  }, [username]);
  if (!data) return <main className="p-8">Memuat...</main>;
  return (
    <main className="mx-auto max-w-2xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>{data.name} <span className="text-muted-foreground">@{data.username}</span></CardTitle>
        </CardHeader>
        <CardContent>
          <Link href={`/portfolios?user=${data.username}`} className="text-sm underline">Lihat karya</Link>
        </CardContent>
      </Card>
    </main>
  );
}
