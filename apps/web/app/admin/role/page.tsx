import type { Metadata } from "next";

import { RolesView } from "@/components/admin/roles/roles-view";

export const metadata: Metadata = {
  title: "Role",
};

export default function AdminRolesPage() {
  return <RolesView />;
}
