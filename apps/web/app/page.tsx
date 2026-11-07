import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

export default function Home() {
  return (
    <div className="mx-auto w-full max-w-6xl px-4">
      <section className="flex flex-col items-center gap-5 py-16 text-center md:py-24">
        <h1 className="text-4xl font-semibold tracking-tight md:text-5xl">
          Grafikarsa
        </h1>
        <p className="max-w-xl text-lg text-muted-foreground">
          Karya, tulisan, dan cerita warga SMKN 4 Malang.
        </p>
        <div className="flex flex-wrap items-center justify-center gap-3">
          <Button asChild>
            <Link href="/karya">Jelajahi Karya</Link>
          </Button>
          <Button asChild variant="outline">
            <Link href="/blog">Baca Blog</Link>
          </Button>
        </div>
      </section>

      <section aria-label="Pratinjau karya" className="pb-16">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, index) => (
            <div key={index} className="rounded-xl border p-3">
              <Skeleton className="aspect-video w-full rounded-lg" />
              <Skeleton className="mt-4 h-4 w-3/4" />
              <Skeleton className="mt-2 h-4 w-1/2" />
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
