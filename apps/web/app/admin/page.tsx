import type { Metadata } from "next";

import { AdminDashboard } from "@/components/admin/admin-dashboard";

export const metadata: Metadata = {
  title: { absolute: "Dashboard · Admin · Grafikarsa" },
};

export default function AdminDashboardPage() {
  return <AdminDashboard />;
}
