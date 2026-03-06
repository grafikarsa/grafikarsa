/**
 * Debug Banner Component
 * 
 * ⚠️ DEVELOPMENT ONLY - NOT FOR PRODUCTION
 * 
 * Displays a warning banner when debug mode is active.
 * This component only renders in development environment.
 */

'use client';

import { AlertTriangle } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { isDebugMode } from '@/lib/utils/debug';

interface DebugBannerProps {
  pageName: string;
}

export function DebugBanner({ pageName }: DebugBannerProps) {
  if (!isDebugMode()) {
    return null;
  }

  return (
    <Alert className="mb-4 border-amber-500 bg-amber-50 dark:bg-amber-950/20">
      <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400" />
      <AlertDescription className="text-sm text-amber-800 dark:text-amber-300">
        <strong>🔧 DEBUG MODE ACTIVE:</strong> Showing empty state for &quot;{pageName}&quot; page. 
        This is for development testing only and will not appear in production.
      </AlertDescription>
    </Alert>
  );
}
