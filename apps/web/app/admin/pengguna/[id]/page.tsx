import type { Metadata } from "next";

import { UserDetailView } from "@/components/admin/users/user-detail-view";

export const metadata: Metadata = {
  title: "Detail pengguna",
};

export default async function AdminUserDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  return <UserDetailView id={id} />;
}
