"use client";

import { useQuery } from "@tanstack/react-query";
import { useQueryState } from "nuqs";
import { useMemo, useTransition } from "react";
import { apiGetWithMeta } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

/**
 * DataTable server-side untuk halaman admin.
 *
 * PRD §12.3: "Admin users/roles: DataTable server-side (page/sort/search via
 * URL `nuqs`)".
 *
 * Kenapa server-side, bukan filter di browser: daftar user sekolah bisa
 * mencapai ribuan baris. Menarik semuanya ke browser untuk lalu difilter
 * berarti memuat seluruh tabel sekaligus — lambat, boros bandwidth, dan
 * tidak akan layak di VPS 4 GB.
 *
 * State (halaman, urut, pencarian) disimpan di URL lewat `nuqs`, jadi:
 *   - state bisa di-share lewat link,
 *   - tombol back/forward browser bekerja,
 *   - refresh tidak kehilangan posisi.
 */

export type DataTableColumn<T> = {
  key: string;
  label: string;
  /** Kolom yang dipakai untuk mengurutkan. Kosongkan kalau tidak bisa diurutkan. */
  sortKey?: string;
  render: (row: T) => React.ReactNode;
  className?: string;
};

export type DataTableMeta = {
  page: number;
  per_page: number;
  total: number;
};

type DataTableProps<T> = {
  /** Path API (tanpa query string), mis. "v1/admin/users". */
  endpoint: string;
  columns: DataTableColumn<T>[];
  /** Param URL tambahan yang selalu ikut dikirim (mis. filter_scope). */
  extraParams?: Record<string, string | undefined>;
  pageSize?: number;
  emptyMessage?: string;
};

export function DataTable<T>({
  endpoint,
  columns,
  extraParams,
  pageSize = 20,
  emptyMessage = "Belum ada data.",
}: DataTableProps<T>) {
  const [isPending, startTransition] = useTransition();

  // State tabel di URL (nuqs). Halaman & pengurutan memakai 'push' supaya
  // tombol back browser benar-benar kembali ke halaman sebelumnya;
  // pencarian memakai 'replace' supaya mengetik tidak menumpuk history.
  const [pageRaw, setPage] = useQueryState("page", {
    defaultValue: "1",
    history: "push",
  });
  const [sort, setSort] = useQueryState("sort", {
    defaultValue: "username",
    history: "push",
  });
  const [dir, setDir] = useQueryState("dir", {
    defaultValue: "asc",
    history: "push",
  });
  const [search, setSearch] = useQueryState("q", {
    defaultValue: "",
    history: "replace",
  });

  const page = Math.max(1, Number(pageRaw) || 1);

  const query = useMemo(() => {
    const p = new URLSearchParams();
    p.set("page", String(page));
    p.set("per_page", String(pageSize));
    p.set("sort", sort);
    p.set("dir", dir);
    if (search) p.set("search", search);
    for (const [k, v] of Object.entries(extraParams ?? {})) {
      if (v !== undefined && v !== "") p.set(k, v);
    }
    return p.toString();
  }, [page, pageSize, sort, dir, search, extraParams]);

  const { data, isLoading } = useQuery({
    queryKey: [endpoint, query],
    queryFn: async () => {
      // apiGetWithMeta (bukan fetch polos): endpoint ini butuh header
      // Authorization — tanpa itu API membalas 401 dan tabel kosong.
      return apiGetWithMeta<T>(`${endpoint}?${query}`);
    },
  });

  const rows = (data?.data ?? []) as T[];
  const total = data?.meta?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  function toggleSort(sortKey: string | undefined) {
    if (!sortKey) return;
    const nextDir = sort === sortKey && dir === "asc" ? "desc" : "asc";
    startTransition(() => {
      void setSort(sortKey);
      void setDir(nextDir);
      void setPage("1");
    });
  }

  return (
    <div className="flex flex-col gap-4">
      <Input
        placeholder="Cari username/nama/email..."
        defaultValue={search}
        onChange={(e) => {
          const v = e.target.value;
          startTransition(() => {
            void setSearch(v);
            void setPage("1");
          });
        }}
      />

      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((c) => (
              <TableHead
                key={c.key}
                className={c.className}
                onClick={() => toggleSort(c.sortKey)}
                aria-sort={
                  sort === c.sortKey ? (dir === "asc" ? "ascending" : "descending") : "none"
                }
              >
                {c.label}
                {c.sortKey ? (
                  <span className="ml-1 text-xs opacity-60">
                    {sort === c.sortKey ? (dir === "asc" ? "▲" : "▼") : "↕"}
                  </span>
                ) : null}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 && !isLoading ? (
            <TableRow>
              <TableCell colSpan={columns.length} className="text-muted-foreground">
                {emptyMessage}
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row, i) => (
              <TableRow key={i}>
                {columns.map((c) => (
                  <TableCell key={c.key} className={c.className}>
                    {c.render(row)}
                  </TableCell>
                ))}
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">
          {isLoading
            ? "Memuat..."
            : total === 0
              ? "Tidak ada data"
              : `Halaman ${page} dari ${totalPages} · total ${total} data`}
        </span>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1 || isPending}
            onClick={() =>
              startTransition(() => {
                void setPage(String(page - 1));
              })
            }
          >
            Sebelumnya
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages || isPending}
            onClick={() =>
              startTransition(() => {
                void setPage(String(page + 1));
              })
            }
          >
            Berikutnya
          </Button>
        </div>
      </div>
    </div>
  );
}
