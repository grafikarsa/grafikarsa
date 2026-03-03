import { ImageResponse } from 'next/og';

export const runtime = 'edge';

export const alt = 'Grafikarsa - Platform Portofolio SMKN 4 Malang';
export const size = {
    width: 1200,
    height: 630,
};

export const contentType = 'image/png';

export default async function Image() {
    return new ImageResponse(
        (
            <div
                style={{
                    height: '100%',
                    width: '100%',
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: '#000000',
                    color: '#ffffff',
                    position: 'relative',
                    fontFamily: 'sans-serif',
                    padding: '80px',
                }}
            >
                {/* Subtle Grid Pattern Overlay */}
                <div
                    style={{
                        position: 'absolute',
                        top: 0,
                        left: 0,
                        right: 0,
                        bottom: 0,
                        backgroundImage: 'radial-gradient(circle, #ffffff 1px, transparent 1px)',
                        backgroundSize: '40px 40px',
                        opacity: 0.05,
                    }}
                />

                {/* Border Frame */}
                <div
                    style={{
                        position: 'absolute',
                        top: '40px',
                        left: '40px',
                        right: '40px',
                        bottom: '40px',
                        border: '1px solid rgba(255, 255, 255, 0.1)',
                        pointerEvents: 'none',
                    }}
                />

                <div
                    style={{
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        justifyContent: 'center',
                        zIndex: 10,
                    }}
                >
                    {/* Logo */}
                    <div
                        style={{
                            width: '140px',
                            height: '140px',
                            border: '4px solid #ffffff',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            marginBottom: '48px',
                            background: '#000000',
                        }}
                    >
                        <span
                            style={{
                                fontSize: '90px',
                                fontWeight: 900,
                                letterSpacing: '-4px',
                                color: '#ffffff',
                            }}
                        >
                            G
                        </span>
                    </div>

                    <h1
                        style={{
                            fontSize: '110px',
                            fontWeight: 900,
                            margin: 0,
                            lineHeight: 1,
                            letterSpacing: '-6px',
                            textTransform: 'uppercase',
                        }}
                    >
                        Grafikarsa
                    </h1>

                    <div
                        style={{
                            height: '2px',
                            width: '120px',
                            background: '#ffffff',
                            margin: '32px 0',
                        }}
                    />

                    <p
                        style={{
                            fontSize: '28px',
                            color: '#a1a1aa',
                            fontWeight: 400,
                            letterSpacing: '1px',
                            maxWidth: '800px',
                            textAlign: 'center',
                            textTransform: 'uppercase',
                        }}
                    >
                        Platform Katalog Portofolio & Social Network Siswa
                    </p>
                </div>

                {/* Corner Accents */}
                <div style={{ position: 'absolute', top: '35px', left: '35px', width: '20px', height: '2px', background: '#fff' }} />
                <div style={{ position: 'absolute', top: '35px', left: '35px', width: '2px', height: '20px', background: '#fff' }} />

                <div style={{ position: 'absolute', bottom: '35px', right: '35px', width: '20px', height: '2px', background: '#fff' }} />
                <div style={{ position: 'absolute', bottom: '35px', right: '35px', width: '2px', height: '20px', background: '#fff' }} />
            </div>
        ),
        { ...size }
    );
}
