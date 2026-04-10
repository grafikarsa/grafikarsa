import { socialPlatformConfigs } from '@/lib/constants/social-platforms';
import { SocialPlatform } from '@/lib/types';
import { cn } from '@/lib/utils';

interface PlatformIconProps {
  platform: SocialPlatform;
  className?: string;
  size?: 'sm' | 'md' | 'lg';
}

const sizeClasses = {
  sm: 'h-3.5 w-3.5',
  md: 'h-4 w-4',
  lg: 'h-5 w-5',
};

export function PlatformIcon({ platform, className, size = 'md' }: PlatformIconProps) {
  const config = socialPlatformConfigs[platform];
  if (!config) return null;

  const isLucideIcon = typeof config.icon !== 'string';
  const sizeClass = sizeClasses[size];

  if (isLucideIcon) {
    // Render Lucide icon component (for personal_website)
    const IconComponent = config.icon as React.ComponentType<{ className?: string }>;
    return <IconComponent className={cn(sizeClass, className)} />;
  }

  // Render SVG string with brand color
  return (
    <span
      className={cn(
        'inline-flex flex-shrink-0 [&>svg]:h-full [&>svg]:w-full [&>svg]:fill-current',
        sizeClass,
        className
      )}
      style={{ color: config.brandColor }}
      dangerouslySetInnerHTML={{ __html: config.icon as string }}
    />
  );
}
