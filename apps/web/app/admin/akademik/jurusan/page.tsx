import type { Metadata } from "next";

import { JurusanView } from "@/components/admin/jurusan-view";

export const metadata: Metadata = {
  title: "Jurusan",
};

export default function AdminJurusanPage() {
  return <JurusanView />;
}
