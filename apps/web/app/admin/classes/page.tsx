"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { apiGet, apiSend } from "@/lib/api";

type Major = { id: string; code: string; name: string };
type Cls = { id: string; name: string; grade: number; major_code: string; students: number };

export default function ClassesPage() {
  const [majors, setMajors] = useState<Major[]>([]);
  const [classes, setClasses] = useState<Cls[]>([]);
  const [grade, setGrade] = useState("10");
  const [group, setGroup] = useState("A");
  const [majorId, setMajorId] = useState("");

  async function load() {
    const m = await apiGet<Major[]>("v1/admin/majors");
    setMajors(m);
    const c = await apiGet<Cls[]>("v1/admin/classes");
    setClasses(c);
  }

  useEffect(() => {
    load().catch(() => toast.error("Gagal memuat."));
  }, []);

  async function create() {
    try {
      const years = await apiGet<{ id: string; is_active: boolean }[]>("v1/admin/years");
      const active = years.find((y) => y.is_active);
      if (!active) {
        toast.error("Belum ada tahun aktif.");
        return;
      }
      await apiSend("v1/admin/classes", "POST", {
        year_id: active.id,
        major_id: majorId,
        grade: Number(grade),
        group,
      });
      toast.success("Kelas dibuat.");
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  return (
    <main className="mx-auto max-w-3xl p-4">
      <Card>
        <CardHeader>
          <CardTitle>Kelas</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <div className="flex gap-2">
            <select
              value={majorId}
              onChange={(e) => setMajorId(e.target.value)}
              className="border rounded p-2"
            >
              <option value="">Jurusan...</option>
              {majors.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name}
                </option>
              ))}
            </select>
            <Input
              value={grade}
              onChange={(e) => setGrade(e.target.value)}
              placeholder="10/11/12"
              className="w-24"
            />
            <Input
              value={group}
              onChange={(e) => setGroup(e.target.value)}
              placeholder="A"
              className="w-16"
            />
            <Button onClick={create}>Buat</Button>
          </div>
          {classes.map((c) => (
            <p key={c.id} className="border-b py-1 text-sm">
              {c.name} — {c.students} siswa
            </p>
          ))}
        </CardContent>
      </Card>
    </main>
  );
}
