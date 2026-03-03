'use client';

import { cn } from '@/lib/utils';
import {
  FloatingAvatars,
  PortfolioStack,
  LikeAnimation,
  UploadAnimation,
  FeedScrollAnimation,
} from './bento-animations';

interface BentoItem {
  title: string;
  description: string;
  className: string;
  animation: React.ReactNode;
}

const bentoItems: BentoItem[] = [
  {
    title: 'Temukan Karya Kreatif',
    description: 'Jelajahi portofolio siswa dan alumni yang unik dan inspiratif.',
    className: 'md:col-span-2 md:row-span-2',
    animation: <PortfolioStack />,
  },
  {
    title: 'Terhubung dengan Komunitas',
    description: 'Ikuti teman, alumni, dan temukan kreator lain.',
    className: 'md:col-span-1 md:row-span-2',
    animation: <FloatingAvatars />,
  },
  {
    title: 'Upload Mudah',
    description: 'Unggah karya dengan cepat dan tampilkan gaya kreatifmu.',
    className: 'md:col-span-1 md:row-span-1',
    animation: <UploadAnimation />,
  },
  {
    title: 'Feed & Explore',
    description: 'Pantau karya terbaru setiap hari.',
    className: 'md:col-span-1 md:row-span-1',
    animation: <FeedScrollAnimation />,
  },
  {
    title: 'Apresiasi Karya',
    description: 'Like dan dukung karya teman-temanmu.',
    className: 'md:col-span-1 md:row-span-1',
    animation: <LikeAnimation />,
  },
];

export function AboutSection() {
  return (
    <section className="py-20">
      <div className="mx-auto max-w-6xl px-6">
        {/* Header */}
        <div className="mb-12 text-center">
          <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">
            Tentang Grafikarsa
          </h2>
          <p className="mx-auto mt-4 max-w-2xl text-lg text-muted-foreground">
            Platform untuk menampilkan portofolio dan membangun jaringan komunitas warga SMKN 4 Malang
          </p>
        </div>

        {/* Bento Grid - 5 items layout */}
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3 auto-rows-[200px]">
          {bentoItems.map((item) => (
            <div
              key={item.title}
              className={cn(
                'group relative overflow-hidden rounded-2xl border bg-card transition-all duration-300 hover:shadow-lg hover:-translate-y-0.5',
                item.className
              )}
            >
              {/* Animation - z-0 to stay behind text */}
              <div className="absolute inset-0 z-0">
                {item.animation}
              </div>

              {/* Content Overlay - z-10 to stay in front */}
              <div className="absolute inset-x-0 bottom-0 z-10 bg-gradient-to-t from-card via-card/90 to-transparent p-4 pt-12 pointer-events-none">
                <h3 className="font-semibold text-base">{item.title}</h3>
                <p className="mt-0.5 text-sm text-muted-foreground line-clamp-2">
                  {item.description}
                </p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
