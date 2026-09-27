import { ApiResponse, User } from '@/lib/types';

// SSR: prefer internal Docker hostname (http://backend:8080) to avoid
// EAI_AGAIN when resolving the public API hostname from inside the container.
// Client components tetap pakai NEXT_PUBLIC_API_URL via lib/api/client.ts.
const API_BASE_URL =
    process.env.INTERNAL_API_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export async function getUserProfile(username: string): Promise<User | null> {
    try {
        const res = await fetch(`${API_BASE_URL}/users/${encodeURIComponent(username)}`, {
            next: { revalidate: 60 }, // Cache for 1 minute
        });

        if (!res.ok) return null;

        const data: ApiResponse<User> = await res.json();
        return data.data ?? null;
    } catch (error) {
        console.error(`Error fetching user profile for ${username}:`, error);
        return null;
    }
}

export async function getPortfolioDetail(username: string, slug: string): Promise<any | null> {
    try {
        const res = await fetch(`${API_BASE_URL}/portfolios/${encodeURIComponent(slug)}?username=${encodeURIComponent(username)}`, {
            next: { revalidate: 60 },
        });

        if (!res.ok) return null;

        const data: ApiResponse<any> = await res.json();
        return data.data ?? null;
    } catch (error) {
        console.error(`Error fetching portfolio ${slug} for ${username}:`, error);
        return null;
    }
}
