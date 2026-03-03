/**
 * Embed Helpers - URL parsing and conversion utilities for rich embed blocks
 */

// ============================================================================
// FIGMA
// ============================================================================

/**
 * Convert Figma file/proto URL to embeddable URL
 * @example https://www.figma.com/file/xyz... -> https://www.figma.com/embed?embed_host=share&url=...
 */
export function getFigmaEmbedUrl(url: string): string | null {
    if (!url) return null;
    const figmaPattern = /figma\.com\/(file|proto|design|board)\/([a-zA-Z0-9]+)/;
    if (figmaPattern.test(url)) {
        return `https://www.figma.com/embed?embed_host=share&url=${encodeURIComponent(url)}`;
    }
    return null;
}

/**
 * Validate if URL is a valid Figma URL
 */
export function isValidFigmaUrl(url: string): boolean {
    if (!url) return false;
    const figmaPattern = /^https?:\/\/(www\.)?figma\.com\/(file|proto|design|board)\/[a-zA-Z0-9]+/;
    return figmaPattern.test(url);
}

// ============================================================================
// CANVA
// ============================================================================

/**
 * Convert Canva design URL to embeddable URL
 * @example https://www.canva.com/design/xyz... -> https://www.canva.com/design/xyz/view?embed
 */
export function getCanvaEmbedUrl(url: string): string | null {
    if (!url) return null;
    const canvaPattern = /canva\.com\/design\/([a-zA-Z0-9_-]+)/;
    const match = url.match(canvaPattern);
    if (match) {
        // Extract the design ID and construct embed URL
        return `https://www.canva.com/design/${match[1]}/view?embed`;
    }
    return null;
}

/**
 * Validate if URL is a valid Canva URL
 */
export function isValidCanvaUrl(url: string): boolean {
    if (!url) return false;
    const canvaPattern = /^https?:\/\/(www\.)?canva\.com\/design\/[a-zA-Z0-9_-]+/;
    return canvaPattern.test(url);
}

// ============================================================================
// GOOGLE SLIDES / PPT
// ============================================================================

/**
 * Convert Google Slides URL to embeddable URL
 * @example https://docs.google.com/presentation/d/xyz... -> ...embed
 */
export function getGoogleSlidesEmbedUrl(url: string): string | null {
    if (!url) return null;
    const slidesPattern = /docs\.google\.com\/presentation\/d\/([a-zA-Z0-9_-]+)/;
    const match = url.match(slidesPattern);
    if (match) {
        return `https://docs.google.com/presentation/d/${match[1]}/embed?start=false&loop=false&delayms=3000`;
    }
    return null;
}

/**
 * Validate if URL is a valid Google Slides URL
 */
export function isValidGoogleSlidesUrl(url: string): boolean {
    if (!url) return false;
    const slidesPattern = /^https?:\/\/docs\.google\.com\/presentation\/d\/[a-zA-Z0-9_-]+/;
    return slidesPattern.test(url);
}

/**
 * Get Office Online viewer URL for uploaded PPT/PPTX files
 * Note: File must be publicly accessible
 */
export function getOfficeViewerUrl(fileUrl: string): string {
    return `https://view.officeapps.live.com/op/embed.aspx?src=${encodeURIComponent(fileUrl)}`;
}

// ============================================================================
// GOOGLE DOCS VIEWER (For PDF, DOC, DOCX)
// ============================================================================

/**
 * Get Google Docs viewer URL for documents
 * Works well for PDF, DOC, DOCX files
 */
export function getGoogleDocsViewerUrl(fileUrl: string): string {
    return `https://docs.google.com/viewer?url=${encodeURIComponent(fileUrl)}&embedded=true`;
}

// ============================================================================
// WEBSITE EMBED
// ============================================================================

/**
 * Whitelist of domains that are safe to embed in iframes
 */
const ALLOWED_IFRAME_DOMAINS = [
    // Design Tools
    'figma.com',
    'canva.com',
    'whimsical.com',
    'miro.com',
    'pitch.com',
    'prezi.com',
    'slides.com',

    // Code Playgrounds
    'codepen.io',
    'codesandbox.io',
    'stackblitz.com',
    'jsfiddle.net',
    'replit.com',

    // 3D & Creative
    'spline.design',
    'sketchfab.com',
    'rive.app',

    // Productivity
    'notion.so',
    'docs.google.com',
    'sheets.google.com',
    'drive.google.com',
    'airtable.com',

    // Video
    'loom.com',
    'vimeo.com',

    // Audio
    'open.spotify.com',
    'soundcloud.com',

    // Maps
    'maps.google.com',
    'google.com/maps',
];



/**
 * Check if a URL's domain is in the allowed whitelist
 */
export function isAllowedDomain(url: string): boolean {
    if (!url) return false;
    try {
        const hostname = new URL(url).hostname;
        return ALLOWED_IFRAME_DOMAINS.some(domain =>
            hostname === domain || hostname.endsWith('.' + domain)
        );
    } catch {
        return false;
    }
}

/**
 * Validate URL format
 */
export function isValidUrl(url: string): boolean {
    if (!url) return false;
    try {
        new URL(url);
        return url.startsWith('http://') || url.startsWith('https://');
    } catch {
        return false;
    }
}

// ============================================================================
// FILE UPLOAD HELPERS
// ============================================================================

const DOCUMENT_MIME_TYPES = {
    pdf: ['application/pdf'],
    ppt: [
        'application/vnd.ms-powerpoint',
        'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    ],
    doc: [
        'application/msword',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    ],
};

const DOCUMENT_EXTENSIONS = {
    pdf: ['.pdf'],
    ppt: ['.ppt', '.pptx'],
    doc: ['.doc', '.docx'],
};

/**
 * Get accepted file types for a block type
 */
export function getAcceptedFileTypes(blockType: 'pdf' | 'ppt' | 'doc'): string {
    const mimes = DOCUMENT_MIME_TYPES[blockType] || [];
    const exts = DOCUMENT_EXTENSIONS[blockType] || [];
    return [...mimes, ...exts].join(',');
}

/**
 * Validate file type for a block
 */
export function isValidFileType(file: File, blockType: 'pdf' | 'ppt' | 'doc'): boolean {
    const allowedMimes = DOCUMENT_MIME_TYPES[blockType] || [];
    const allowedExts = DOCUMENT_EXTENSIONS[blockType] || [];

    const mimeOk = allowedMimes.includes(file.type);
    const extOk = allowedExts.some(ext => file.name.toLowerCase().endsWith(ext));

    return mimeOk || extOk;
}

/**
 * Format file size for display
 */
export function formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

/**
 * Maximum file size for documents (20MB)
 */
export const MAX_DOCUMENT_SIZE = 20 * 1024 * 1024;
