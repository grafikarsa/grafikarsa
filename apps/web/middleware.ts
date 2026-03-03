import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// Note: Auth protection is handled client-side in each page component
// because we use localStorage for token storage (not cookies).
// This middleware only handles basic routing logic.

export function middleware(request: NextRequest) {
  // Middleware runs on server, can't access localStorage
  // All auth checks are done client-side in page components
  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Match all request paths except:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public files (images, etc.)
     */
    '/((?!api|_next/static|_next/image|favicon.ico|.*\\..*|_next).*)',
  ],
};
