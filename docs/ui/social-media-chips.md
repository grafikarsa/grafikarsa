# Social Media Chips

## Overview

Social media badges di halaman profil user menggunakan komponen chip yang menampilkan icon platform dan username/handle user dengan warna brand masing-masing platform.

## Design Specifications

### Visual Style
- **Shape:** Rounded pill/chip (fully rounded corners)
- **Border:** No border
- **Background:** Platform brand color with 10% opacity
- **Text Color:** Platform brand color (full opacity)
- **Icon:** Platform icon from Lucide React
- **Typography:** 14px (text-sm), font-medium

### Interaction
- **Hover:** Scale up to 105% with smooth transition
- **Hover (optional):** Show external link icon
- **Click:** Opens social media profile in new tab
- **Cursor:** Pointer

### Spacing
- **Padding:** 12px horizontal, 6px vertical (px-3 py-1.5)
- **Gap:** 8px between icon and text (gap-2)
- **Margin:** 8px between chips (gap-2)

## Platform Brand Colors

| Platform | Brand Color (Light) | Brand Color (Dark) | Example |
|----------|---------------------|-------------------|---------|
| Facebook | #0866FF | #5B9FFF | Blue |
| Instagram | #FF0069 | #FF5C96 | Pink/Red gradient |
| GitHub | #181717 | #E6E6E6 | Black → Light gray |
| LinkedIn | #0A66C2 | #4A9EE6 | Blue |
| Twitter/X | #000000 | #E6E6E6 | Black → Light gray |
| YouTube | #FF0000 | #FF5C5C | Red |
| TikTok | #000000 | #E6E6E6 | Black → Light gray |
| Behance | #1769FF | #6B9FFF | Blue |
| Dribbble | #EA4C89 | #FF8CB8 | Pink |
| Threads | #000000 | #E6E6E6 | Black → Light gray |
| Bluesky | #1185FE | #5CA8FF | Blue |
| Medium | #000000 | #E6E6E6 | Black → Light gray |
| GitLab | #FC6D26 | #FF9A6B | Orange |
| Personal Website | #6366F1 | #A5B4FC | Indigo |

## Username Extraction

Komponen otomatis mengekstrak username/handle dari URL:

### Format Username per Platform

- **Instagram:** `@username` (from instagram.com/username)
- **Twitter/X:** `@username` (from x.com/username)
- **TikTok:** `@username` (from tiktok.com/@username)
- **Threads:** `@username` (from threads.net/@username)
- **GitHub:** `username` (from github.com/username)
- **GitLab:** `username` (from gitlab.com/username)
- **LinkedIn:** `username` (from linkedin.com/in/username)
- **YouTube:** `@channel` (from youtube.com/@channel)
- **Behance:** `username` (from behance.net/username)
- **Dribbble:** `username` (from dribbble.com/username)
- **Medium:** `@username` (from medium.com/@username)
- **Bluesky:** `username` (from bsky.app/profile/username.bsky.social)
- **Facebook:** `username` (from facebook.com/username)
- **Personal Website:** `domain.com` (domain name)

## Component Usage

### Basic Usage

```tsx
import { SocialChip } from '@/components/user/social-chip';

<SocialChip 
  link={{ 
    platform: 'instagram', 
    url: 'https://instagram.com/johndoe' 
  }} 
/>
```

### With External Icon

```tsx
<SocialChip 
  link={{ 
    platform: 'github', 
    url: 'https://github.com/johndoe' 
  }}
  showExternalIcon={true}
/>
```

### Multiple Chips

```tsx
<div className="flex flex-wrap gap-2">
  {user.social_links.map((link) => (
    <SocialChip key={link.platform} link={link} />
  ))}
</div>
```

## Implementation Files

- **Component:** `apps/web/components/user/social-chip.tsx`
- **Configuration:** `apps/web/lib/constants/social-platforms.ts`
- **Types:** `apps/web/lib/types/user.ts`

## Accessibility

- ✅ Semantic HTML (`<a>` tag)
- ✅ External link attributes (`target="_blank"` with `rel="noopener noreferrer"`)
- ✅ Descriptive title attribute
- ✅ Keyboard accessible
- ✅ Screen reader friendly
- ✅ Color contrast compliant (brand colors on light backgrounds)

## Examples

### Instagram Chip
```
[Instagram Icon] @johndoe
Light: Background: rgba(228, 64, 95, 0.1), Text: #FF0069
Dark:  Background: rgba(255, 92, 150, 0.15), Text: #FF5C96
```

### GitHub Chip
```
[GitHub Icon] johndoe
Light: Background: rgba(24, 23, 23, 0.1), Text: #181717
Dark:  Background: rgba(230, 230, 230, 0.15), Text: #E6E6E6
```

### Personal Website Chip
```
[Globe Icon] johndoe.com
Light: Background: rgba(99, 102, 241, 0.1), Text: #6366F1
Dark:  Background: rgba(165, 180, 252, 0.15), Text: #A5B4FC
```

## Migration Notes

### Before (Old Design)
- Circular icon buttons
- No username/handle shown
- Generic muted colors
- Hover changes background to muted

### After (New Design)
- Pill-shaped chips
- Username/handle displayed
- Platform brand colors
- Hover scales up the chip

## Future Enhancements

- [ ] Add more platform icons (custom SVGs for TikTok, Behance, etc.)
- [ ] Support for custom platform colors in admin
- [ ] Verified badge for official accounts
- [ ] Click analytics tracking
- [ ] Copy username on right-click
