# User Components

## Social Media Chips

Social media badges di halaman profil menggunakan komponen `SocialChip` yang menampilkan:
- Icon brand SVG dari masing-masing platform (kecuali personal website yang pakai Lucide Globe)
- Username/handle user di platform tersebut
- Warna brand platform (sebagai warna text dan fill SVG)
- Background dengan opacity dari warna brand (10% opacity)
- Hover effect dengan scale animation

### Komponen

**SocialChip** (`social-chip.tsx`)
- Menampilkan chip dengan warna brand platform
- Otomatis extract username/handle dari URL
- Menggunakan `PlatformIcon` untuk render icon
- Responsive dan accessible

**PlatformIcon** (`platform-icon.tsx`)
- Komponen reusable untuk render icon platform
- Mendukung SVG string dan Lucide icon
- Otomatis apply warna brand
- Configurable size: sm (3.5), md (4), lg (5)

### Konfigurasi Platform

Konfigurasi warna brand dan SVG icon untuk setiap platform ada di:
`apps/web/lib/constants/social-platforms.ts`

**Format Icon:**
- Semua platform menggunakan SVG string dari Simple Icons (https://simpleicons.org/)
- Kecuali `personal_website` yang menggunakan `Globe` icon dari Lucide React
- SVG menggunakan `fill-current` untuk mengikuti warna brand

Platform yang didukung:
- Facebook (#0866FF) - SVG brand icon
- Instagram (#FF0069) - SVG brand icon
- GitHub (#181717) - SVG brand icon
- LinkedIn (#0A66C2) - SVG brand icon
- Twitter/X (#000000) - SVG brand icon
- YouTube (#FF0000) - SVG brand icon
- TikTok (#000000) - SVG brand icon
- Behance (#1769FF) - SVG brand icon
- Dribbble (#EA4C89) - SVG brand icon
- Threads (#000000) - SVG brand icon
- Bluesky (#1185FE) - SVG brand icon
- Medium (#000000) - SVG brand icon
- GitLab (#FC6D26) - SVG brand icon
- Personal Website (#6366F1) - Lucide Globe icon

### Usage

**SocialChip (untuk profil):**
```tsx
import { SocialChip } from '@/components/user/social-chip';

<SocialChip link={{ platform: 'instagram', url: 'https://instagram.com/username' }} />
```

**PlatformIcon (untuk form label, dll):**
```tsx
import { PlatformIcon } from '@/components/user/platform-icon';

<Label>
  <PlatformIcon platform="instagram" size="sm" />
  Instagram
</Label>
```

### Extract Username

Fungsi `extractSocialHandle` otomatis mengekstrak username dari URL:
- Instagram: @username
- Twitter: @username
- GitHub: username
- LinkedIn: username (skip 'in' prefix)
- Personal Website: domain name
- dll.

### Technical Details

**SVG Rendering:**
- SVG string di-render dengan `dangerouslySetInnerHTML`
- Class `fill-current` diterapkan untuk mengikuti warna parent
- Warna brand diterapkan via inline style `color: brandColor`
- SVG otomatis menggunakan `currentColor` untuk fill

**Lucide Icon (personal_website):**
- Render sebagai React component
- Menggunakan class `h-4 w-4` untuk sizing
- Warna mengikuti parent color

### Form Integration

Di halaman edit profil (`user-edit-form.tsx`), setiap input field social media menampilkan:
- Logo platform di sebelah kiri label (menggunakan `PlatformIcon`)
- Nama platform
- Input field untuk URL

Contoh:
```tsx
<Label className="flex items-center gap-2">
  <PlatformIcon platform="instagram" size="sm" />
  Instagram
</Label>
<Input placeholder="https://instagram.com/username" />
```
