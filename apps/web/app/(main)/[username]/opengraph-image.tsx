import { ImageResponse } from 'next/og';
import { getUserProfile } from '@/lib/api/server-fetch';

export const runtime = 'edge';

export const alt = 'Profile Preview';
export const size = {
    width: 1200,
    height: 630,
};

export const contentType = 'image/png';

function getAbsoluteUrl(url: string | null | undefined): string | undefined {
    if (!url) return undefined;
    if (url.startsWith('http')) return url;
    const baseUrl = process.env.NEXT_PUBLIC_API_URL?.replace('/api/v1', '') || 'http://localhost:8080';
    return `${baseUrl.replace(/\/$/, '')}/${url.replace(/^\//, '')}`;
}

export default async function Image({ params }: { params: { username: string } }) {
    const { username } = await params;
    const user = await getUserProfile(username);

    if (!user) {
        return new ImageResponse(
            (
                <div
                    style={{
                        fontSize: 48,
                        background: 'black',
                        color: 'white',
                        width: '100%',
                        height: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                    }}
                >
                    User Not Found
                </div>
            ),
            { ...size }
        );
    }

    return new ImageResponse(
        (
            <div
                style={{
                    height: '100%',
                    width: '100%',
                    display: 'flex',
                    flexDirection: 'row',
                    background: '#000000',
                    color: '#ffffff',
                    fontFamily: 'sans-serif',
                    position: 'relative',
                    padding: '60px',
                    gap: '80px',
                    alignItems: 'center',
                }}
            >
                {/* Subtle Texture Overlay */}
                <div
                    style={{
                        position: 'absolute',
                        top: 0,
                        left: 0,
                        right: 0,
                        bottom: 0,
                        backgroundImage: 'linear-gradient(rgba(255,255,255,0.03) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.03) 1px, transparent 1px)',
                        backgroundSize: '50px 50px',
                        zIndex: 1,
                    }}
                />

                {/* Left: Identity & Branding */}
                <div style={{ display: 'flex', flexDirection: 'column', flex: 1, zIndex: 10 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '16px', marginBottom: '40px' }}>
                        <div style={{ padding: '6px 12px', border: '1px solid #ffffff', fontSize: '18px', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '2px' }}>
                            {user.role}
                        </div>
                        {user.kelas?.nama && (
                            <span style={{ fontSize: '20px', color: '#71717a', fontWeight: 500, textTransform: 'uppercase' }}>
                                {user.kelas.nama}
                            </span>
                        )}
                    </div>

                    <h1 style={{ fontSize: '96px', fontWeight: 900, margin: 0, lineHeight: 0.9, letterSpacing: '-4px' }}>
                        {user.nama}
                    </h1>
                    <p style={{ fontSize: '36px', color: '#a1a1aa', marginTop: '20px', fontWeight: 400 }}>
                        @{user.username}
                    </p>

                    <div style={{ height: '4px', width: '80px', background: '#fff', margin: '40px 0' }} />

                    <div style={{ display: 'flex', gap: '60px' }}>
                        <div style={{ display: 'flex', flexDirection: 'column' }}>
                            <span style={{ fontSize: '14px', color: '#71717a', textTransform: 'uppercase', letterSpacing: '2px', fontWeight: 700, marginBottom: '4px' }}>Projects</span>
                            <span style={{ fontSize: '48px', fontWeight: 900 }}>{user.portfolio_count || 0}</span>
                        </div>
                        <div style={{ display: 'flex', flexDirection: 'column' }}>
                            <span style={{ fontSize: '14px', color: '#71717a', textTransform: 'uppercase', letterSpacing: '2px', fontWeight: 700, marginBottom: '4px' }}>Followers</span>
                            <span style={{ fontSize: '48px', fontWeight: 900 }}>{user.follower_count || 0}</span>
                        </div>
                    </div>
                </div>

                {/* Right: Profile Picture Frame */}
                <div style={{ display: 'flex', position: 'relative', zIndex: 10 }}>
                    {/* Geometric frame */}
                    <div style={{
                        position: 'absolute',
                        top: '-20px',
                        right: '-20px',
                        width: '320px',
                        height: '320px',
                        border: '2px solid rgba(255,255,255,0.2)',
                    }} />

                    <img
                        src={getAbsoluteUrl(user.avatar_url) || `https://api.dicebear.com/7.x/avataaars/svg?seed=${username}`}
                        alt=""
                        style={{
                            width: '320px',
                            height: '320px',
                            border: '2px solid #ffffff',
                        }}
                    />

                    {/* Branding Logo at bottom right of image */}
                    <div style={{
                        position: 'absolute',
                        bottom: '-60px',
                        right: '-60px',
                        width: '120px',
                        height: '120px',
                        background: '#fff',
                        color: '#000',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: '80px',
                        fontWeight: 900,
                    }}>
                        G
                    </div>
                </div>
            </div>
        ),
        { ...size }
    );
}
