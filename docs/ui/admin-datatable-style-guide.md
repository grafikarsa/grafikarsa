# Admin DataTable - Unified Visual Style Guide

## Overview
Panduan desain terpadu untuk semua data table di admin dashboard Grafikarsa. Tujuan: konsistensi visual, UX yang baik, dan maintainability.

---

## 🎨 Design Principles

### 1. Consistency (Konsistensi)
- Semua table menggunakan komponen dan pattern yang sama
- Spacing, typography, dan colors yang seragam
- Interaction patterns yang predictable

### 2. Clarity (Kejelasan)
- Hierarchy yang jelas antara header, body, dan actions
- Status dan badges yang mudah dibedakan
- Readable typography dengan contrast yang baik

### 3. Efficiency (Efisiensi)
- Quick actions accessible via dropdown menu
- Inline status badges untuk quick scanning
- Responsive design untuk berbagai screen sizes

---

## 📐 Layout Structure

### Container
```tsx
<Card className="overflow-hidden p-0">
  <Table>
    {/* Content */}
  </Table>
</Card>
```

**Rules:**
- Card wrapper dengan `overflow-hidden` dan `p-0`
- No padding pada Card, padding handled by Table cells
- Border radius dari Card component

---

## 🎯 Table Header

### Visual Style
```tsx
<TableHeader>
  <TableRow className="border-b bg-muted/30">
    <TableHead className="...">Column Name</TableHead>
  </TableRow>
</TableHeader>
```

**Rules:**
- Background: `bg-muted/30` (subtle, harmonious distinction)
- Border: `border-b` untuk visual separation
- Font weight: Default (medium) dari TableHead
- Text color: Default muted-foreground
- Sticky header: Optional, untuk long tables

**Rationale:**
- Subtle background that doesn't clash with overall design
- Maintains visual hierarchy without being too prominent
- Works well in both light and dark modes
- Border-bottom adds extra definition between header and body

### Column Width Guidelines
- Index/Number: `w-12` atau `w-16`
- Actions: `w-[80px]` atau `w-[100px]`
- Status/Badge: `w-[100px]` atau `w-[120px]`
- Date: `w-[120px]` atau `w-[150px]`
- Main content: No fixed width (flex)
- Text alignment: 
  - Left: Default untuk text
  - Center: Untuk index numbers
  - Right: Untuk actions

---

## 📊 Table Body

### Row Style
```tsx
<TableRow className="group hover:bg-muted/50">
  {/* Cells */}
</TableRow>
```

**Rules:**
- Hover state: `hover:bg-muted/50`
- Group class: Untuk child hover effects
- No zebra striping (cleaner look)
- Border: Default dari TableRow

### Cell Padding
- Default: TableCell default padding
- Consistent vertical alignment: `items-center`

---

## 🏷️ Status Badges

### Color System
```tsx
const statusStyles = {
  // Success states
  active: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  published: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  resolved: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  
  // Warning states
  pending: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
  pending_review: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400',
  
  // Info states
  read: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  draft: 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
  
  // Danger states
  inactive: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  rejected: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  
  // Neutral states
  archived: 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400',
};
```

**Rules:**
- Use Badge component with custom className
- Include icon when meaningful (optional)
- Capitalize text appropriately
- Consistent sizing: Default badge size

### Badge with Icon
```tsx
<Badge className={statusStyles[status]}>
  <Icon className="h-3 w-3 mr-1" />
  Status Text
</Badge>
```

---

## 👤 User Display

### Avatar + Name Pattern
```tsx
<div className="flex items-center gap-3">
  <Avatar className="h-10 w-10 border">
    <AvatarImage src={user.avatar_url} alt={user.nama} />
    <AvatarFallback className="text-sm font-medium">
      {user.nama?.charAt(0)}
    </AvatarFallback>
  </Avatar>
  <div className="min-w-0">
    <p className="truncate font-medium">{user.nama}</p>
    <p className="truncate text-sm text-muted-foreground">
      @{user.username}
    </p>
  </div>
</div>
```

