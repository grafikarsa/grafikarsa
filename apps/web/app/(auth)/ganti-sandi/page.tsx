import type { Metadata } from "next";

import { ChangePasswordForm } from "@/components/auth/change-password-form";

export const metadata: Metadata = {
  title: "Ganti sandi",
};

export default function GantiSandiPage() {
  return <ChangePasswordForm />;
}
