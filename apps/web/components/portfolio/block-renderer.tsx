import { useEffect } from 'react';
import {
  ContentBlock,
  TextBlockPayload,
  ImageBlockPayload,
  TableBlockPayload,
  YoutubeBlockPayload,
  ButtonBlockPayload,
  EmbedBlockPayload,
  FigmaBlockPayload,
  CanvaBlockPayload,
  PPTBlockPayload,
  PDFBlockPayload,
  DocBlockPayload,
} from '@/lib/types';
import { Button } from '@/components/ui/button';
import { ExternalLink, Download, FileText, Globe } from 'lucide-react';
import { ImageWithLightbox } from '@/components/ui/lightbox';
import {
  getFigmaEmbedUrl,
  getCanvaEmbedUrl,
  getGoogleSlidesEmbedUrl,
  getGoogleDocsViewerUrl,
  isAllowedDomain,
} from '@/lib/utils/embed-helpers';

interface BlockRendererProps {
  blocks: ContentBlock[];
}

export function BlockRenderer({ blocks }: BlockRendererProps) {
  const sortedBlocks = [...blocks].sort((a, b) => a.block_order - b.block_order);

  return (
    <div className="space-y-6">
      {sortedBlocks.map((block) => (
        <RenderBlock key={block.id} block={block} />
      ))}
    </div>
  );
}

function RenderBlock({ block }: { block: ContentBlock }) {
  switch (block.block_type) {
    case 'text':
      return <TextBlock payload={block.payload as TextBlockPayload} />;
    case 'image':
      return <ImageBlock payload={block.payload as ImageBlockPayload} />;
    case 'table':
      return <TableBlock payload={block.payload as TableBlockPayload} />;
    case 'youtube':
      return <YoutubeBlock payload={block.payload as YoutubeBlockPayload} />;
    case 'button':
      return <ButtonBlock payload={block.payload as ButtonBlockPayload} />;
    case 'embed':
      return <EmbedBlock payload={block.payload as EmbedBlockPayload} />;
    // New block types
    case 'figma':
      return <FigmaBlock payload={block.payload as FigmaBlockPayload} />;
    case 'canva':
      return <CanvaBlock payload={block.payload as CanvaBlockPayload} />;
    case 'ppt':
      return <PPTBlock payload={block.payload as PPTBlockPayload} />;
    case 'pdf':
      return <PDFBlock payload={block.payload as PDFBlockPayload} />;
    case 'doc':
      return <DocBlock payload={block.payload as DocBlockPayload} />;
    default:
      return null;
  }
}

function TextBlock({ payload }: { payload: TextBlockPayload }) {
  return (
    <div
      className="prose prose-neutral dark:prose-invert max-w-none"
      dangerouslySetInnerHTML={{ __html: payload.content }}
    />
  );
}

function ImageBlock({ payload }: { payload: ImageBlockPayload }) {
  return (
    <figure className="space-y-2">
      <div className="relative aspect-video overflow-hidden rounded-lg bg-muted">
        <ImageWithLightbox
          src={payload.url}
          alt={payload.caption || 'Image'}
          className="h-full w-full"
        />
      </div>
      {payload.caption && (
        <figcaption className="text-center text-sm text-muted-foreground">{payload.caption}</figcaption>
      )}
    </figure>
  );
}

