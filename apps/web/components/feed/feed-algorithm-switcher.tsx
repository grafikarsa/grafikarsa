'use client';

import { FeedAlgorithm } from '@/lib/types';
import { Sparkles, Clock, Users } from 'lucide-react';
import { cn } from '@/lib/utils';
import { motion } from 'framer-motion';

interface FeedAlgorithmSwitcherProps {
  value: FeedAlgorithm;
  onChange: (algorithm: FeedAlgorithm) => void;
  isAuthenticated: boolean;
}

export function FeedAlgorithmSwitcher({
  value,
  onChange,
  isAuthenticated,
}: FeedAlgorithmSwitcherProps) {
  const tabs = [
    {
      value: 'smart' as FeedAlgorithm,
      label: 'FYP',
      icon: Sparkles,
      disabled: !isAuthenticated,
    },
    {
      value: 'recent' as FeedAlgorithm,
      label: 'Terbaru',
      icon: Clock,
      disabled: false,
    },
    {
      value: 'following' as FeedAlgorithm,
      label: 'Following',
      icon: Users,
      disabled: !isAuthenticated,
    },
  ];

  return (
    <div className="flex relative overflow-hidden">
      {tabs.map((tab) => {
        const Icon = tab.icon;
        const isActive = value === tab.value;
        
        return (
          <button
            key={tab.value}
            onClick={() => !tab.disabled && onChange(tab.value)}
            disabled={tab.disabled}
            className={cn(
              'flex-1 flex items-center justify-center gap-1.5 md:gap-2 px-3 md:px-4 py-3 md:py-4 text-xs md:text-sm font-medium transition-colors duration-200 relative z-10',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
              'active:scale-95 transition-transform',
              isActive
                ? 'text-foreground'
                : 'text-muted-foreground hover:text-foreground/80',
              tab.disabled 
                ? 'opacity-50 cursor-not-allowed' 
                : 'cursor-pointer'
            )}
          >
            <Icon className="h-3.5 w-3.5 md:h-4 md:w-4 shrink-0" />
            <span className="whitespace-nowrap">{tab.label}</span>
          </button>
        );
      })}
      
      {/* Animated underline indicator */}
      <motion.div
        className="absolute bottom-0 h-0.5 md:h-1 bg-primary rounded-t-full"
        initial={false}
        animate={{
          left: `${tabs.findIndex(t => t.value === value) * (100 / tabs.length)}%`,
          width: `${100 / tabs.length}%`,
        }}
        transition={{
          type: 'spring',
          stiffness: 300,
          damping: 30,
          mass: 0.8,
        }}
      />
    </div>
  );
}
