'use client';

import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { FeedAlgorithm } from '@/lib/types';
import { Sparkles, Clock, Users } from 'lucide-react';

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
  return (
    <Tabs value={value} onValueChange={(v) => onChange(v as FeedAlgorithm)}>
      <TabsList className="w-full grid grid-cols-3">
        <TabsTrigger value="smart" disabled={!isAuthenticated} className="gap-1.5">
          <Sparkles className="h-4 w-4" />
          <span className="hidden sm:inline">FYP</span>
          <span className="sm:hidden">FYP</span>
        </TabsTrigger>
        <TabsTrigger value="recent" className="gap-1.5">
          <Clock className="h-4 w-4" />
          <span>Terbaru</span>
        </TabsTrigger>
        <TabsTrigger value="following" disabled={!isAuthenticated} className="gap-1.5">
          <Users className="h-4 w-4" />
          <span>Following</span>
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}
