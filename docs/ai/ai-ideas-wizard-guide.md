# AI Project Ideas Generator - Wizard Guide

## Overview

Halaman AI Project Ideas Generator telah diperbarui dengan wizard multi-step yang lebih user-friendly dan mengurangi cognitive overload.

## Key Improvements

### Phase 1: Multi-Step Wizard & Form Improvements

#### 1. Multi-Step Wizard (3 Steps)
- **Step 1: Profil** - Jurusan & Minat
- **Step 2: Proyek** - Tipe & Kesulitan
- **Step 3: Generate** - Review & Generate

**Benefits:**
- Mengurangi cognitive overload dengan fokus pada 1-2 input per step
- Progress indicator yang jelas
- Validasi per step sebelum lanjut
- User tidak overwhelmed dengan form panjang

#### 2. Improved Interest Input (Combobox)
- Autocomplete dengan dropdown suggestions
- Quick add badges di atas input (lebih prominent)
- Visual counter (X/10 selections)
- Keyboard navigation support (Enter to add, Escape to close)
- Click suggestion langsung tanpa perlu ketik

**Benefits:**
- Friction berkurang drastis
- User bisa add interests lebih cepat
- Better discoverability dengan suggestions di atas

#### 3. Form State Persistence
- Auto-save form data ke localStorage
- Restore form saat user kembali
- Preserve data setelah error

**Benefits:**
- User tidak perlu isi ulang jika terjadi error
- Better UX untuk session yang terputus
- Draft saving otomatis

#### 4. Better Error Handling
- Error toast dengan "Coba Lagi" action button
- Form data preserved setelah error
- Clear error messages dari backend

**Benefits:**
- User bisa retry tanpa isi ulang form
- Error recovery yang smooth
- Better user feedback

### Phase 2: Results Presentation & Mobile Optimization

#### 5. Carousel Results View
- Tampilkan 1 idea at a time
- Navigation dengan arrows dan dots indicator
- Collapsible details (toggle "Lihat Detail Lengkap")
- Swipe-friendly untuk mobile

**Benefits:**
- Tidak overwhelming dengan 5 ideas sekaligus
- User fokus pada 1 idea
- Better mobile experience
- Reduced visual clutter

#### 6. Mobile-Optimized Layout
- Responsive wizard progress (simplified di mobile)
- Touch-friendly buttons (min 44x44px)
- Proper spacing untuk mobile
- No horizontal scroll

**Benefits:**
- Better mobile UX
- Easier navigation di mobile
- Touch-optimized interactions

#### 7. Feedback Mechanism
- Like button (thumbs up)
- Skip button (thumbs down) - auto delete
- Save button (untuk future feature)
- Delete individual idea

**Benefits:**
- User bisa curate ideas
- Better engagement
- Personalization foundation

#### 8. Better Loading States
- Animated loading progress dengan stages
- Progress messages: "Menganalisis minat..." → "Mencari teknologi..." → "Membuat ide..."
- Progress bar dengan percentage
- Estimated time indicator (10-15 detik)

**Benefits:**
- User tahu apa yang terjadi
- Perceived performance lebih baik
- Engaging loading experience

## User Flow

### New Flow (Wizard-Based)

```
Landing
  ↓
Step 1: Profil (Jurusan + Interests)
  ↓ [Validasi: jurusan & min 1 interest]
Step 2: Proyek (Type + Difficulty)
  ↓ [Validasi: project type selected]
Step 3: Review & Generate
  ↓ [Show summary, confirm]
Loading (with progress stages)
  ↓
Results (Carousel View)
  ↓
Actions: Like / Skip / Save / Delete
  ↓
Generate Again or Start Over
```

## Components

### 1. WizardProgress
**Location:** `apps/web/components/ai/wizard-progress.tsx`

**Props:**
- `currentStep: number` - Current step (1-3)
- `totalSteps: number` - Total steps (3)
- `steps: Array<{label, description}>` - Step definitions

**Features:**
- Visual progress with circles and connecting lines
- Completed steps show checkmark
- Current step highlighted
- Responsive (simplified on mobile)

### 2. InterestCombobox
**Location:** `apps/web/components/ai/interest-combobox.tsx`

**Props:**
- `selectedInterests: string[]` - Currently selected interests
- `onInterestsChange: (interests: string[]) => void` - Callback
- `suggestions: string[]` - Available suggestions
- `maxSelections?: number` - Max selections (default: 10)

**Features:**
- Autocomplete dropdown
- Quick add badges (prominent)
- Visual counter
- Keyboard navigation
- Click outside to close

### 3. IdeaCarousel
**Location:** `apps/web/components/ai/idea-carousel.tsx`

**Props:**
- `ideas: ProjectIdea[]` - Array of ideas
- `onLike?: (index: number) => void` - Like callback
- `onSkip?: (index: number) => void` - Skip callback
- `onSave?: (index: number) => void` - Save callback
- `onDelete?: (index: number) => void` - Delete callback

**Features:**
- One idea at a time
- Navigation arrows + dots
- Collapsible details
- Action buttons (Like/Skip/Save)
- Delete button in header