**Rules:**
- Avatar size: `h-10 w-10` untuk main display, `h-6 w-6` untuk compact
- Border on avatar: `border` class
- Truncate long names: `truncate` class
- Secondary info: `text-sm text-muted-foreground`
- Min-width-0 on text container untuk proper truncation

### Compact User Display
```tsx
<div className="flex items-center gap-2">
  <Avatar className="h-6 w-6">
    <AvatarImage src={user.avatar_url} />
    <AvatarFallback className="text-xs">{user.nama?.charAt(0)}</AvatarFallback>
  </Avatar>
  <span className="text-sm truncate max-w-[150px]">{user.nama}</span>
</div>
```

---

## ⚡ Actions Column

### Dropdown Menu Pattern
```tsx
<TableCell className="text-right">
  <DropdownMenu>
    <DropdownMenuTrigger asChild>
      <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
        <MoreVertical className="h-4 w-4" />
        <span className="sr-only">Menu</span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      <DropdownMenuItem onClick={handleView}>
        <Eye className="mr-2 h-4 w-4" />
        Lihat Detail
      </DropdownMenuItem>
      <DropdownMenuItem onClick={handleEdit}>
        <Pencil className="mr-2 h-4 w-4" />
        Edit
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem 
        onClick={handleDelete}
        className="text-destructive focus:text-destructive"
      >
        <Trash2 className="mr-2 h-4 w-4" />
        Hapus
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</TableCell>
```

**Rules:**
- Use MoreVertical icon (3 dots vertical)
- Button: `variant="ghost" size="sm" className="h-8 w-8 p-0"`
- Align dropdown: `align="end"`
- Icon size in menu: `h-4 w-4 mr-2`
- Destructive actions: Red text with separator above
- SR-only text for accessibility

---

## 📅 Date Display

### Format
```tsx
<TableCell>
  <span className="text-sm text-muted-foreground">
    {formatDate(date)}
  </span>
</TableCell>
```

**Rules:**
- Use `formatDate()` utility function
- Text size: `text-sm`
- Color: `text-muted-foreground`
- No icon needed (context is clear)

---

## 🔢 Index/Number Column

### Style
```tsx
<TableCell className="text-center text-muted-foreground">
  {(page - 1) * limit + index + 1}
</TableCell>
```

**Rules:**
- Text alignment: `text-center`
- Color: `text-muted-foreground`
- Calculate based on pagination
- Column width: `w-12` atau `w-16`

---

## 📱 Responsive Behavior

### Mobile Considerations
- Hide less important columns on mobile
- Use responsive grid for cards on small screens
- Ensure touch targets are at least 44x44px
- Consider switching to card layout on mobile

### Breakpoint Strategy
```tsx
<TableHead className="hidden md:table-cell">Optional Column</TableHead>
<TableCell className="hidden md:table-cell">Optional Data</TableCell>
```

---

## 🎭 Empty States

### Pattern
```tsx
<Card className="border-dashed py-16">
  <div className="flex flex-col items-center justify-center">
    <div className="rounded-full bg-muted p-4">
      <Icon className="h-8 w-8 text-muted-foreground" />
    </div>
    <h3 className="mt-4 text-lg font-semibold">Tidak ada data</h3>
    <p className="mt-1 text-sm text-muted-foreground">
      Deskripsi atau call-to-action
    </p>
  </div>
</Card>
```

**Rules:**
- Dashed border card
- Centered content with icon
- Clear messaging
- Optional CTA button

---

## 🔄 Loading States

### Skeleton Pattern
```tsx
<div className="space-y-4 p-6">
  {[...Array(8)].map((_, i) => (
    <Skeleton key={i} className="h-16 w-full" />
  ))}
</div>
```

