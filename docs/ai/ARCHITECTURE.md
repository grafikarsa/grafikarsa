# AI Ideas Generator - Architecture

## 📐 Component Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    AI Ideas Page                        │
│                 (ai-ideas/page.tsx)                     │
└─────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ Empty State  │   │    Wizard    │   │   Results    │
│  Component   │   │     Flow     │   │   Display    │
└──────────────┘   └──────────────┘   └──────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│   Wizard     │   │   Interest   │   │   Loading    │
│  Progress    │   │  Combobox    │   │   Progress   │
└──────────────┘   └──────────────┘   └──────────────┘
                                               │
                                               ▼
                                      ┌──────────────┐
                                      │     Idea     │
                                      │   Carousel   │
                                      └──────────────┘
```

## 🔄 State Flow

```
┌─────────────────────────────────────────────────────────┐
│                    Component State                      │
├─────────────────────────────────────────────────────────┤
│ • currentStep: number (1-3)                            │
│ • formData: GenerateProjectIdeasRequest                │
│ • savedIdeas: ProjectIdea[]                            │
│ • hasGenerated: boolean                                │
│ • showEmptyState: boolean                              │
│ • showClearDialog: boolean                             │
└─────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ localStorage │   │  API Calls   │   │  UI Updates  │
├──────────────┤   ├──────────────┤   ├──────────────┤
│ • form_draft │   │ • generate   │   │ • step nav   │
│ • ideas      │   │ • jurusan    │   │ • validation │
└──────────────┘   └──────────────┘   └──────────────┘
```

## 📊 Data Flow

```
User Input
    │
    ▼
┌─────────────────┐
│  Form Validation│
└─────────────────┘
    │
    ▼
┌─────────────────┐
│  localStorage   │ ◄─── Auto-save
│   (Draft)       │
└─────────────────┘
    │
    ▼
┌─────────────────┐
│  API Request    │
│  (Generate)     │
└─────────────────┘
    │
    ▼
┌─────────────────┐
│  Loading State  │ ◄─── Progress stages
└─────────────────┘
    │
    ▼
┌─────────────────┐
│  API Response   │
└─────────────────┘
    │
    ▼
┌─────────────────┐
│  localStorage   │ ◄─── Persist ideas
│   (Ideas)       │
└─────────────────┘
    │
    ▼
┌─────────────────┐
│  Carousel View  │
└─────────────────┘
    │
    ▼
User Actions
(Like/Skip/Save)
```

## 🎯 User Journey

```
┌──────────────────────────────────────────────────────────┐
│                    First Visit                           │
└──────────────────────────────────────────────────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │ Empty State  │
                    │ (Actionable) │
                    └──────────────┘
                            │
                    Click "Mulai"
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                    Step 1: Profil                        │
├──────────────────────────────────────────────────────────┤
│ • Select Jurusan (dropdown)                             │
│ • Add Interests (autocomplete + quick add)              │
│ • Validation: min 1 interest                            │
└──────────────────────────────────────────────────────────┘
                            │
                    Click "Lanjut"
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                    Step 2: Proyek                        │
├──────────────────────────────────────────────────────────┤
│ • Select Project Type (dropdown)                        │
│ • Select Difficulty (visual cards)                      │
│ • Validation: type selected                             │
└──────────────────────────────────────────────────────────┘
                            │
                    Click "Lanjut"
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                    Step 3: Review                        │
├──────────────────────────────────────────────────────────┤
│ • Show summary of all inputs                            │
│ • Confirm before generate                               │
│ • Can go back to edit                                   │
└──────────────────────────────────────────────────────────┘
                            │
                Click "Generate"
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                  Loading Progress                        │
├──────────────────────────────────────────────────────────┤
│ Stage 1: Menganalisis minat... (3s)                    │
│ Stage 2: Mencari teknologi... (4s)                     │
│ Stage 3: Membuat ide... (5s)                           │
│ Stage 4: Menyelesaikan... (3s)                         │
└──────────────────────────────────────────────────────────┘
                            │
                    Success (15s)
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│                    Results Carousel                      │
├──────────────────────────────────────────────────────────┤
│ • View 1 idea at a time                                 │
│ • Navigate with arrows/dots                             │
│ • Expand details                                        │
│ • Actions: Like/Skip/Save/Delete                        │
└──────────────────────────────────────────────────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │ Generate New │
                    │      or      │
                    │  Start Over  │
                    └──────────────┘
```

## 🔌 API Integration

```
┌─────────────────────────────────────────────────────────┐
│                    Frontend (Next.js)                   │
└─────────────────────────────────────────────────────────┘
                            │
                            │ POST /api/v1/ai/generate-project-ideas
                            │ Authorization: Bearer <token>
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                    Backend (Go/Fiber)                   │
│                  ai_handler.go                          │
└─────────────────────────────────────────────────────────┘
                            │
                            │ API Key: GOOGLE_GEMINI_API_KEY
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                  Google Gemini AI                       │
│              (gemini-3.1-flash-lite)                    │
└─────────────────────────────────────────────────────────┘
                            │
                            │ JSON Response
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                    Parse & Return                       │
│              GenerateProjectIdeasResponse               │
└─────────────────────────────────────────────────────────┘
```

## 🗂️ File Structure

```
apps/web/
├── app/(main)/ai-ideas/
│   └── page.tsx                    # Main page component
│
├── components/ai/
│   ├── wizard-progress.tsx         # Step indicator
│   ├── interest-combobox.tsx       # Smart input
│   ├── idea-carousel.tsx           # Results display
│   ├── loading-progress.tsx        # Loading animation
│   ├── empty-state.tsx             # Initial state
│   └── mobile-edit-drawer.tsx      # Mobile drawer
│
├── hooks/
│   ├── use-local-storage.ts        # localStorage hook
│   └── use-media-query.ts          # Responsive hooks
│
└── lib/api/ai/
    └── index.ts                    # API client

