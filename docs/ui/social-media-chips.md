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

| Platform | Brand Color | Example |
|----------|-------------|---------|
| Facebook | #1877F2 | Blue |
| Instagram | #E4405F | Pink/Red gradient |
| GitHub | #181717 | Black |
| LinkedIn | #0A66C2 | Blue |
| Twitter/X | #000000 | Black |
| YouTube | #FF0000 | Red |
| TikTok | #000000 | Black |
| Behance | #1769FF | Blue |
| Dribbble | #EA4C89 | Pink |
| Threads | #000000 | Black |
| Bluesky | #0085FF | Blue |
| Medium | #000000 | Black |
| GitLab | #FC6D26 | Orange |
| Personal Website | #6366F1 | Indigo |

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
Background: rgba(228, 64, 95, 0.1)
Text: #E4405F
```

### GitHub Chip
```
[GitHub Icon] johndoe
Background: rgba(24, 23, 23, 0.1)
Text: #181717
```

### Personal Website Chip
```
[Globe Icon] johndoe.com
Background: rgba(99, 102, 241, 0.1)
Text: #6366F1
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
