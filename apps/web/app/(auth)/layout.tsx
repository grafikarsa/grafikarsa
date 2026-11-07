import Link from "next/link";

import { NextStepGuard } from "@/components/auth/next-step-guard";

export default function AuthLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <div className="flex flex-1 items-start justify-center px-4 py-12 sm:items-center">
      <div className="w-full max-w-sm">
        <Link
          href="/"
          className="mb-8 block text-center text-2xl font-semibold tracking-tight"
        >
          Grafikarsa
        </Link>
        {children}
        <NextStepGuard />
      </div>
    </div>
  );
}
