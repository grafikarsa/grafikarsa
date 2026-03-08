'use client';

import { useRef } from 'react';
import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { Search, LogIn } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PortfolioCard } from '@/components/portfolio/portfolio-card';
import { topApi } from '@/lib/api/public';
import { GeistPixelCircle } from 'geist/font/pixel';
import { cn } from '@/lib/utils';

export function NewHeroSection() {
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['top-projects-hero'],
    queryFn: () => topApi.getTopProjects(),
    staleTime: 5 * 60 * 1000,
  });

  const projects = data?.data || [];

  // Duplicate projects 2x for seamless infinite scroll (reduced from 4x)
  const duplicatedProjects = projects.length > 0 ? [...projects, ...projects] : [];

  return (
    <section className="relative min-h-screen flex flex-col items-center justify-center overflow-hidden bg-background py-12">
      {/* Hero Content */}
      <div className="relative z-10 flex flex-col items-center text-center px-6 mb-12">
        {/* GRAFIKARSA Title with Geist Pixel Circle */}
        <h1 
          className={cn(
            GeistPixelCircle.className,
            "text-5xl xs:text-6xl sm:text-7xl md:text-8xl lg:text-9xl font-bold tracking-wider mb-4"
          )}
        >
          GRAFIKARSA
        </h1>

        {/* Subtitle */}
        <p className="text-base sm:text-lg md:text-xl text-muted-foreground max-w-2xl mb-6">
          Platform Portfolio Digital dan Social Network SMKN 4 Malang
        </p>

        {/* CTA Buttons */}
        <div className="flex flex-col sm:flex-row gap-4">
          <Button asChild size="lg" className="rounded-full px-8">
            <Link href="/portfolios">
              <Search className="mr-2 h-4 w-4" />
              Jelajahi Portfolio
            </Link>
          </Button>
          <Button asChild size="lg" variant="outline" className="rounded-full px-8">
            <Link href="/login">
              <LogIn className="mr-2 h-4 w-4" />
              Login
            </Link>
          </Button>
        </div>
      </div>

      {/* Infinite Horizontal Scroll Portfolio Cards */}
      {isLoading && (
        <div className="text-center text-muted-foreground py-12">
          <p>Memuat portfolio...</p>
        </div>
      )}

      {!isLoading && projects.length > 0 && (
        <div className="relative w-full overflow-hidden">
          {/* Left Fade */}
          <div className="absolute left-0 top-0 bottom-0 w-32 md:w-48 bg-gradient-to-r from-background to-transparent z-10 pointer-events-none" />
          
          {/* Right Fade */}
          <div className="absolute right-0 top-0 bottom-0 w-32 md:w-48 bg-gradient-to-l from-background to-transparent z-10 pointer-events-none" />

          {/* Scrolling Container - Using CSS Animation */}
          <div className="py-6 px-6">
            <div 
              ref={scrollContainerRef}
              className="flex gap-6 animate-scroll-horizontal"
              role="region"
              aria-label="Portfolio showcase"
              tabIndex={0}
              onFocus={(e) => e.currentTarget.style.animationPlayState = 'paused'}
              onBlur={(e) => e.currentTarget.style.animationPlayState = 'running'}
            >
              {duplicatedProjects.map((project, index) => (
                <div key={`${project.id}-${index}`} className="flex-shrink-0">
                  <PortfolioCard 
                    portfolio={{
                      id: project.id,
                      judul: project.judul,
                      slug: project.slug,
                      thumbnail_url: project.thumbnail_url ?? undefined,
                      published_at: undefined,
                      created_at: new Date().toISOString(),
                      updated_at: new Date().toISOString(),
                      user: {
                        id: project.username,
                        username: project.username,
                        nama: project.user_nama,
                        avatar_url: project.user_avatar ?? undefined,
                        banner_url: undefined,
                        role: 'student' as const,
                        kelas: undefined,
                        jurusan: undefined,
                        tahun_masuk: undefined,
                        tahun_lulus: undefined,
                      },
                      tags: [],
                      status: 'published' as const,
                    }}
                    username={project.username}
                  />
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {!isLoading && projects.length === 0 && (
        <div className="text-center text-muted-foreground py-12">
          <p>Belum ada portfolio untuk ditampilkan</p>
        </div>
      )}
    </section>
  );
}
