'use client';

import { Moon, Sun } from 'lucide-react';
import { Switch } from '@/components/ui/switch';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { useEffect, useState } from 'react';
import { cn } from '@/lib/utils';

export function ThemeToggle() {
  const [theme, setTheme] = useState<'light' | 'dark'>('light');
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | null;
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const initialTheme = savedTheme || (prefersDark ? 'dark' : 'light');
    setTheme(initialTheme);
    document.documentElement.classList.toggle('dark', initialTheme === 'dark');
  }, []);

  const toggleTheme = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    localStorage.setItem('theme', newTheme);
    document.documentElement.classList.toggle('dark', newTheme === 'dark');
  };

  if (!mounted) {
    return (
      <div className="flex h-6 w-11 items-center justify-center rounded-full bg-muted">
        <Sun className="h-3 w-3 text-muted-foreground" />
      </div>
    );
  }

  return (
    <TooltipProvider delayDuration={0}>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            onClick={toggleTheme}
            className="relative inline-flex h-6 w-11 items-center rounded-full transition-colors hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            aria-label="Toggle theme"
          >
            {/* Track background */}
            <span className="absolute inset-0 rounded-full bg-muted transition-colors duration-300" />
            
            {/* Thumb with icon and bouncy animation */}
            <span
              className={cn(
                'relative z-10 flex h-5 w-5 items-center justify-center rounded-full bg-background shadow-sm transition-all duration-300 ease-out',
                theme === 'dark' ? 'translate-x-5' : 'translate-x-0.5',
                'hover:scale-110 active:scale-95'
              )}
              style={{
                transition: 'transform 0.3s cubic-bezier(0.68, -0.55, 0.265, 1.55), scale 0.2s ease'
              }}
            >
              {theme === 'light' ? (
                <Sun className="h-3 w-3 text-foreground" />
              ) : (
                <Moon className="h-3 w-3 text-foreground" />
              )}
            </span>
          </button>
        </TooltipTrigger>
        <TooltipContent>
          {theme === 'light' ? 'Mode Gelap' : 'Mode Terang'}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
