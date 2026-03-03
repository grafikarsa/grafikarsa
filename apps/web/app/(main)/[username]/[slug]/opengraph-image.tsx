import { ImageResponse } from 'next/og';
import { getPortfolioDetail } from '@/lib/api/server-fetch';

export const runtime = 'edge';

export const alt = 'Portfolio Preview';
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

export default async function Image({ params }: { params: { username: string; slug: string } }) {
    const { username, slug } = await params;
    const portfolio = await getPortfolioDetail(username, slug);

    if (!portfolio) {
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
                    Portfolio Not Found
                </div>
            ),
            { ...size }
        );
    }

    const hasThumbnail = !!portfolio.thumbnail_url;

    return new ImageResponse(
        (
            <div
                style={{
                    height: '100%',
                    width: '100%',
                    display: 'flex',
                    flexDirection: 'column',
                    background: '#000000',
                    color: 'white',
                    position: 'relative',
                    fontFamily: 'sans-serif',
                    overflow: 'hidden',
                    padding: '60px',
                }}
            >
                {/* Background Layer: High Contrast Grayscale Thumbnail */}
                {hasThumbnail && (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                        src={getAbsoluteUrl(portfolio.thumbnail_url)}
                        alt=""
                        style={{
                            position: 'absolute',
                            top: 0,
                            left: 0,
                            width: '100%',
                            height: '100%',
                            opacity: 0.4,
                        }}
                    />
                )}

                {/* Dark Vignette Overlay */}
                <div
                    style={{
                        position: 'absolute',
                        top: 0,
                        left: 0,
                        width: '100%',
                        height: '100%',
                        background: 'linear-gradient(to bottom, transparent 0%, rgba(0,0,0,0.8) 100%)',
                    }}
                />

                {/* Content Section */}
                <div
                    style={{
                        display: 'flex',
                        flexDirection: 'column',
                        justifyContent: 'flex-end',
                        height: '100%',
                        zIndex: 10,
                    }}
                >
                    <div style={{ display: 'flex', gap: '8px', marginBottom: '24px' }}>
                        <div style={{ padding: '4px 12px', border: '1px solid #fff', fontSize: '14px', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '2px' }}>
                            PORTFOLIO
                        </div>
                        {portfolio.series && (
                            <div style={{ padding: '4px 12px', background: '#fff', color: '#000', fontSize: '14px', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '2px' }}>
                                SERIES
                            </div>
                        )}
                    </div>

                    <h1
                        style={{
                            fontSize: '90px',
                            fontWeight: 900,
                            margin: 0,
                            lineHeight: 1,
                            letterSpacing: '-4px',
                            maxWidth: '1000px',
                            marginBottom: '40px',
                            textTransform: 'uppercase',
                        }}
                    >
                        {portfolio.judul}
                    </h1>

                    {/* Bottom Info Row */}
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderTop: '2px solid #fff', paddingTop: '32px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '20px' }}>
                            {/* Author Circle */}
                            <div style={{ width: '60px', height: '60px', border: '2px solid #fff', overflow: 'hidden', background: '#333' }}>
                                <img
                                    src={getAbsoluteUrl(portfolio.user?.avatar_url) || `https://api.dicebear.com/7.x/avataaars/svg?seed=${username}`}
                                    alt=""
                                    style={{ width: '100%', height: '100%' }}
                                />
                            </div>
                            <div style={{ display: 'flex', flexDirection: 'column' }}>
                                <span style={{ fontSize: '24px', fontWeight: 800 }}>{portfolio.user?.nama}</span>
                                <span style={{ fontSize: '16px', color: '#a1a1aa', fontWeight: 600 }}>@{portfolio.user?.username}</span>
                            </div>
                        </div>

                        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
                            <div style={{ width: '40px', height: '40px', border: '2px solid #fff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                <span style={{ fontSize: '24px', fontWeight: 900 }}>G</span>
                            </div>
                            <span style={{ fontSize: '28px', fontWeight: 900, letterSpacing: '-1px' }}>GRAFIKARSA</span>
                        </div>
                    </div>
                </div>

                {/* Decorative Lines */}
                <div style={{ position: 'absolute', top: '0', right: '100px', width: '2px', height: '100px', background: 'rgba(255,255,255,0.2)' }} />
                <div style={{ position: 'absolute', top: '0', right: '120px', width: '2px', height: '60px', background: 'rgba(255,255,255,0.2)' }} />
            </div>
        ),
        { ...size }
    );
}
