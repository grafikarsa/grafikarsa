# Debug Mode - Empty State Testing

⚠️ **DEVELOPMENT ONLY - NOT FOR PRODUCTION**

## Overview

Debug mode allows developers to test empty states across all admin panel pages without needing to clear the database. This is useful for:
- UI/UX testing of empty states
- Screenshot documentation
- Design review
- Component testing

## Setup

### 1. Enable Debug Mode

Add to your `.env.local` file:

```bash
NEXT_PUBLIC_DEBUG_MODE=true
```

### 2. Restart Development Server

```bash
npm run dev
```

## Features

When debug mode is enabled:

1. **Empty State Override**: All admin pages will show their empty state UI regardless of actual data
2. **Debug Banner**: A warning banner appears at the top of each page indicating debug mode is active
3. **Development Only**: Debug mode is automatically disabled in production builds

## Admin Pages with Debug Mode

The following admin pages support debug mode:

### Data Management
- ✅ **Penilaian Portfolio** (`/admin/assessments`) - Shows empty portfolio assessment list
- ✅ **Kelola Portfolio** (`/admin/portfolios`) - Shows empty portfolio list
- 🔄 **Kelola User** (`/admin/users`) - Coming soon
- 🔄 **Kelola Tag** (`/admin/tags`) - Coming soon
- 🔄 **Kelola Series** (`/admin/series`) - Coming soon

### Academic Data
- 🔄 **Tahun Ajaran** (`/admin/academic-years`) - Coming soon
- 🔄 **Jurusan** (`/admin/majors`) - Coming soon
- 🔄 **Kelas** (`/admin/classes`) - Coming soon

### Assessment System
- 🔄 **Metrik Penilaian** (`/admin/assessment-metrics`) - Coming soon

### Content Moderation
- 🔄 **Moderasi** (`/admin/moderation`) - Coming soon
- 🔄 **Feedback** (`/admin/feedback`) - Coming soon

### System
- 🔄 **Changelog** (`/admin/changelogs`) - Coming soon
- 🔄 **Special Roles** (`/admin/special-roles`) - Coming soon
- 🔄 **Import Data** (`/admin/import`) - Coming soon

## Implementation Guide

### For Developers

To add debug mode support to a new admin page:

1. **Import debug utilities**:
```typescript
import { getDebugEmptyState } from '@/lib/utils/debug';
import { DebugBanner } from '@/components/admin/debug-banner';
```

2. **Add debug mode check**:
```typescript
const debugMode = getDebugEmptyState();
const displayData = debugMode ? [] : actualData;
```

3. **Add debug banner**:
```tsx
{debugMode && <DebugBanner pageName="Your Page Name" />}
```

4. **Use display data in render**:
```tsx
{displayData.length === 0 ? (
  // Empty state UI
) : (
  // Normal data display
)}
```

### Example Implementation

```typescript
export default function AdminUsersPage() {
  const { data } = useQuery({
    queryKey: ['admin-users'],
    queryFn: () => adminUsersApi.getUsers(),
  });

  const users = data?.data || [];
  
  // Debug mode: Force empty state
  const debugMode = getDebugEmptyState();
  const displayUsers = debugMode ? [] : users;

  return (
    <div>
      {/* Debug Banner */}
      {debugMode && <DebugBanner pageName="Kelola User" />}
      
      {/* Content */}
      {displayUsers.length === 0 ? (
        <EmptyState />
      ) : (
        <UserList users={displayUsers} />
      )}
    </div>
  );
}
```

## Security

- Debug mode is **automatically disabled** in production (`NODE_ENV=production`)
- The `isDebugMode()` function returns `false` in production regardless of env variable
- Debug banner and empty state override only work in development

## Troubleshooting

### Debug mode not working?

1. Check `.env.local` has `NEXT_PUBLIC_DEBUG_MODE=true`
2. Restart the development server
3. Verify you're in development mode (`NODE_ENV=development`)
4. Check browser console for any errors

### Debug mode showing in production?

This should never happen. If it does:
1. Check `NODE_ENV` is set to `production`
2. Verify the build process
3. Check the `isDebugMode()` function in `lib/utils/debug.ts`

## Best Practices

1. **Never commit** `.env.local` with debug mode enabled
2. **Always disable** debug mode before taking production screenshots
3. **Document** empty states when adding new pages
4. **Test** both empty and populated states during development
5. **Review** empty state UI/UX regularly

## Related Files

- `lib/utils/debug.ts` - Debug utility functions
- `components/admin/debug-banner.tsx` - Debug warning banner component
- `.env.example` - Environment variable template
- `.env.local` - Local development environment (not committed)
