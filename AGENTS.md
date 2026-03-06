# AI Agent Instructions

This document provides instructions for AI coding agents working on this project.

## Frontend, Design & UI/UX Tasks

**CRITICAL: When working on any frontend, design, or UI/UX related tasks, you MUST use the `ui-ux-pro-max` skill.**

### When to Use UI/UX Pro Max Skill

Use this skill when:
- Designing new UI components or pages
- Building or modifying frontend interfaces
- Implementing design systems
- Choosing color palettes and typography
- Creating landing pages, dashboards, or admin panels
- Reviewing code for UX issues
- Implementing accessibility requirements
- Working with React, Next.js, Vue, Svelte, or any frontend framework
- Styling with Tailwind CSS or shadcn/ui
- Creating responsive layouts
- Implementing animations and interactions
- Building forms, modals, cards, tables, charts, etc.

### Skill Location

```
.agents/skills/ui-ux-pro-max/
```

### How to Use

The skill provides a comprehensive design intelligence system with:
- 50+ UI styles
- 97 color palettes
- 57 font pairings
- 99 UX guidelines
- 25 chart types
- 9 technology stacks

**Always start with the design system generator:**

```bash
python3 .agents/skills/ui-ux-pro-max/scripts/search.py "<product_type> <industry> <keywords>" --design-system -p "Project Name"
```

**Example for this project (Grafikarsa - Portfolio Platform):**

```bash
# For admin dashboard
python3 .agents/skills/ui-ux-pro-max/scripts/search.py "admin dashboard education portfolio management" --design-system -p "Grafikarsa Admin"

# For student portfolio pages
python3 .agents/skills/ui-ux-pro-max/scripts/search.py "portfolio showcase creative student education" --design-system -p "Grafikarsa Portfolio"

# For landing page
python3 .agents/skills/ui-ux-pro-max/scripts/search.py "education platform portfolio showcase school" --design-system -p "Grafikarsa Landing"
```

### Workflow

1. **Analyze Requirements** - Extract product type, style keywords, industry, stack
2. **Generate Design System** - Run `--design-system` command (REQUIRED)
3. **Supplement with Detailed Searches** - Use domain-specific searches as needed
4. **Get Stack Guidelines** - Use `--stack` flag for implementation best practices
5. **Implement** - Build the UI following the design system
6. **Review** - Check against the pre-delivery checklist

### Pre-Delivery Checklist

Before delivering any frontend code, verify:

#### Visual Quality
- [ ] No emojis used as icons (use SVG from Lucide React instead)
- [ ] All icons from consistent icon set (Lucide React)
- [ ] Hover states don't cause layout shift
- [ ] Use theme colors directly (bg-primary, text-foreground)

#### Interaction
- [ ] All clickable elements have `cursor-pointer`
- [ ] Hover states provide clear visual feedback
- [ ] Transitions are smooth (150-300ms)
- [ ] Focus states visible for keyboard navigation

#### Light/Dark Mode
- [ ] Light mode text has sufficient contrast (4.5:1 minimum)
- [ ] Glass/transparent elements visible in light mode
- [ ] Borders visible in both modes
- [ ] Test both modes before delivery

#### Layout
- [ ] Responsive at 375px, 768px, 1024px, 1440px
- [ ] No horizontal scroll on mobile
- [ ] No content hidden behind fixed navbars
- [ ] Consistent spacing and padding

#### Accessibility
- [ ] All images have alt text
- [ ] Form inputs have labels
- [ ] Color is not the only indicator
- [ ] `prefers-reduced-motion` respected
- [ ] Minimum 44x44px touch targets
- [ ] Keyboard navigation works properly

### Common Anti-Patterns to Avoid

❌ **Don't:**
- Use emojis as UI icons (🎨 🚀 ⚙️)
- Use `bg-white/10` in light mode (too transparent)
- Use gray-400 or lighter for body text in light mode
- Mix different icon sizes randomly
- Create hover states that shift layout
- Stick navbar to `top-0 left-0 right-0` without spacing
- Use instant state changes or transitions >500ms

✅ **Do:**
- Use SVG icons from Lucide React
- Use `bg-white/80` or higher in light mode
- Use `text-slate-900` for body text in light mode
- Use consistent icon sizing (w-5 h-5 or w-6 h-6)
- Use color/opacity transitions on hover
- Add spacing for floating navbars (`top-4 left-4 right-4`)
- Use smooth transitions (150-300ms)

## Backend & Go Tasks

When working on backend code (Go), refer to:
```
.agents/skills/golang-pro/
```

## Project Context

**Project:** Grafikarsa - Platform Katalog Portofolio & Social Network SMKN 4 Malang

**Tech Stack:**
- Frontend: Next.js 15, React 19, TypeScript, Tailwind CSS, shadcn/ui
- Backend: Go (Fiber framework), PostgreSQL, MinIO
- Deployment: Docker, VPS

**Design System:**
- Primary: Blue (#3B82F6)
- UI Library: shadcn/ui
- Icons: Lucide React
- Fonts: Inter (system font)

**Key Features:**
- Student portfolio showcase
- Admin dashboard for moderation
- Social features (follow, like, comment)
- Portfolio assessment system
- Series templates for structured portfolios
- Feed algorithm (smart, recent, following)

## Important Notes

1. **Always use the skill** - Don't skip the design system generation step
2. **Read the full SKILL.md** - Located at `.agents/skills/ui-ux-pro-max/SKILL.md`
3. **Follow the checklist** - Verify all items before delivering code
4. **Be consistent** - Use the same design patterns across all pages
5. **Test responsiveness** - Check on mobile, tablet, and desktop
6. **Consider accessibility** - This is a school project, accessibility matters

## Questions?

If you're unsure about design decisions:
1. Run the design system generator with relevant keywords
2. Search specific domains (style, color, typography, ux)
3. Check the UX guidelines for best practices
4. Review existing components in `apps/web/components/`
5. Refer to the PRD at `docs/prd.md`
