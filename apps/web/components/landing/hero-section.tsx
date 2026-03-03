'use client';

import { useState } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { useQuery } from '@tanstack/react-query';
import { motion } from 'motion/react';
import { Button } from '@/components/ui/button';
import { TextEffect } from '@/components/ui/text-effect';
import { AnimatedGroup } from '@/components/ui/animated-group';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { topApi } from '@/lib/api/public';
import { cn } from '@/lib/utils';

const transitionVariants = {
  item: {
    hidden: {
      opacity: 0,
      filter: 'blur(12px)',
      y: 12,
    },
    visible: {
      opacity: 1,
      filter: 'blur(0px)',
      y: 0,
      transition: {
        type: 'spring',
        bounce: 0.3,
        duration: 1.5,
      },
    },
  },
};

// Floating card positions with rotation - adjusted to prevent clipping
const floatingPositions = [
  { top: '18%', left: '5%', rotate: -8, floatY: 12, duration: 5 },
  { top: '52%', left: '6%', rotate: 6, floatY: 10, duration: 6 },
  { top: '20%', right: '5%', rotate: 10, floatY: 14, duration: 5.5 },
  { top: '54%', right: '6%', rotate: -6, floatY: 11, duration: 6.5 },
];

// Placeholder data when no projects exist - using Unsplash stock images
const placeholderProjects = [
  { id: '1', judul: 'Desain Grafis', image: 'https://images.unsplash.com/photo-1626785774573-4b799315345d?w=400&q=80' },
  { id: '2', judul: 'Fotografi', image: 'https://images.unsplash.com/photo-1554080353-a576cf803bda?w=400&q=80' },
  { id: '3', judul: 'Ilustrasi Digital', image: 'https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=400&q=80' },
  { id: '4', judul: 'UI/UX Design', image: 'https://images.unsplash.com/photo-1561070791-2526d30994b5?w=400&q=80' },
];

interface FloatingPortfolioCardProps {
  project: {
    id: string;
    judul: string;
    slug: string;
    thumbnail_url: string | null;
    username: string;
    user_nama: string;
    user_avatar: string | null;
  };
  position: (typeof floatingPositions)[0];
  index: number;
}

function FloatingPortfolioCard({ project, position, index }: FloatingPortfolioCardProps) {
  const [isHovered, setIsHovered] = useState(false);

  return (
    <motion.div
      initial={{ opacity: 0, y: 30, rotate: position.rotate }}
      animate={{
        opacity: 1,
        y: [0, -position.floatY, 0],
        rotate: position.rotate,
      }}
      transition={{
        opacity: { duration: 0.6, delay: index * 0.15 },
        y: {
          duration: position.duration,
          repeat: Infinity,
          ease: 'easeInOut',
          delay: index * 0.3,
        },
      }}
      whileHover={{
        scale: 1.08,
        rotate: 0,
        zIndex: 50,
        transition: { duration: 0.3 },
      }}
      onHoverStart={() => setIsHovered(true)}
      onHoverEnd={() => setIsHovered(false)}
      className={cn(
        'absolute hidden xl:block',
        'w-40 rounded-xl border bg-card shadow-xl overflow-hidden cursor-pointer'
      )}
      style={{
        top: position.top,
        left: position.left,
        right: position.right,
      }}
    >
      <Link href={`/${project.username}/${project.slug}`} className="block">
        {/* Thumbnail */}
        <div className="relative aspect-[4/3] w-full bg-muted overflow-hidden">
          {project.thumbnail_url ? (
            <Image
              src={project.thumbnail_url}
              alt={project.judul}
              fill
              className={cn('object-cover transition-transform duration-300', isHovered && 'scale-110')}
              sizes="200px"
            />
          ) : (
            <div className="flex h-full items-center justify-center text-xs text-muted-foreground">No Image</div>
          )}
        </div>

        {/* Content */}
        <div className="p-2.5">
          <h4 className="text-xs font-medium leading-tight line-clamp-1">{project.judul}</h4>
          <div className="mt-1.5 flex items-center gap-1.5">
            <Avatar className="h-5 w-5">
              <AvatarImage src={project.user_avatar || undefined} alt={project.user_nama} />
              <AvatarFallback className="text-[8px]">{project.user_nama?.charAt(0)}</AvatarFallback>
            </Avatar>
            <span className="text-[10px] text-muted-foreground truncate">{project.user_nama}</span>
          </div>
        </div>
      </Link>
    </motion.div>
  );
}

