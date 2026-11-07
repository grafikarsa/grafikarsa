import type { Metadata } from "next";

import { TahunAjaranView } from "@/components/admin/tahun-ajaran-view";

export const metadata: Metadata = {
  title: "Tahun Ajaran",
};

export default function AdminTahunAjaranPage() {
  return <TahunAjaranView />;
}