apps/backend/
└── internal/
    ├── handler/
    │   └── ai_handler.go           # API handler
    └── dto/
        └── ai.go                   # Data types

docs/ai/
├── ai-ideas-wizard-guide.md        # Full guide
├── phase-1-2-implementation-summary.md
├── QUICK-START.md                  # Quick reference
└── ARCHITECTURE.md                 # This file
```

## 🔐 Security

```
┌─────────────────────────────────────────────────────────┐
│                    Security Layers                      │
└─────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│     Auth     │   │     API      │   │    Input     │
│  Middleware  │   │     Keys     │   │ Validation   │
├──────────────┤   ├──────────────┤   ├──────────────┤
│ • JWT token  │   │ • Gemini key │   │ • Sanitize   │
│ • Required   │   │ • Env var    │   │ • Validate   │
└──────────────┘   └──────────────┘   └──────────────┘
```

## 📦 Dependencies

### Frontend
- `@tanstack/react-query` - Data fetching
- `@radix-ui/*` - UI primitives
- `lucide-react` - Icons
- `sonner` - Toast notifications
- `vaul` - Drawer component
- `framer-motion` - Animations (optional)

### Backend
- `github.com/gofiber/fiber/v2` - Web framework
- `github.com/google/generative-ai-go` - Gemini AI
- `google.golang.org/api` - Google API client

## 🎨 Design System

### Component Hierarchy
```
Page (Container)
├── Header (Title + Description)
├── Empty State (Initial)
│   ├── Icon (Animated)
│   ├── CTA Button
│   └── Features Grid
├── Wizard (Multi-step)
│   ├── Progress Indicator
│   ├── Step 1 (Form)
│   │   ├── Jurusan Select
│   │   └── Interest Combobox
│   ├── Step 2 (Form)
│   │   ├── Type Select
│   │   └── Difficulty Cards
│   └── Step 3 (Review)
│       └── Summary Card
├── Loading (Progress)
│   ├── Animated Icon
│   ├── Progress Bar
│   └── Stage Messages
└── Results (Carousel)
    ├── Navigation
    ├── Idea Card
    │   ├── Header
    │   ├── Content
    │   └── Details (Collapsible)
    └── Actions
```

## 🔄 State Management

### Local State (useState)
- `currentStep` - Wizard navigation
- `formData` - Form inputs
- `savedIdeas` - Generated ideas
- `hasGenerated` - Generation status
- `showEmptyState` - Initial state
- `showClearDialog` - Confirmation

### Persistent State (localStorage)
- `ai_ideas_form_draft` - Form backup
- `ai_project_ideas` - Ideas backup

### Server State (React Query)
- `jurusan` - Jurusan list (cached)
- `generateIdeas` - Generation mutation

## 🚀 Performance

### Optimizations
1. **Lazy Loading** - Components loaded on demand
2. **Memoization** - Prevent unnecessary re-renders
3. **Debouncing** - localStorage saves debounced
4. **Caching** - React Query caches jurusan
5. **Code Splitting** - Route-based splitting

### Bundle Size
- Main page: ~15KB (gzipped)
- Components: ~8KB (gzipped)
- Total: ~23KB (gzipped)

## 📱 Responsive Strategy

### Mobile First
```
Base Styles (Mobile)
    │
    ▼
Tablet Overrides (md:)
    │
    ▼
Desktop Overrides (lg:)
```

### Breakpoints
- `sm`: 640px
- `md`: 768px
- `lg`: 1024px
- `xl`: 1280px
- `2xl`: 1536px

## ♿ Accessibility

### WCAG 2.1 AA Compliance
- ✅ Color contrast 4.5:1
- ✅ Keyboard navigation
- ✅ Focus indicators
- ✅ ARIA labels
- ✅ Screen reader support
- ✅ Touch targets 44x44px
- ✅ Form labels
- ✅ Error messages

## 🧪 Testing Strategy

### Unit Tests
- Component rendering
- Hook behavior
- Utility functions

### Integration Tests
- Wizard flow
- API integration
- localStorage persistence

### E2E Tests
- Complete user journey
- Error scenarios
- Mobile experience

## 📈 Monitoring

### Metrics to Track
1. Form completion rate
2. Generation success rate
3. Average time per step
4. Error rate
5. Mobile vs desktop usage
6. Most popular interests
7. Most popular project types

### Analytics Events
- `wizard_step_completed`
- `idea_generated`
- `idea_liked`
- `idea_skipped`
- `idea_saved`
- `error_occurred`

---

**Last Updated:** 7 Maret 2026  
**Version:** 2.0
