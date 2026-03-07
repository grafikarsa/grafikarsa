# AI Ideas Generator - Quick Start Guide

## 🚀 Quick Start

### For Developers

1. **Navigate to the page:**
   ```
   http://localhost:3000/ai-ideas
   ```

2. **Components are located in:**
   ```
   apps/web/components/ai/
   ├── wizard-progress.tsx      # Progress indicator
   ├── interest-combobox.tsx    # Smart interest selector
   ├── idea-carousel.tsx        # Results carousel
   ├── loading-progress.tsx     # Loading animation
   ├── empty-state.tsx          # Empty state
   └── mobile-edit-drawer.tsx   # Mobile drawer
   ```

3. **Main page:**
   ```
   apps/web/app/(main)/ai-ideas/page.tsx
   ```

### For Users

1. **Start:** Click "Mulai Generate Ide" on empty state
2. **Step 1:** Select jurusan and add interests (use quick add badges)
3. **Step 2:** Choose project type and difficulty
4. **Step 3:** Review and click "Generate Ide Proyek"
5. **Wait:** Watch the progress (10-15 seconds)
6. **Browse:** Navigate through ideas with arrows
7. **Actions:** Like, Skip, or Save ideas

---

## 🎯 Key Features

### Multi-Step Wizard
- 3 clear steps with validation
- Progress indicator
- Back/Next navigation
- Mobile responsive

### Smart Interest Input
- Autocomplete dropdown
- Quick add suggestions
- Visual counter (X/10)
- Keyboard shortcuts

### Carousel Results
- One idea at a time
- Collapsible details
- Navigation arrows + dots
- Action buttons

### Loading Progress
- 4 animated stages
- Progress percentage
- Estimated time
- Engaging visuals

---

## 🔧 Configuration

### Environment Variables
```env
# Backend
GOOGLE_GEMINI_API_KEY=your_api_key_here

# Frontend
NEXT_PUBLIC_AI_FEATURES_ENABLED=true
```

### Storage Keys
```typescript
// localStorage keys
'ai_ideas_form_draft'  // Form data
'ai_project_ideas'     // Generated ideas
```

---

## 📱 Responsive Breakpoints

- **Mobile:** < 768px
- **Tablet:** 768px - 1024px
- **Desktop:** > 1024px

---

## 🎨 Design Tokens

### Colors
- Primary: `bg-primary` (#3B82F6)
- Success: `bg-green-500/10`
- Warning: `bg-yellow-500/10`
- Danger: `bg-red-500/10`

### Spacing
- Container: `max-w-4xl`
- Padding: `p-6 md:p-8`
- Gap: `gap-6`

### Typography
- Heading: `text-2xl md:text-3xl font-bold`
- Body: `text-base md:text-lg`
- Small: `text-sm text-muted-foreground`

---

## 🐛 Common Issues

### Form not saving
- Check localStorage is enabled
- Clear browser cache

### Autocomplete not working
- Verify suggestions array
- Check console for errors

### API errors
- Verify Gemini API key
- Check backend logs
- Test network connection

---

## 📚 Documentation

- **Full Guide:** `docs/ai/ai-ideas-wizard-guide.md`
- **Implementation:** `docs/ai/phase-1-2-implementation-summary.md`
- **API Config:** `docs/ai/ai-features-configuration.md`

---

## ✅ Testing Checklist

### Functional
- [ ] Wizard navigation works
- [ ] Form validation per step
- [ ] Interest autocomplete
- [ ] Generate API call
- [ ] Carousel navigation
- [ ] Actions (Like/Skip/Delete)

### Responsive
- [ ] Mobile (375px)
- [ ] Tablet (768px)
- [ ] Desktop (1024px+)

### Accessibility
- [ ] Keyboard navigation
- [ ] Screen reader
- [ ] Focus indicators
- [ ] Touch targets (44x44px)

---

## 🎓 Tips

1. **Use Quick Add** - Click suggestion badges instead of typing
2. **Keyboard Shortcuts** - Press Enter to add interest
3. **Collapsible Details** - Click "Lihat Detail Lengkap" for more info
4. **Carousel Navigation** - Use arrow keys or click dots
5. **Error Recovery** - Form data is auto-saved, just retry

---

## 📞 Support

Need help? Check:
1. Full documentation in `docs/ai/`
2. Component source code in `apps/web/components/ai/`
3. Console logs for errors
4. Backend logs: `docker logs grafikarsa-backend`

---

**Last Updated:** 7 Maret 2026  
**Version:** 2.0 (Phase 1 & 2 Complete)