**Rules:**
- Show skeleton rows matching expected content
- Maintain layout structure
- Use Skeleton component from shadcn/ui

---

## 📄 Pagination

### Pattern
```tsx
{pagination && pagination.total_pages > 1 && (
  <div className="flex items-center justify-between">
    <p className="text-sm text-muted-foreground">
      Menampilkan {items.length} dari {pagination.total_count} items
    </p>
    <div className="flex items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        onClick={() => setPage((p) => Math.max(1, p - 1))}
        disabled={page === 1}
      >
        Previous
      </Button>
      <span className="text-sm">
        Halaman {page} dari {pagination.total_pages}
      </span>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setPage((p) => Math.min(pagination.total_pages, p + 1))}
        disabled={page === pagination.total_pages}
      >
        Next
      </Button>
    </div>
  </div>
)}
```

**Rules:**
- Show only if more than 1 page
- Display current page and total
- Show item count on left
- Disable buttons at boundaries

---

## 🎨 Color Palette Reference

### Status Colors
- **Success/Active**: Green-100/700 (light), Green-900/400 (dark)
- **Warning/Pending**: Yellow-100/700 (light), Yellow-900/400 (dark)
- **Info/Read**: Blue-100/700 (light), Blue-900/400 (dark)
- **Danger/Error**: Red-100/700 (light), Red-900/400 (dark)
- **Neutral/Draft**: Gray-100/700 (light), Gray-800/300 (dark)

### Text Colors
- **Primary**: Default foreground
- **Secondary**: `text-muted-foreground`
- **Emphasis**: `font-medium` or `font-semibold`

---

## ✅ Implementation Checklist

Untuk setiap data table, pastikan:

- [ ] Card wrapper dengan `overflow-hidden p-0`
- [ ] TableHeader dengan `border-b bg-muted/30`
- [ ] TableRow dengan `group hover:bg-muted/50`
- [ ] Status badges menggunakan unified color system
- [ ] User display dengan Avatar + truncated text
- [ ] Actions column dengan MoreVertical dropdown
- [ ] Date display dengan `text-sm text-muted-foreground`
- [ ] Index column dengan `text-center text-muted-foreground`
- [ ] Empty state dengan dashed border card
- [ ] Loading state dengan skeleton
- [ ] Pagination dengan item count
- [ ] Responsive considerations
- [ ] Accessibility (SR-only text, proper labels)

---

## 🚀 Migration Strategy

1. **Create shared components** (optional):
   - `DataTableContainer`
   - `DataTableHeader`
   - `DataTableRow`
   - `StatusBadge`
   - `UserCell`
   - `ActionsCell`

2. **Update existing tables**:
   - Users table ✓
   - Portfolios table (card-based, different pattern)
   - Assessments table (card-based, different pattern)
   - Feedback table ✓
   - Other admin tables

3. **Test thoroughly**:
   - Visual consistency
   - Responsive behavior
   - Dark mode
   - Accessibility

---

## 📝 Notes

- Portfolios dan Assessments menggunakan card-based layout, bukan table
- Feedback table sudah mengikuti pattern yang baik
- Users table perlu sedikit penyesuaian untuk konsistensi penuh
- Pertimbangkan membuat shared components untuk reusability

---

**Last Updated**: 2026-03-06
**Version**: 1.2.0

---

## 📝 Changelog

### v1.2.0 (2026-03-06)
- Updated header background to `border-b bg-muted/30` for more subtle, harmonious look
- Previous `bg-slate-100 dark:bg-slate-800` was too harsh and didn't fit overall style
- New color maintains visual hierarchy while blending better with design system

### v1.1.0 (2026-03-06)
- Updated header background dari `bg-muted/50` ke `bg-slate-100 dark:bg-slate-800 border-b`
- Added border-bottom to header untuk extra visual separation
- Improved contrast between header and page background

### v1.0.0 (2026-03-06)
- Initial unified style guide creation
- Defined all core patterns and components
