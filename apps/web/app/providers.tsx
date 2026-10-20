"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "next-themes";
import { NuqsAdapter } from "nuqs/adapters/next/app";
import { useState } from "react";
import { Toaster } from "sonner";
import { AuthProvider } from "@/components/providers/auth-provider";

export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { staleTime: 60_000, gcTime: 5 * 60_000, refetchOnWindowFocus: false, retry: 1 },
        },
      }),
  );
  return (
    // NuqsAdapter: nuqs butuh adapter framework untuk membaca & menulis
    // state di URL. Tanpa ini, halaman yang memakai useQueryState gagal
    // saat prerender.
    <NuqsAdapter>
      <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
        <QueryClientProvider client={client}>
          <AuthProvider>{children}</AuthProvider>
          <Toaster position="top-center" richColors />
        </QueryClientProvider>
      </ThemeProvider>
    </NuqsAdapter>
  );
}
