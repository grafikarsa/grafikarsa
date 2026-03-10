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
      <div className="flex h-9 w-14 items-center justify-center rounded-full bg-muted">
        <Sun className="h-4 w-4 text-muted-foreground" />
      </div>
    );
  }

  return (
    <TooltipProvider delayDuration={0}>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            onClick={toggleTheme}
            className="relative inline-flex h-9 w-14 items-center rounded-full bg-muted transition-colors hover:bg-muted/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            aria-label="Toggle theme"
          >
            {/* Track background with gradient */}
            <span
              className={cn(
                'absolute inset-0 rounded-full transition-colors duration-300',
                theme === 'dark' ? 'bg-slate-800' : 'bg-amber-100'
              )}
            />
            
            {/* Thumb with icon and bouncy animation */}
            <span
              className={cn(
                'relative z-10 flex h-7 w-7 items-center justify-center rounded-full bg-white shadow-md transition-all duration-300 ease-out',
                theme === 'dark' ? 'translate-x-6' : 'translate-x-1',
                'hover:scale-110 active:scale-95'
              )}
              style={{
                transition: 'transform 0.3s cubic-bezier(0.68, -0.55, 0.265, 1.55), scale 0.2s ease'
              }}
            >
              {theme === 'light' ? (
                <Sun className="h-4 w-4 text-amber-500" />
              ) : (
                <Moon className="h-4 w-4 text-slate-700" />
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
