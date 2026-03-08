# AI Agent Instructions

This document provides comprehensive instructions for AI coding agents working on the Grafikarsa project.

---

## 📋 Project Context

**Project Name:** Grafikarsa  
**Description:** Platform Katalog Portofolio & Social Network untuk Warga SMKN 4 Malang

### Purpose
Grafikarsa adalah platform showcase portfolio dan social networking yang dirancang khusus untuk siswa dan alumni SMKN 4 Malang. Platform ini memungkinkan siswa untuk:
- Menampilkan karya kreatif mereka dalam bentuk portfolio
- Membangun jaringan dengan sesama siswa dan alumni
- Mendapatkan feedback dan apresiasi dari komunitas
- Mengeksplorasi karya inspiratif dari teman-teman mereka

### Tech Stack

**Frontend:**
- Next.js 15 (App Router)
- React 19
- TypeScript
- Tailwind CSS v4
- shadcn/ui components
- Framer Motion (animations)
- React Query (data fetching)
- Zustand (state management)

**Backend:**
- Go (Fiber framework)
- PostgreSQL (database)
- MinIO (object storage)

**Deployment:**
- Docker
- VPS

### Design System

**Colors:**
- Primary: Blue (#3B82F6)
- Using CSS variables for theming (light/dark mode)
- Semantic color tokens: `bg-primary`, `text-foreground`, `bg-muted`, etc.

**Typography:**
- Font: Inter (system font)
- Monospace: JetBrains Mono

**Icons:**
- Lucide React (NEVER use emojis as UI icons)

**UI Library:**
- shadcn/ui components
- Consistent border radius: `--radius: 0.625rem`

**Image Standards:**
- Avatar: 1:1 ratio (800x800px recommended)
- Banner: 3:1 ratio (1500x500px recommended)
- Portfolio Thumbnail: 4:3 ratio (1200x900px recommended)
- See `docs/ui/image-standards.md` for details

### Key Features

1. **Portfolio Management**
   - Create, edit, delete portfolios
   - Modular content blocks (text, image, video, links, tables)
   - Draft, review, and publish workflow
   - Series templates for structured portfolios

2. **Social Features**
   - Follow/unfollow users
   - Like and comment on portfolios
   - Activity feed (smart, recent, following)
   - User profiles with stats

3. **Admin Dashboard**
   - User management
   - Portfolio moderation (approve/reject)
   - Tag management
   - Series template management
   - Special roles assignment

4. **Discovery**
   - Browse portfolios by tags, class, major
   - Search users and portfolios
   - Explore feed with smart algorithm

### User Roles

- **Guest:** Can browse public portfolios and user profiles
- **Student:** Can create portfolios, follow users, interact with content
- **Admin:** Full access to moderation and management features

---

## 🛠️ Agent Skills Reference

This project uses specialized agent skills for different types of tasks. Each skill provides focused expertise and best practices for its domain.

### Skills Overview Table

| Skill | Location | Use When | Key Features |
|-------|----------|----------|--------------|
| **frontend-design** | `.kiro/skills/frontend-design/` | Building UI components, pages, or interfaces | Grafikarsa design system, shadcn/ui patterns, accessibility guidelines, responsive design |
| **golang-pro** | `.kiro/skills/golang-pro/` | Writing or modifying Go backend code | Go best practices, Fiber framework patterns, database operations, API design |
| **systematic-debugging** | `.kiro/skills/systematic-debugging/` | Encountering bugs, test failures, or unexpected behavior | Root cause analysis, hypothesis testing, defense-in-depth validation |
| **test-driven-development** | `.kiro/skills/test-driven-development/` | Writing tests or implementing TDD workflow | Test patterns, mocking strategies, test organization |
| **context7** | `.agents/skills/context7/` | Working with external libraries or need up-to-date documentation | Fetch current library docs, verify API signatures, check latest versions, library best practices |

---

## 📚 Detailed Skills Guide

### 1. Frontend Design Skill

**Location:** `.kiro/skills/frontend-design/SKILL.md`

**Use this skill when:**
- Building new UI components or pages
- Modifying existing frontend interfaces
- Implementing responsive layouts
- Adding animations or interactions
- Creating forms, modals, cards, or any UI element
- Styling with Tailwind CSS or shadcn/ui
- Working with React, Next.js, or TypeScript

**Key Guidelines:**
- Follow Grafikarsa design system (colors, typography, spacing)
- Use shadcn/ui components consistently
- Always use Lucide React icons (NEVER emojis)
- Ensure WCAG 2.1 AA accessibility compliance
- Test in both light and dark modes
- Implement mobile-first responsive design
- Use smooth transitions (150-300ms)
- Provide helpful loading and empty states

**Pre-Delivery Checklist:**
- [ ] No emojis used as icons
- [ ] Proper color contrast (4.5:1 minimum)
- [ ] Responsive at 375px, 768px, 1024px, 1440px
- [ ] Keyboard navigation works
- [ ] Light and dark modes tested
- [ ] All images have alt text
- [ ] Touch targets are 44x44px minimum

**Common Patterns:**
```tsx
// Page layout
<div className="container mx-auto max-w-5xl px-6 py-8 md:px-12 lg:px-16">
  <h1 className="mb-6 text-3xl font-bold">Page Title</h1>
  {/* Content */}
</div>

// Card grid
<div className="grid grid-cols-[repeat(auto-fill,minmax(320px,320px))] gap-6">
  {items.map(item => <Card key={item.id}>...</Card>)}
</div>

// Image with aspect ratio
<div className="relative aspect-[4/3] overflow-hidden rounded-lg">
  <Image src={url} alt="Description" fill className="object-cover object-center" />
</div>
```

---

### 2. Golang Pro Skill

**Location:** `.kiro/skills/golang-pro/`

**Use this skill when:**
- Writing or modifying Go backend code
- Implementing API endpoints
- Working with database operations
- Adding authentication/authorization logic
- Optimizing backend performance
- Handling file uploads (MinIO)
- Writing Go tests

**Key Guidelines:**
- Follow Go best practices and idioms
- Use Fiber framework patterns
- Implement proper error handling
- Write clean, testable code
- Use dependency injection
- Follow repository pattern
- Implement proper logging

**Project Structure:**
```
apps/backend/
├── cmd/
│   └── api/          # Main application entry
├── internal/
│   ├── auth/         # JWT authentication
│   ├── config/       # Configuration
│   ├── database/     # Database connection
│   ├── domain/       # Domain models
│   ├── dto/          # Data transfer objects
│   ├── handler/      # HTTP handlers
│   ├── middleware/   # Middleware
│   ├── repository/   # Database repositories
│   ├── service/      # Business logic
│   └── storage/      # MinIO storage
```

**Common Patterns:**
```go
// Handler pattern
func (h *Handler) GetPortfolio(c *fiber.Ctx) error {
    id := c.Params("id")
    portfolio, err := h.portfolioRepo.GetByID(c.Context(), id)
    if err != nil {
        return c.Status(404).JSON(dto.ErrorResponse("Portfolio not found"))
    }
    return c.JSON(dto.SuccessResponse(portfolio))
}

// Repository pattern
type PortfolioRepository interface {
    GetByID(ctx context.Context, id string) (*domain.Portfolio, error)
    Create(ctx context.Context, portfolio *domain.Portfolio) error
    Update(ctx context.Context, portfolio *domain.Portfolio) error
}
```

---

### 3. Systematic Debugging Skill

**Location:** `.kiro/skills/systematic-debugging/SKILL.md`

**Use this skill when:**
- Encountering ANY bug or unexpected behavior
- Tests are failing
- Performance issues arise
- Build failures occur
- Integration problems happen
- CI/CD pipeline fails
- **ESPECIALLY when:** Under time pressure, "quick fix" seems obvious, or previous fixes didn't work

**The Iron Law:**
```
NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST
```

**The Four Phases (MUST complete in order):**

1. **Root Cause Investigation**
   - Read error messages carefully
   - Reproduce consistently
   - Check recent changes
   - Gather evidence
   - Trace data flow

2. **Pattern Analysis**
   - Find working examples
   - Compare against references
   - Identify differences
   - Understand dependencies

3. **Hypothesis and Testing**
   - Form single hypothesis
   - Test minimally (one variable at a time)
   - Verify before continuing
   - If 3+ fixes failed → Question the architecture

4. **Implementation**
   - Create failing test case first
   - Implement single fix
   - Verify fix works
   - No bundled changes

**Red Flags - STOP and Follow Process:**
- "Quick fix for now, investigate later"
- "Just try changing X and see if it works"
- "Add multiple changes, run tests"
- "Skip the test, I'll manually verify"
- "It's probably X, let me fix that"
- "I don't fully understand but this might work"
- "One more fix attempt" (when already tried 2+)

**Supporting Techniques:**
- `root-cause-tracing.md` - Trace bugs backward through call stack
- `defense-in-depth.md` - Add validation at multiple layers
- `condition-based-waiting.md` - Replace arbitrary timeouts with condition polling

**Real-World Impact:**
- Systematic approach: 15-30 minutes to fix
- Random fixes approach: 2-3 hours of thrashing
- First-time fix rate: 95% vs 40%
- New bugs introduced: Near zero vs common

---

### 4. Test-Driven Development Skill

**Location:** `.kiro/skills/test-driven-development/SKILL.md`

**Use this skill when:**
- Writing new features with tests
- Implementing TDD workflow
- Refactoring existing code
- Adding test coverage
- Writing unit, integration, or e2e tests

**TDD Workflow:**
1. Write a failing test
2. Write minimal code to pass
3. Refactor while keeping tests green
4. Repeat

**Testing Patterns:**
```tsx
// Frontend (React Testing Library)
import { render, screen, fireEvent } from '@testing-library/react';

test('button click increments counter', () => {
  render(<Counter />);
  const button = screen.getByRole('button');
  fireEvent.click(button);
  expect(screen.getByText('Count: 1')).toBeInTheDocument();
});
```

```go
// Backend (Go testing)
func TestGetPortfolio(t *testing.T) {
    repo := &MockPortfolioRepo{}
    handler := NewHandler(repo)
    
    app := fiber.New()
    app.Get("/portfolios/:id", handler.GetPortfolio)
    
    req := httptest.NewRequest("GET", "/portfolios/123", nil)
    resp, _ := app.Test(req)
    
    assert.Equal(t, 200, resp.StatusCode)
}
```

---

### 5. Context7 Documentation Fetcher

**Location:** `.agents/skills/context7/SKILL.md`

**Use this skill when:**
- Working with ANY external library (React, Next.js, Tailwind, etc.)
- Installing new dependencies or frameworks
- User asks about library APIs, patterns, or best practices
- Implementing features that rely on third-party packages
- Debugging library-specific issues
- Need current documentation beyond training data cutoff
- **ESPECIALLY when:** Installing dependencies - ALWAYS check docs for latest versions

**CRITICAL:** Use this PROACTIVELY. Don't guess library APIs or rely on outdated knowledge.

**Workflow:**

1. **Search for library:**
```bash
py ~/.agents/skills/context7/scripts/context7.py search "<library-name>"
```

Example:
```bash
py ~/.agents/skills/context7/scripts/context7.py search "next.js"
```

2. **Fetch documentation:**
```bash
py ~/.agents/skills/context7/scripts/context7.py context "<library-id>" "<query>"
```

Example:
```bash
py ~/.agents/skills/context7/scripts/context7.py context "/vercel/next.js" "app router middleware"
```

**Quick Reference:**

| Task | Command |
|------|---------|
| Find React docs | `search "react"` |
| Get React hooks info | `context "/facebook/react" "useEffect cleanup"` |
| Find Next.js docs | `search "next.js"` |
| Get Next.js routing | `context "/vercel/next.js" "app router dynamic routes"` |
| Find Tailwind docs | `search "tailwind css"` |
| Get Tailwind config | `context "/tailwindlabs/tailwindcss" "configuration"` |
| Find Framer Motion | `search "framer motion"` |
| Get animation patterns | `context "/framer/motion" "animation variants"` |

**Common Use Cases for Grafikarsa:**
```bash
# Next.js 15 features
py ~/.agents/skills/context7/scripts/context7.py context "/vercel/next.js" "server actions form handling"

# React 19 features
py ~/.agents/skills/context7/scripts/context7.py context "/facebook/react" "use hook transitions"

# Tailwind CSS v4
py ~/.agents/skills/context7/scripts/context7.py context "/tailwindlabs/tailwindcss" "v4 migration"

# Framer Motion animations
py ~/.agents/skills/context7/scripts/context7.py context "/framer/motion" "layout animations"

# React Query patterns
py ~/.agents/skills/context7/scripts/context7.py context "/tanstack/query" "mutations optimistic updates"
```

**Options:**
- `--type txt|md` - Output format (default: txt)
- `--tokens N` - Limit response tokens

**Note:** API key is stored in `.agents/skills/context7/.env` (hidden file)

---

## 🎯 Decision Tree: Which Skill to Use?

```
Is the task about...

├─ Need library documentation?
│  └─ Use: context7 (FIRST!)
│     Examples: Installing packages, checking API signatures, latest versions
│
├─ Frontend UI/UX?
│  └─ Use: frontend-design
│     Examples: Building components, styling, layouts, animations
│
├─ Backend Go code?
│  └─ Use: golang-pro
│     Examples: API endpoints, database queries, business logic
│
├─ Bug or unexpected behavior?
│  └─ Use: systematic-debugging
│     Examples: Test failures, errors, performance issues
│
└─ Writing tests?
   └─ Use: test-driven-development
      Examples: Unit tests, integration tests, TDD workflow
```

**Pro Tip:** When working with external libraries, ALWAYS start with `context7` to get current documentation before implementing.

---

## 📖 Additional Resources

### Documentation
- **PRD:** `docs/prd.md` - Product requirements and features
- **Image Standards:** `docs/ui/image-standards.md` - Image specifications
- **API Documentation:** Check backend code for endpoint details

### Code Organization
- **Frontend:** `apps/web/` - Next.js application
- **Backend:** `apps/backend/` - Go API server
- **Components:** `apps/web/components/` - Reusable UI components
- **API Client:** `apps/web/lib/api/` - Frontend API integration
- **Types:** `apps/web/lib/types/` - TypeScript type definitions

### Common Commands
```bash
# Frontend development
cd apps/web
npm run dev

# Backend development
cd apps/backend
go run cmd/api/main.go

# Run tests
npm test              # Frontend
go test ./...         # Backend

# Build for production
npm run build         # Frontend
go build cmd/api/main.go  # Backend
```

---

## ⚠️ Critical Rules

1. **ALWAYS use context7 when working with external libraries** - Don't rely on outdated knowledge
2. **ALWAYS read the relevant skill documentation before starting work**
3. **Follow the design system** - Don't deviate from established patterns
4. **Test your changes** - Both manually and with automated tests
5. **Consider accessibility** - This is a school project, accessibility matters
6. **Use systematic debugging** - Don't guess when encountering bugs
7. **Write clean, maintainable code** - Others will work on this
8. **Document complex logic** - Help future developers understand
9. **Ask for clarification** - If requirements are unclear, ask the user

---

## 🤝 Contributing Guidelines

When working on this project:
- Follow the established code style and patterns
- Write meaningful commit messages
- Test thoroughly before submitting
- Update documentation when adding features
- Consider backward compatibility
- Think about edge cases and error handling
- Optimize for performance and user experience

---

## 📞 Questions?

If you're unsure about:
- **Library APIs or versions:** Use `context7` to fetch current documentation
- **Design decisions:** Check `frontend-design` skill and existing components
- **Backend patterns:** Check `golang-pro` skill and existing handlers
- **Debugging approach:** Follow `systematic-debugging` skill phases
- **Testing strategy:** Refer to `test-driven-development` skill

Remember: The skills are there to help you deliver high-quality, consistent code. Use them proactively!
