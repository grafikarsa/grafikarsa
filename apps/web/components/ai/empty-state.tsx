'use client';

import { Sparkles, Lightbulb, Rocket, Target } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';

interface EmptyStateProps {
  onGetStarted?: () => void;
}

export function EmptyState({ onGetStarted }: EmptyStateProps) {
  return (
    <Card className="border-2 border-dashed p-8 md:p-12">
      <div className="mx-auto max-w-2xl text-center">
        {/* Icon */}
        <div className="mb-6 flex justify-center">
          <div className="relative">
            <div className="absolute inset-0 animate-ping rounded-full bg-primary/20" />
            <div className="relative flex h-20 w-20 items-center justify-center rounded-full bg-primary/10">
              <Sparkles className="h-10 w-10 text-primary" />
            </div>
          </div>
        </div>

        {/* Title & Description */}
        <h3 className="mb-3 text-2xl font-bold md:text-3xl">
          Belum Ada Ide Proyek
        </h3>
        <p className="mb-8 text-base text-muted-foreground md:text-lg">
          Mulai generate ide proyek portfolio yang sesuai dengan jurusan dan minatmu menggunakan AI
        </p>

        {/* CTA Button */}
        {onGetStarted && (
          <Button onClick={onGetStarted} size="lg" className="mb-8 gap-2">
            <Sparkles className="h-5 w-5" />
            Mulai Generate Ide
          </Button>
        )}

        {/* Features */}
        <div className="mt-8 grid gap-6 text-left sm:grid-cols-3">
          <div className="space-y-2">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Lightbulb className="h-5 w-5 text-primary" />
            </div>
            <h4 className="font-semibold">Ide Kreatif</h4>
            <p className="text-sm text-muted-foreground">
              AI akan membuat 5 ide proyek unik sesuai minatmu
            </p>
          </div>

          <div className="space-y-2">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Target className="h-5 w-5 text-primary" />
            </div>
            <h4 className="font-semibold">Sesuai Jurusan</h4>
            <p className="text-sm text-muted-foreground">
              Proyek disesuaikan dengan jurusan dan tingkat keahlianmu
            </p>
          </div>

          <div className="space-y-2">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Rocket className="h-5 w-5 text-primary" />
            </div>
            <h4 className="font-semibold">Siap Dikerjakan</h4>
            <p className="text-sm text-muted-foreground">
              Lengkap dengan teknologi, estimasi waktu, dan tujuan pembelajaran
            </p>
          </div>
        </div>

        {/* Example Preview */}
        <div className="mt-8 rounded-lg border bg-muted/30 p-4 text-left">
          <p className="mb-2 text-xs font-medium text-muted-foreground">Contoh Ide:</p>
          <p className="font-semibold">Sistem Manajemen Perpustakaan Digital</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Aplikasi web untuk mengelola koleksi buku, peminjaman, dan pengembalian dengan fitur pencarian dan notifikasi otomatis.
          </p>
          <div className="mt-3 flex flex-wrap gap-2">
            <span className="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary">
              React
            </span>
            <span className="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary">
              Node.js
            </span>
            <span className="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary">
              PostgreSQL
            </span>
          </div>
        </div>
      </div>
    </Card>
  );
}
