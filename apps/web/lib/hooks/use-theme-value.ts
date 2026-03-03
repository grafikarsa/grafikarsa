'use client';

import { useState, useEffect } from 'react';

export function useThemeValue() {
  const [theme, setTheme] = useState<'light' | 'dark'>('light');
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    
    // Get initial theme
    const getTheme = () => {
      const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | null;
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      return savedTheme || (prefersDark ? 'dark' : 'light');
    };
    
    setTheme(getTheme());

    // Listen for theme changes via storage event (cross-tab)
    const handleStorage = (e: StorageEvent) => {
      if (e.key === 'theme') {
        setTheme((e.newValue as 'light' | 'dark') || 'light');
      }
    };

    // Listen for class changes on document (same tab)
    const observer = new MutationObserver(() => {
      const isDark = document.documentElement.classList.contains('dark');
      setTheme(isDark ? 'dark' : 'light');
    });

    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    });

    window.addEventListener('storage', handleStorage);

    return () => {
      window.removeEventListener('storage', handleStorage);
      observer.disconnect();
    };
  }, []);

  return { theme, mounted };
}
