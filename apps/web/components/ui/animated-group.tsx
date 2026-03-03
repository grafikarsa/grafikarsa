'use client';

import { cn } from '@/lib/utils';
import React from 'react';

interface AnimatedGroupProps {
  children: React.ReactNode;
  className?: string;
  variants?: unknown;
}

export function AnimatedGroup({ children, className }: AnimatedGroupProps) {
  return (
    <div className={cn('animate-in fade-in slide-in-from-bottom-4 duration-700', className)}>
      {children}
    </div>
  );
}
