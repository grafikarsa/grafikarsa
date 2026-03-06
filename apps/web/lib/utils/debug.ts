/**
 * Debug Utilities
 * 
 * ⚠️ DEVELOPMENT ONLY - NOT FOR PRODUCTION
 * 
 * This module provides debug utilities for testing empty states and UI components
 * during development. These features should NEVER be enabled in production.
 * 
 * Usage:
 * - Set NEXT_PUBLIC_DEBUG_MODE=true in .env.local
 * - Use isDebugMode() to check if debug mode is enabled
 * - Use getDebugEmptyState() to force empty state rendering
 */

/**
 * Check if debug mode is enabled
 * Only works in development environment
 */
export function isDebugMode(): boolean {
  if (process.env.NODE_ENV !== 'development') {
    return false;
  }
  return process.env.NEXT_PUBLIC_DEBUG_MODE === 'true';
}

/**
 * Get debug empty state flag
 * When enabled, forces components to render empty states
 * 
 * @returns true if debug mode is enabled and empty state should be shown
 */
export function getDebugEmptyState(): boolean {
  return isDebugMode();
}

/**
 * Debug banner component props
 */
export interface DebugBannerProps {
  message?: string;
}

/**
 * Get debug banner message
 */
export function getDebugBannerMessage(pageName: string): string {
  return `🔧 DEBUG MODE: Showing empty state for "${pageName}" page`;
}