### 4. LoadingProgress
**Location:** `apps/web/components/ai/loading-progress.tsx`

**Features:**
- Animated icon with pulse effect
- Progress bar with percentage
- Stage-based messages
- Stage indicators
- Estimated time

## Storage Keys

### localStorage Keys
- `ai_ideas_form_draft` - Form data draft (auto-saved)
- `ai_project_ideas` - Generated ideas (persisted)

## Accessibility

### Keyboard Navigation
- Tab through all interactive elements
- Enter to add interest in combobox
- Escape to close dropdown
- Arrow keys for carousel navigation

### Screen Reader Support
- All form inputs have labels
- ARIA labels for icon buttons
- Focus management in wizard steps
- Live region for generated ideas

### Touch Targets
- All buttons minimum 44x44px
- Adequate spacing between interactive elements
- Touch-friendly carousel navigation

## Responsive Breakpoints

- **Mobile (< 768px):**
  - Simplified wizard progress (vertical)
  - Full-width cards
  - Stacked action buttons
  - Larger touch targets

- **Tablet (768px - 1024px):**
  - Horizontal wizard progress
  - Optimized card layout
  - Side-by-side action buttons

- **Desktop (> 1024px):**
  - Full wizard progress with descriptions
  - Maximum width container (max-w-4xl)
  - Optimal spacing and typography

## Performance Considerations

### Optimizations
- Form data auto-save debounced (via useEffect)
- localStorage operations minimized
- Lazy loading for heavy components
- Efficient re-renders with proper state management

### Bundle Size
- All components use existing shadcn/ui primitives
- No additional heavy dependencies
- Lucide icons (tree-shakeable)

## Future Enhancements (Phase 3)

### Planned Features
1. Onboarding tour for first-time users
2. Export ideas to PDF
3. Share ideas via link
4. Save ideas to portfolio directly
5. AI refinement ("Generate more like this")
6. Idea rating and feedback to improve AI
7. Template-based idea generation
8. Collaborative idea sharing

## Testing Checklist

### Functional Testing
- [ ] Wizard navigation works (Next/Back)
- [ ] Form validation per step
- [ ] Interest combobox autocomplete
- [ ] Quick add suggestions work
- [ ] Form data persists on refresh
- [ ] Generate API call successful
- [ ] Loading progress shows stages
- [ ] Carousel navigation works
- [ ] Like/Skip/Delete actions work
- [ ] Clear all confirmation dialog
- [ ] Start over resets form

### Responsive Testing
- [ ] Mobile (375px) - No horizontal scroll
- [ ] Tablet (768px) - Proper layout
- [ ] Desktop (1024px+) - Optimal spacing
- [ ] Touch targets adequate on mobile
- [ ] Buttons accessible on all sizes

### Accessibility Testing
- [ ] Keyboard navigation works
- [ ] Focus visible on all elements
- [ ] Screen reader announces changes
- [ ] Color contrast sufficient
- [ ] No reliance on color alone

### Error Handling
- [ ] Network error shows retry option
- [ ] Form data preserved after error
- [ ] Clear error messages
- [ ] Graceful degradation

## Known Limitations

1. **Ideas not saved to database** - Currently localStorage only
2. **No idea refinement** - Can't ask AI to modify specific idea
3. **No sharing** - Can't share ideas with others
4. **No export** - Can't export to PDF/image
5. **No analytics** - Can't track which ideas are popular

## Troubleshooting

### Form data not persisting
- Check localStorage is enabled
- Check browser privacy settings
- Clear localStorage and try again

### Autocomplete not working
- Check suggestions array is populated
- Verify input is not disabled
- Check console for errors

### Carousel not navigating
- Verify ideas array has multiple items
- Check button disabled states
- Verify event handlers attached

### Loading stuck
- Check API endpoint is responding
- Verify Gemini API key is configured
- Check network tab for errors
- Verify timeout settings (30s)

## API Integration

### Endpoint
`POST /api/v1/ai/generate-project-ideas`

### Request
```json
{
  "jurusan": "Rekayasa Perangkat Lunak",
  "interests": ["Web Development", "UI/UX Design"],
  "project_type": "web_app",
  "difficulty": "intermediate"
}
```

### Response
```json
{
  "success": true,
  "data": {
    "ideas": [
      {
        "title": "Sistem Manajemen Perpustakaan",
        "description": "...",
        "technologies": ["React", "Node.js"],
        "difficulty": "intermediate",
        "estimated_time": "4-6 minggu",
        "learning_goals": ["..."]
      }
    ]
  }
}
```

### Error Handling
- Network errors: Show retry toast
- Validation errors: Show field-specific errors
- API errors: Show error message with retry option
- Timeout: Show timeout message with retry

## Maintenance

### Regular Updates
- Update interest suggestions based on trends
- Refine AI prompts for better ideas
- Monitor error rates and fix issues
- Gather user feedback for improvements

### Monitoring
- Track generation success rate
- Monitor API response times
- Track user completion rate per step
- Monitor localStorage usage

