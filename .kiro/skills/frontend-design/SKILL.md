---
name: frontend-design
description: Create production-grade frontend interfaces following Grafikarsa design system. Use this skill when building web components, pages, or applications for the Grafikarsa portfolio platform (examples include portfolio pages, dashboards, profile components, or any web UI). Generates polished, accessible code that follows established design patterns.
license: Complete terms in LICENSE.txt
---

This skill guides creation of production-grade frontend interfaces that follow Grafikarsa's established design system and best practices. Implement real working code with exceptional attention to consistency, accessibility, and user experience.

The user provides frontend requirements: a component, page, application, or interface to build. They may include context about the purpose, audience, or technical constraints.

## Project Context: Grafikarsa

**About:** Platform Katalog Portofolio & Social Network for SMKN 4 Malang students to showcase their creative work.

**Tech Stack:**
- Frontend: Next.js 15, React 19, TypeScript
- Styling: Tailwind CSS v4, shadcn/ui components
- Icons: Lucide React (NEVER use emojis)
- Fonts: Inter (system font)
- Animations: Framer Motion, CSS transitions
- State: React Query, Zustand

**Design Philosophy:**
- Clean, modern, and professional
- Accessibility-first approach
- Consistent spacing and typography
- Smooth micro-interactions
- Light/dark mode support
- Mobile-first responsive design

## Design System

