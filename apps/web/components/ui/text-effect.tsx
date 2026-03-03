'use client';

import { cn } from '@/lib/utils';
import React from 'react';

interface TextEffectProps {
  children: React.ReactNode;
  as?: 'h1' | 'h2' | 'h3' | 'h4' | 'p' | 'span' | 'div';
  className?: string;
  preset?: 'fade-in-blur' | 'fade-in' | 'slide-up';
  per?: 'word' | 'char' | 'line';
  speedSegment?: number;
  delay?: number;
}

export function TextEffect({
  children,
  as: Component = 'p',
  className,
}: TextEffectProps) {
  return (
    <Component className={cn('animate-in fade-in slide-in-from-bottom-4 duration-700', className)}>
      {children}
    </Component>
  );
}