// Placeholder card when no real projects exist
function PlaceholderCard({
  placeholder,
  position,
  index,
}: {
  placeholder: (typeof placeholderProjects)[0];
  position: (typeof floatingPositions)[0];
  index: number;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 30, rotate: position.rotate }}
      animate={{
        opacity: 1,
        y: [0, -position.floatY, 0],
        rotate: position.rotate,
      }}
      transition={{
        opacity: { duration: 0.6, delay: index * 0.15 },
        y: {
          duration: position.duration,
          repeat: Infinity,
          ease: 'easeInOut',
          delay: index * 0.3,
        },
      }}
      className={cn('absolute hidden xl:block', 'w-40 rounded-xl border bg-card shadow-xl overflow-hidden')}
      style={{
        top: position.top,
        left: position.left,
        right: position.right,
      }}
    >
      {/* Stock Image Thumbnail */}
      <div className="relative aspect-[4/3] w-full bg-muted overflow-hidden">
        <Image src={placeholder.image} alt={placeholder.judul} fill className="object-cover" sizes="200px" />
      </div>

      {/* Content */}
      <div className="p-2.5">
        <h4 className="text-xs font-medium leading-tight">{placeholder.judul}</h4>
        <div className="mt-1.5 flex items-center gap-1.5">
          <div className="h-5 w-5 rounded-full bg-muted" />
          <span className="text-[10px] text-muted-foreground">Karya Siswa</span>
        </div>
      </div>
    </motion.div>
  );
}

export function HeroSection() {
  const { data } = useQuery({
    queryKey: ['top-projects-hero'],
    queryFn: () => topApi.getTopProjects(),
    staleTime: 5 * 60 * 1000,
  });

  const projects = data?.data || [];

  return (
    <main className="overflow-hidden">
      {/* Background gradients */}
      <div
        aria-hidden
        className="absolute inset-0 isolate hidden opacity-65 contain-strict lg:block"
      >
        <div className="w-140 h-320 -translate-y-87.5 absolute left-0 top-0 -rotate-45 rounded-full bg-[radial-gradient(68.54%_68.72%_at_55.02%_31.46%,hsla(0,0%,85%,.08)_0,hsla(0,0%,55%,.02)_50%,hsla(0,0%,45%,0)_80%)]" />
        <div className="h-320 absolute left-0 top-0 w-60 -rotate-45 rounded-full bg-[radial-gradient(50%_50%_at_50%_50%,hsla(0,0%,85%,.06)_0,hsla(0,0%,45%,.02)_80%,transparent_100%)] [translate:5%_-50%]" />
        <div className="h-320 -translate-y-87.5 absolute left-0 top-0 w-60 -rotate-45 bg-[radial-gradient(50%_50%_at_50%_50%,hsla(0,0%,85%,.04)_0,hsla(0,0%,45%,.02)_80%,transparent_100%)]" />
      </div>

      <section className="relative flex min-h-screen items-center justify-center">
        {/* Floating Portfolio Cards - show real projects or placeholders */}
        {projects.length > 0
          ? floatingPositions.map((position, index) => {
              const project = projects[index % projects.length];
              if (!project) return null;
              return <FloatingPortfolioCard key={index} project={project} position={position} index={index} />;
            })
          : floatingPositions.map((position, index) => (
              <PlaceholderCard key={index} placeholder={placeholderProjects[index]} position={position} index={index} />
            ))}

        <div className="relative z-10 w-full">
          <div className="mx-auto max-w-4xl px-6">
            <div className="flex flex-col items-center text-center">
              <TextEffect
                preset="fade-in-blur"
                speedSegment={0.3}
                as="h1"
                className="text-balance text-4xl font-medium tracking-tight sm:text-5xl md:text-6xl"
              >
                Platform Portofolio Digital SMKN 4 Malang
              </TextEffect>
              <TextEffect
                per="line"
                preset="fade-in-blur"
                speedSegment={0.3}
                delay={0.5}
                as="p"
                className="mt-6 max-w-2xl text-pretty text-lg text-muted-foreground"
              >
                Jelajahi karya siswa dan alumni. Temukan inspirasi dari berbagai portofolio kreatif dan inovatif.
              </TextEffect>

              <AnimatedGroup
                variants={{
                  container: {
                    visible: {
                      transition: {
                        staggerChildren: 0.05,
                        delayChildren: 0.75,
                      },
                    },
                  },
                  ...transitionVariants,
                }}
                className="mt-10 flex flex-col items-center gap-4 sm:flex-row"
              >
                <Button asChild size="lg" className="rounded-full px-8">
                  <Link href="/portfolios">Jelajahi Portofolio</Link>
                </Button>
                <Button asChild size="lg" variant="outline" className="rounded-full px-8">
                  <Link href="/users">Lihat Siswa & Alumni</Link>
                </Button>
              </AnimatedGroup>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
