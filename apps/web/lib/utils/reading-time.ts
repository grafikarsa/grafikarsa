import { ContentBlock } from "@/lib/types";

export function calculateReadingTime(blocks?: ContentBlock[]): number {
    if (!blocks || blocks.length === 0) return 1;

    let textContent = '';

    blocks.forEach(block => {
        if (block.block_type === 'text' && block.payload) {
            // cast payload to any to safely access content
            const payload = block.payload as any;
            if (typeof payload.content === 'string') {
                // Remove HTML tags
                const text = payload.content.replace(/<[^>]*>?/gm, '');
                textContent += text + ' ';
            }
        }
    });

    const words = textContent.trim().split(/\s+/).length;
    const wordsPerMinute = 200; // Average reading speed
    const minutes = Math.ceil(words / wordsPerMinute);

    return minutes < 1 ? 1 : minutes;
}
