"use client";

import { useEffect } from "react";

import { Button } from "@/components/ui/button";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="mx-auto flex min-h-[60vh] w-full max-w-6xl flex-col items-center justify-center gap-4 px-4 py-16 text-center">
      <h1 className="text-3xl font-semibold tracking-tight md:text-4xl">
        Terjadi kesalahan
      </h1>
      <p className="max-w-md text-muted-foreground">
        Coba muat ulang halaman.
      </p>
      <Button onClick={() => reset()}>Coba lagi</Button>
    </div>
  );
}
