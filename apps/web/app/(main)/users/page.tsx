"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { toast } from "sonner";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { apiGet } from "@/lib/api";

type User = { id: string; username: string; name: string };

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [q, setQ] = useState("");

  useEffect(() => {
    apiGet<User[]>("v1/users")
      .then(setUsers)
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : "Gagal."));
  }, []);

  const filtered = users.filter(
    (u) => u.username.includes(q) || u.name.toLowerCase().includes(q.toLowerCase()),
  );

  return (
    <main className="mx-auto max-w-2xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Pengguna</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Cari..." />
          {filtered.map((u) => (
            <Link key={u.id} href={`/${u.username}`} className="border-b py-2 hover:underline">
              {u.name} <span className="text-muted-foreground">@{u.username}</span>
            </Link>
          ))}
        </CardContent>
      </Card>
    </main>
  );
}
