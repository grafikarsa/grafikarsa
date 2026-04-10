import { SocialLink } from '@/lib/types';
import { socialPlatformConfigs, extractSocialHandle } from '@/lib/constants/social-platforms';
import { cn } from '@/lib/utils';
import { PlatformIcon } from './platform-icon';

interface SocialChipProps {
  link: SocialLink;
  className?: string;
}

/**
 * Convert hex color to RGB values
 */
function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result
    ? {
        r: parseInt(result[1], 16),
        g: parseInt(result[2], 16),
        b: parseInt(result[3], 16),
      }
    : null;
}

export function SocialChip({ link, className }: SocialChipProps) {
  const config = socialPlatformConfigs[link.platform];
  if (!config) return null;

  const handle = extractSocialHandle(link.url, link.platform);
  const brandColor = config.brandColor;

  // Convert brand color to RGB for opacity
  const rgb = hexToRgb(brandColor);
  const bgColor = rgb ? `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.1)` : `${brandColor}1A`;

  return (
    <a
      href={link.url}
      target="_blank"
      rel="noopener noreferrer"
      className={cn(
        'group inline-flex items-center gap-2 rounded-full px-3 py-1.5 text-sm font-medium transition-all hover:scale-105 hover:shadow-sm',
        className
      )}
      style={{
        backgroundColor: bgColor,
        color: brandColor,
      }}
      title={`${config.label}: ${handle}`}
    >
      <PlatformIcon platform={link.platform} size="md" />
      <span className="max-w-[150px] truncate">{handle}</span>
    </a>
  );
}
