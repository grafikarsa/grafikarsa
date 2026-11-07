import type { Metadata } from "next";

import { RecoverPasswordForm } from "@/components/auth/recover-password-form";

export const metadata: Metadata = {
  title: "Lupa sandi",
};

export default function LupaSandiPage() {
  return <RecoverPasswordForm />;
}
