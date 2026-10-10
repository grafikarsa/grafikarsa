"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { apiGet, apiSend } from "@/lib/api";

type User = { id: string; username: string; name: string; role_primary: string; status: string };

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [search, setSearch] = useState("");

  async function load() {
    try {
      const data = await apiGet<User[]>("v1/admin/users");
      setUsers(Array.isArray(data) ? data : (data ?? []));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal memuat.");
    }
  }

  useEffect(() => {
    load();
  }, []);

  const filtered = users.filter((u) => u.username.includes(search) || u.name.toLowerCase().includes(search.toLowerCase()));

  return (
    <main className="mx-auto max-w-4xl p-8">
      <Card>
        <CardHeader>
          <CardTitle>Pengguna</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <Input placeholder="Cari username/nama..." value={search} onChange={(e) => setSearch(e.target.value)} />
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>Nama</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((u) => (
                <TableRow key={u.id}>
                  <TableCell>{u.username}</TableCell>
                  <TableCell>{u.name}</TableCell>
                  <TableCell>{u.role_primary}</TableCell>
                  <TableCell>{u.status}</TableCell>
                  <TableCell>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={async () => {
                        try {
                          const r = await apiSend<{ password: string }>(`v1/admin/users/${u.id}/reset-password`, "POST");
                          toast.success(`Password baru: ${r.password}`);
                        } catch (err) {
                          toast.error(err instanceof Error ? err.message : "Gagal.");
                        }
                      }}
                    >
                      Reset password
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </main>
  );
}
