import type { Metadata } from "next";

import { ActivationFlow } from "@/components/auth/activation-flow";

export const metadata: Metadata = {
  title: "Aktivasi akun",
};

export default function AktivasiPage() {
  return <ActivationFlow />;
}