function TableBlock({ payload }: { payload: TableBlockPayload }) {
  const headers = payload.headers || [];
  const rows = payload.rows || [];

  if (headers.length === 0 && rows.length === 0) {
    return null;
  }

  return (
    <div className="overflow-x-auto rounded-lg border">
      <table className="w-full">
        {headers.length > 0 && (
          <thead className="bg-muted">
            <tr>
              {headers.map((header, i) => (
                <th key={i} className="px-4 py-2 text-left font-medium">
                  {header}
                </th>
              ))}
            </tr>
          </thead>
        )}
        <tbody>
          {rows.map((row, i) => (
            <tr key={i} className="border-t">
              {(row || []).map((cell, j) => (
                <td key={j} className="px-4 py-2">
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function YoutubeBlock({ payload }: { payload: YoutubeBlockPayload }) {
  return (
    <div className="space-y-2">
      <div className="relative aspect-video overflow-hidden rounded-lg">
        <iframe
          src={`https://www.youtube.com/embed/${payload.video_id}`}
          title={payload.title || 'YouTube video'}
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
          allowFullScreen
          className="absolute inset-0 h-full w-full"
        />
      </div>
      {payload.title && (
        <p className="text-center text-sm text-muted-foreground">{payload.title}</p>
      )}
    </div>
  );
}

function ButtonBlock({ payload }: { payload: ButtonBlockPayload }) {
  return (
    <div className="flex justify-center">
      <a href={payload.url} target="_blank" rel="noopener noreferrer">
        <Button className="gap-2">
          {payload.text}
          <ExternalLink className="h-4 w-4" />
        </Button>
      </a>
    </div>
  );
}

function EmbedBlock({ payload }: { payload: EmbedBlockPayload }) {
  return (
    <div className="space-y-2">
      <div
        className="overflow-hidden rounded-lg"
        dangerouslySetInnerHTML={{ __html: payload.html }}
      />
      {payload.title && (
        <p className="text-center text-sm text-muted-foreground">{payload.title}</p>
      )}
    </div>
  );
}

// ============================================================================
// NEW BLOCK TYPES
// ============================================================================

function FigmaBlock({ payload }: { payload: FigmaBlockPayload }) {
  const embedUrl = getFigmaEmbedUrl(payload.url);
  if (!embedUrl) {
    return (
      <p className="text-sm text-muted-foreground italic">
        URL Figma tidak valid
      </p>
    );
  }

  return (
    <div className="space-y-2">
      <div className="aspect-video w-full overflow-hidden rounded-lg border shadow-sm">
        <iframe
          src={embedUrl}
          className="h-full w-full"
          allowFullScreen
          loading="lazy"
          title={payload.title || 'Figma Design'}
        />
      </div>
      {payload.title && (
        <p className="text-center text-sm text-muted-foreground">{payload.title}</p>
      )}
    </div>
  );
}

function CanvaBlock({ payload }: { payload: CanvaBlockPayload }) {
  const embedUrl = getCanvaEmbedUrl(payload.url);
  if (!embedUrl) {
    return (
      <p className="text-sm text-muted-foreground italic">
        URL Canva tidak valid
      </p>
    );
  }

  return (
    <div className="space-y-2">
      <div className="aspect-video w-full overflow-hidden rounded-lg border shadow-sm">
        <iframe
          src={embedUrl}
          className="h-full w-full"
          allowFullScreen
          loading="lazy"
          title={payload.title || 'Canva Design'}
        />
      </div>
      {payload.title && (
        <p className="text-center text-sm text-muted-foreground">{payload.title}</p>
      )}
    </div>
  );
}

function PPTBlock({ payload }: { payload: PPTBlockPayload }) {
  const embedUrl = getGoogleSlidesEmbedUrl(payload.url);
  if (!embedUrl) {
    return (
      <p className="text-sm text-muted-foreground italic">
        URL Presentasi tidak valid
      </p>
    );
  }

  return (
    <div className="space-y-2">
      <div className="aspect-video w-full overflow-hidden rounded-lg border shadow-sm">
        <iframe
          src={embedUrl}
          className="h-full w-full"
          allowFullScreen
          loading="lazy"
          title={payload.title || 'Presentation'}
        />
      </div>
      {payload.title && (
        <p className="text-center text-sm text-muted-foreground">{payload.title}</p>
      )}
    </div>
  );
}

function PDFBlock({ payload }: { payload: PDFBlockPayload }) {
  if (!payload.url) {
    return null;
  }

  return (
    <div className="space-y-3">
      {payload.title && (
        <h4 className="font-medium">{payload.title}</h4>
      )}
      <div className="aspect-[3/4] max-h-[600px] w-full overflow-hidden rounded-lg border shadow-sm">
        <iframe
          src={`${payload.url}#toolbar=1&navpanes=0`}
          className="h-full w-full"
          title={payload.file_name || 'PDF Document'}
        />
      </div>
      <a
        href={payload.url}
        download={payload.file_name}
        className="inline-flex items-center gap-2 text-sm text-primary hover:underline"
      >
        <Download className="h-4 w-4" />
        Download PDF {payload.file_name && `(${payload.file_name})`}
      </a>
    </div>
  );
}

function DocBlock({ payload }: { payload: DocBlockPayload }) {
  if (!payload.url) {
    return null;
  }

  const viewerUrl = getGoogleDocsViewerUrl(payload.url);

  return (
    <div className="space-y-3">
      {payload.title && (
        <h4 className="font-medium">{payload.title}</h4>
      )}
      <div className="flex items-center gap-3 rounded-lg border bg-muted/30 p-3">
        <FileText className="h-8 w-8 text-blue-500" />
        <div className="flex-1">
          <p className="text-sm font-medium">{payload.file_name || 'Document'}</p>
          <p className="text-xs text-muted-foreground">Word Document</p>
        </div>
        <a
          href={payload.url}
          download={payload.file_name}
          className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
        >
          <Download className="h-4 w-4" />
          Download
        </a>
      </div>
      <div className="aspect-[3/4] max-h-[600px] w-full overflow-hidden rounded-lg border shadow-sm">
        <iframe
          src={viewerUrl}
          className="h-full w-full"
          title={payload.file_name || 'Document'}
        />
      </div>
    </div>
  );
}