### Color Palette
**Primary:** Blue (#3B82F6 / oklch values in CSS)
- Use `bg-primary`, `text-primary`, `border-primary` for consistency
- Primary is used for CTAs, links, and key interactive elements

**Semantic Colors:**
- Background: `bg-background` (white in light, dark in dark mode)
- Foreground: `text-foreground` (near-black in light, near-white in dark)
- Muted: `bg-muted`, `text-muted-foreground` for secondary content
- Card: `bg-card` with `border` for elevated surfaces
- Destructive: `bg-destructive` for errors and dangerous actions

**CRITICAL:** Always use CSS variables (`bg-primary`, `text-foreground`) instead of hardcoded colors. This ensures proper light/dark mode support.

### Typography
**Font Family:** Inter (system font fallback)
- Headings: `font-bold` or `font-semibold`
- Body: `font-normal` or `font-medium`
- Code: `font-mono` (JetBrains Mono)

**Font Sizes (Tailwind):**
- Hero: `text-4xl` to `text-6xl`
- Page Title: `text-2xl` to `text-3xl`
- Section Heading: `text-xl` to `text-2xl`
- Card Title: `text-base` to `text-lg`
- Body: `text-sm` to `text-base`
- Caption: `text-xs` to `text-sm`

**Line Height:** Use default Tailwind leading classes for optimal readability.

### Spacing & Layout
**Container Widths:**
- Standard content: `max-w-5xl` (1024px)
- Wide content: `max-w-6xl` (1152px)
- Full width: `max-w-7xl` (1280px)

**Padding/Margin Scale:**
- Tight: `p-2`, `p-3`, `p-4`
- Standard: `p-4`, `p-6`, `p-8`
- Generous: `p-8`, `p-12`, `p-16`

**Responsive Padding Pattern:**
```tsx
className="px-4 md:px-6 lg:px-8"  // Horizontal
className="py-4 md:py-6 lg:py-8"  // Vertical
```

**Grid Layouts:**
- Portfolio cards: `grid grid-cols-[repeat(auto-fill,minmax(320px,320px))] gap-6`
- Responsive columns: `grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6`

### Border Radius
**Standard:** `--radius: 0.625rem` (10px)
- Small: `rounded-md` (calc(var(--radius) - 2px))
- Default: `rounded-lg` (var(--radius))
- Large: `rounded-xl` (calc(var(--radius) + 4px))
- Full: `rounded-2xl` for banners and hero sections

### Icons
**CRITICAL:** ALWAYS use Lucide React icons. NEVER use emojis as UI icons.

```tsx
import { Plus, Edit, Trash2, Upload } from 'lucide-react';

// Standard sizes
<Icon className="h-4 w-4" />  // Small (buttons, inline)
<Icon className="h-5 w-5" />  // Medium (default)
<Icon className="h-6 w-6" />  // Large (prominent actions)
<Icon className="h-8 w-8" />  // Extra large (empty states)
```

### Components (shadcn/ui)
Use existing shadcn/ui components from `@/components/ui/`:
- Button, Card, Badge, Avatar
- Input, Textarea, Label, Select
- Dialog, Popover, DropdownMenu
- Skeleton, Separator, ScrollArea
- And more...

**Import Pattern:**
```tsx
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
```

### Image Standards
Follow `docs/ui/image-standards.md`:

**Avatar (Profile Photo):**
- Ratio: 1:1 (square)
- Recommended: 800x800px
- Display: Circular mask
- Focal point: Center (face-centered)

**Banner (Profile Cover):**
- Ratio: 3:1 (wide)
- Recommended: 1500x500px
- Display: `aspect-[3/1]` with `object-cover object-center`
- Safe zone: Center 900px width

**Portfolio Thumbnail:**
- Ratio: 4:3 (horizontal)
- Recommended: 1200x900px
- Display: `aspect-[4/3]` with `object-cover object-center`
- Focal point: Upper third (Rule of Thirds)

**Image Component Pattern:**
```tsx
import Image from 'next/image';

<div className="relative aspect-[4/3] overflow-hidden rounded-lg">
  <Image
    src={imageUrl}
    alt="Descriptive alt text"
    fill
    className="object-cover object-center"
  />
</div>
```

## Interaction & Animation

### Transitions
**Standard Duration:** 150-300ms
```tsx
className="transition-colors duration-200"
className="transition-all duration-300"
```

**Hover States:**
- Buttons: `hover:bg-primary/90`
- Cards: `hover:border-primary/50`
- Links: `hover:text-primary`
- Icons: `hover:text-foreground`

**CRITICAL:** Hover states should NOT cause layout shift. Use color/opacity changes, not size/position changes.

### Animations (Framer Motion)
Use for complex interactions:
```tsx
import { motion, AnimatePresence } from 'framer-motion';

<motion.div
  initial={{ opacity: 0, y: 20 }}
  animate={{ opacity: 1, y: 0 }}
  exit={{ opacity: 0, y: -20 }}
  transition={{ duration: 0.3, ease: [0.04, 0.62, 0.23, 0.98] }}
>
  {content}
</motion.div>
```

**Respect User Preferences:**
```tsx
// Check for reduced motion preference
const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
```

### Loading States
Use Skeleton components for loading:
```tsx
import { Skeleton } from '@/components/ui/skeleton';

<Skeleton className="h-48 w-full rounded-lg" />
```

### Empty States
Provide helpful, contextual empty states:
- Icon (Lucide React, not emoji)
- Heading explaining the state
- Description with guidance
- CTA button (if user can take action)

**Pattern:**
```tsx
<div className="rounded-lg border border-dashed bg-muted/30 p-12 text-center">
  <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
    <Icon className="h-8 w-8 text-primary" />
  </div>
  <h3 className="mb-2 text-lg font-semibold">Empty State Title</h3>
  <p className="mb-6 text-sm text-muted-foreground">
    Helpful description
  </p>
  <Button>Take Action</Button>
</div>
```

## Accessibility Requirements

### WCAG 2.1 Level AA Compliance
**Color Contrast:**
- Normal text: 4.5:1 minimum
- Large text (18pt+): 3:1 minimum
- UI components: 3:1 minimum

**Light Mode Specific:**
- NEVER use `text-gray-400` or lighter for body text
- Use `text-slate-900` or `text-foreground` for readable text
- Ensure borders are visible: `border` not `border-gray-100`

**Dark Mode Specific:**
- Test all components in dark mode
- Ensure glass/transparent elements are visible
- Use `border` with proper opacity

### Keyboard Navigation
- All interactive elements must be keyboard accessible
- Visible focus states: `focus:ring-2 focus:ring-primary`
- Logical tab order
- Skip links for main content

### Touch Targets
- Minimum 44x44px for all interactive elements
- Use `p-2` or larger for buttons
- Adequate spacing between clickable items

### Semantic HTML
```tsx
// Good
<button onClick={handleClick}>Click me</button>
<nav aria-label="Main navigation">...</nav>
<main>...</main>

// Bad
<div onClick={handleClick}>Click me</div>
<div className="nav">...</div>
```

### ARIA Labels
```tsx
<button aria-label="Close dialog">
  <X className="h-4 w-4" />
</button>

<img src={url} alt="Portfolio thumbnail showing web design" />
```

## Responsive Design

### Breakpoints (Tailwind)
- Mobile: Default (< 768px)
- Tablet: `md:` (768px+)
- Desktop: `lg:` (1024px+)
- Wide: `xl:` (1280px+)

### Mobile-First Approach
```tsx
// Start with mobile, add larger breakpoints
className="text-sm md:text-base lg:text-lg"
className="p-4 md:p-6 lg:p-8"
className="grid-cols-1 md:grid-cols-2 lg:grid-cols-3"
```

### Test at Key Widths
- 375px (iPhone SE)
- 768px (iPad portrait)
- 1024px (iPad landscape)
- 1440px (Desktop)

### Prevent Horizontal Scroll
- Use `overflow-hidden` on containers when needed
- Test all breakpoints
- Ensure images are responsive

## Common Patterns

### Page Layout
```tsx
export default function Page() {
  return (
    <div className="container mx-auto max-w-5xl px-6 py-8 md:px-12 lg:px-16">
      <h1 className="mb-6 text-3xl font-bold">Page Title</h1>
      {/* Content */}
    </div>
  );
}
```

### Card Grid
```tsx
<div className="grid grid-cols-[repeat(auto-fill,minmax(320px,320px))] gap-6">
  {items.map(item => (
    <Card key={item.id}>
      <CardContent className="p-3">
        {/* Card content */}
      </CardContent>
    </Card>
  ))}
</div>
```

### Form Layout
```tsx
<form onSubmit={handleSubmit} className="space-y-6">
  <div className="space-y-2">
    <Label htmlFor="name">Name</Label>
    <Input id="name" type="text" required />
  </div>
  <Button type="submit">Submit</Button>
</form>
```

### Modal/Dialog
```tsx
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';

<Dialog open={open} onOpenChange={setOpen}>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>Dialog Title</DialogTitle>
    </DialogHeader>
    {/* Content */}
  </DialogContent>
</Dialog>
```

## Anti-Patterns to Avoid

### ❌ DON'T:
- Use emojis as UI icons (🎨 🚀 ⚙️)
- Use `bg-white/10` in light mode (too transparent)
- Use `text-gray-400` or lighter for body text in light mode
- Mix different icon sizes randomly
- Create hover states that shift layout
- Use instant transitions or >500ms durations
- Hardcode colors instead of using CSS variables
- Forget to test dark mode
- Ignore accessibility requirements
- Use arbitrary values when Tailwind classes exist

### ✅ DO:
- Use Lucide React icons consistently
- Use semantic color classes (`bg-primary`, `text-foreground`)
- Test in both light and dark modes
- Provide proper alt text for images
- Use smooth transitions (150-300ms)
- Follow established spacing patterns
- Implement proper loading and empty states
- Ensure keyboard navigation works
- Test on mobile devices
- Use TypeScript for type safety

## Pre-Delivery Checklist

Before delivering any frontend code, verify:

### Visual Quality
- [ ] No emojis used as icons (Lucide React only)
- [ ] All icons from consistent set (Lucide React)
- [ ] Hover states don't cause layout shift
- [ ] Using theme colors (`bg-primary`, `text-foreground`)
- [ ] Consistent spacing and padding
- [ ] Proper border radius applied

### Interaction
- [ ] All clickable elements have `cursor-pointer`
- [ ] Hover states provide clear visual feedback
- [ ] Transitions are smooth (150-300ms)
- [ ] Focus states visible for keyboard navigation
- [ ] Loading states implemented
- [ ] Empty states are helpful and actionable

### Light/Dark Mode
- [ ] Light mode text has sufficient contrast (4.5:1 minimum)
- [ ] Glass/transparent elements visible in light mode
- [ ] Borders visible in both modes
- [ ] Tested both modes before delivery
- [ ] No hardcoded colors (use CSS variables)

### Layout
- [ ] Responsive at 375px, 768px, 1024px, 1440px
- [ ] No horizontal scroll on mobile
- [ ] No content hidden behind fixed elements
- [ ] Consistent spacing and padding
- [ ] Mobile-first approach used

### Accessibility
- [ ] All images have descriptive alt text
- [ ] Form inputs have associated labels
- [ ] Color is not the only indicator
- [ ] `prefers-reduced-motion` respected
- [ ] Minimum 44x44px touch targets
- [ ] Keyboard navigation works properly
- [ ] Semantic HTML used
- [ ] ARIA labels where needed

### Code Quality
- [ ] TypeScript types properly defined
- [ ] No console errors or warnings
- [ ] Proper error handling
- [ ] Loading states handled
- [ ] Optimistic updates where appropriate
- [ ] Following Next.js 15 best practices

## Integration with Existing Codebase

### API Calls
```tsx
import { portfoliosApi, usersApi } from '@/lib/api';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

const { data, isLoading } = useQuery({
  queryKey: ['portfolios'],
  queryFn: () => portfoliosApi.getPortfolios(),
});
```

### Authentication
```tsx
import { useAuthStore } from '@/lib/stores/auth-store';

const { user, isAuthenticated } = useAuthStore();
```

### Routing
```tsx
import { useRouter } from 'next/navigation';
import Link from 'next/link';

const router = useRouter();
router.push('/path');

<Link href="/path">Link text</Link>
```

### Toast Notifications
```tsx
import { toast } from 'sonner';

toast.success('Success message');
toast.error('Error message');
```

## Summary

When building frontend interfaces for Grafikarsa:
1. Follow the established design system (colors, typography, spacing)
2. Use shadcn/ui components consistently
3. Ensure accessibility (WCAG 2.1 AA)
4. Test in light and dark modes
5. Implement responsive design (mobile-first)
6. Use Lucide React icons (never emojis)
7. Add smooth transitions and micro-interactions
8. Provide helpful loading and empty states
9. Write clean, type-safe TypeScript code
10. Test at all breakpoints before delivery

The goal is to create interfaces that are beautiful, accessible, performant, and consistent with the rest of the platform.
