import type { Metadata } from "next";

import { KelasView } from "@/components/admin/kelas-view";

export const metadata: Metadata = {
  title: "Kelas",
};

export default function AdminKelasPage() {
  return <KelasView />;
}
