'use client';

import React from 'react';
import { Document, Page, Text, View, Image, StyleSheet } from '@react-pdf/renderer';
import type { SeriesExportResponse, PortfolioExportItem } from '@/lib/api/admin';

// ── Color Palette (Full Black & White) ──────────────────────────────────────
const C = {
  black: '#000000',
  dark: '#333333',
  mid: '#666666',
  muted: '#888888',
  border: '#cccccc',
  bg: '#f0f0f0',
  white: '#ffffff',
} as const;

// ── Page Constants ──────────────────────────────────────────────────────────
const PAGE_W = 595.28;
const MARGIN = 40;
const CONTENT_W = PAGE_W - MARGIN * 2;
const COL_GAP = 10;
const COL_W = (CONTENT_W - COL_GAP) / 2;

// ── Styles ──────────────────────────────────────────────────────────────────
const s = StyleSheet.create({
  page: {
    fontFamily: 'Helvetica',
    fontSize: 9,
    color: C.white,
  },
  pageContent: {
    padding: MARGIN,
    flex: 1,
  },
  portfolioPageContent: {
    padding: MARGIN,
    paddingTop: 0,
    paddingBottom: MARGIN + 30,
    flex: 1,
  },

  // ── Cover Page ────────────────────────────────────────────────────────────
  cover: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  coverBrand: {
    fontSize: 28,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
    letterSpacing: 2,
    marginBottom: 4,
  },
  coverBrandSub: {
    fontSize: 10,
    fontFamily: 'Helvetica',
    color: C.white,
    marginBottom: 40,
  },
  coverDivider: {
    width: 120,
    height: 1,
    backgroundColor: C.border,
    marginBottom: 30,
  },
  coverTitle: {
    fontSize: 24,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
    textAlign: 'center',
    marginBottom: 8,
  },
  coverSubtitle: {
    fontSize: 12,
    fontFamily: 'Helvetica',
    color: C.white,
    textAlign: 'center',
    marginBottom: 40,
  },
  coverFilters: {
    alignItems: 'center',
    marginBottom: 30,
  },
  coverFilterText: {
    fontSize: 8,
    color: C.white,
    marginBottom: 4,
    textAlign: 'center',
  },
  coverDate: {
    fontSize: 10,
    color: C.white,
  },

  // ── Portfolio Page ────────────────────────────────────────────────────────
  portHeader: {
    alignItems: 'center',
    marginTop: 16,
    marginBottom: 12,
    paddingTop: 10,
    paddingBottom: 8,
    paddingLeft: MARGIN,
    paddingRight: MARGIN,
    marginLeft: MARGIN,
    marginRight: MARGIN,
    borderBottomWidth: 1,
    borderBottomColor: C.border,
  },
  portHeaderBrand: {
    fontSize: 10,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
    letterSpacing: 1,
    textAlign: 'center',
  },

  // Portfolio title box
  portTitleBox: {
    marginBottom: 16,
  },
  portTitleLabel: {
    fontSize: 7,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
    marginBottom: 4,
    textTransform: 'uppercase',
  },
  portTitle: {
    fontSize: 14,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
  },
  portDate: {
    fontSize: 8,
    color: C.white,
    marginTop: 3,
  },

  // Profile section
  profileSection: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    marginBottom: 12,
    gap: 10,
  },
  avatar: {
    width: 40,
    height: 40,
  },
  avatarPlaceholder: {
    width: 40,
    height: 40,
    backgroundColor: C.bg,
    justifyContent: 'center',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: C.border,
  },
  avatarInitial: {
    fontSize: 16,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
  },
  profileInfo: {
    flex: 1,
  },
  profileRow: {
    flexDirection: 'row',
    marginBottom: 2,
  },
  profileLabel: {
    width: 80,
    fontSize: 7,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
    textTransform: 'uppercase',
  },
  profileValue: {
    flex: 1,
    fontSize: 8,
    color: C.white,
  },
  profileValueLink: {
    flex: 1,
    fontSize: 8,
    color: C.white,
    textDecoration: 'underline',
  },
  profileQr: {
    width: 40,
    height: 40,
  },

  // Thumbnail
  thumbnailContainer: {
    width: 140,
  },
  thumbnail: {
    width: 140,
    height: 100,
    objectFit: 'contain',
    backgroundColor: C.bg,
    borderWidth: 1,
    borderColor: C.border,
  },
  thumbnailPlaceholder: {
    width: 140,
    height: 100,
    backgroundColor: C.bg,
    justifyContent: 'center',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: C.border,
  },

  // Content section header
  contentHeader: {
    fontSize: 9,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
    marginBottom: 8,
    textTransform: 'uppercase',
    letterSpacing: 1,
  },
  contentDivider: {
    width: '100%',
    height: 1,
    backgroundColor: C.black,
    marginBottom: 12,
  },

  // Block card
  blockCard: {
    marginBottom: 4,
    borderWidth: 1,
    borderColor: C.border,
    padding: 4,
  },
  blockInstruction: {
    backgroundColor: C.bg,
    padding: 4,
    marginBottom: 4,
    borderBottomWidth: 1,
    borderBottomColor: C.border,
  },
  blockInstructionText: {
    fontSize: 7,
    fontFamily: 'Helvetica-Bold',
    color: C.black,
  },

  // Text block
  textContent: {
    fontSize: 6.5,
    color: C.white,
    lineHeight: 1.4,
  },

  // Image block
  imageBlock: {
    marginBottom: 3,
  },
  imageBlockImg: {
    maxWidth: CONTENT_W - 18,
    maxHeight: 140,
    objectFit: 'contain',
    backgroundColor: C.bg,
  },
  imageBlockPlaceholder: {
    height: 60,
    backgroundColor: C.bg,
    justifyContent: 'center',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: C.border,
  },
  imageCaption: {
    fontSize: 6,
    color: C.white,
    marginTop: 3,
    textAlign: 'center',
  },

  // Table block
  table: {
    borderWidth: 1,
    borderColor: C.border,
  },
  tableRow: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: C.border,
  },
  tableRowLast: {
    flexDirection: 'row',
  },
  tableHeader: {
    backgroundColor: C.black,
  },
  tableCell: {
    flex: 1,
    padding: 5,
    fontSize: 7,
    color: C.white,
    borderRightWidth: 1,
    borderRightColor: C.border,
  },
  tableCellLast: {
    flex: 1,
    padding: 5,
    fontSize: 7,
    color: C.white,
  },
  tableCellHeader: {
    flex: 1,
    padding: 5,
    fontSize: 7,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
    borderRightWidth: 1,
    borderRightColor: C.mid,
  },
  tableCellHeaderLast: {
    flex: 1,
    padding: 5,
    fontSize: 7,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
  },
  tableMore: {
    fontSize: 7,
    color: C.white,
    marginTop: 4,
    fontStyle: 'italic',
  },

  // YouTube / Button / Link blocks
  linkTitle: {
    fontSize: 7,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
    marginBottom: 2,
  },
  linkUrl: {
    fontSize: 6,
    color: C.white,
    textDecoration: 'underline',
  },
  linkBadge: {
    fontSize: 6,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
    marginBottom: 2,
  },

  // Embed block
  embedContainer: {
    backgroundColor: C.bg,
    padding: 6,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: C.border,
  },
  embedIcon: {
    fontSize: 12,
    color: C.white,
    marginBottom: 4,
  },
  embedTitle: {
    fontSize: 7,
    fontFamily: 'Helvetica-Bold',
    color: C.white,
    marginBottom: 2,
  },
  embedUrl: {
    fontSize: 6,
    color: C.white,
  },

  // Two-column layout
  twoColRow: {
    flexDirection: 'row',
    gap: COL_GAP,
    marginBottom: 6,
  },
  twoColCell: {
    flex: 1,
  },

  // Footer
  footer: {
    position: 'absolute',
    bottom: 25,
    left: MARGIN,
    right: MARGIN,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    borderTopWidth: 1,
    borderTopColor: C.border,
    paddingTop: 8,
  },
  footerText: {
    fontSize: 7,
    color: C.white,
  },
});

// ── Helpers ─────────────────────────────────────────────────────────────────

function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    return d.toLocaleDateString('id-ID', {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
    });
  } catch {
    return dateStr;
  }
}

function truncate(text: string, max: number): string {
  if (text.length <= max) return text;
  return text.substring(0, max).trimEnd() + '...';
}

function getInitial(name: string): string {
  return name.charAt(0).toUpperCase();
}

// ── Document Props ──────────────────────────────────────────────────────────
interface PdfDocumentProps {
  data: SeriesExportResponse;
  qrCodes: Map<string, string>;
  imageCache: Map<string, string>;
  jurusanLabel: string;
  kelasLabel: string;
  bgImage?: string;
}

// ── Background Image ───────────────────────────────────────────────────────
const PAGE_H = 841.89;

function PageBackground({ src }: { src?: string }) {
  if (!src) return null;
  return (
    <Image
      src={src}
      fixed
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: PAGE_W,
        height: PAGE_H,
      }}
    />
  );
}

// ── Main Document ───────────────────────────────────────────────────────────
export function PdfDocument({ data, qrCodes, imageCache, jurusanLabel, kelasLabel, bgImage }: PdfDocumentProps) {
  const { series, portfolios } = data;

  return (
    <Document>
      <CoverPage
        seriesName={series.nama}
        portfolioCount={portfolios.length}
        jurusanLabel={jurusanLabel}
        kelasLabel={kelasLabel}
        bgImage={bgImage}
      />

      {portfolios.map((portfolio) => (
        <PortfolioPage
          key={portfolio.id}
          portfolio={portfolio}
          series={series}
          qrCode={qrCodes.get(portfolio.user.username)}
          imageCache={imageCache}
          bgImage={bgImage}
        />
      ))}
    </Document>
  );
}

// ── Cover Page ──────────────────────────────────────────────────────────────
interface CoverPageProps {
  seriesName: string;
  portfolioCount: number;
  jurusanLabel: string;
  kelasLabel: string;
  bgImage?: string;
}

function CoverPage({ seriesName, portfolioCount, jurusanLabel, kelasLabel, bgImage }: CoverPageProps) {
  const today = formatDate(new Date().toISOString());

  return (
    <Page size="A4" style={s.page}>
      <PageBackground src={bgImage} />
      <View style={s.pageContent}>
        <View style={s.cover}>
          <Text style={s.coverBrand}>grafikarsa.com</Text>
          <Text style={s.coverBrandSub}>Katalog Portofolio Digital SMKN 4 Malang</Text>

          <View style={s.coverDivider} />

          <Text style={s.coverTitle}>{seriesName}</Text>
          <Text style={s.coverSubtitle}>{portfolioCount} Portofolio</Text>

          <View style={s.coverDivider} />

          <View style={s.coverFilters}>
            <Text style={s.coverFilterText}>Jurusan: {jurusanLabel}</Text>
            <Text style={s.coverFilterText}>Kelas: {kelasLabel}</Text>
          </View>

          <Text style={s.coverDate}>{today}</Text>
        </View>
      </View>
    </Page>
  );
}

// ── Portfolio Page ──────────────────────────────────────────────────────────
interface PortfolioPageProps {
  portfolio: PortfolioExportItem;
  series: SeriesExportResponse['series'];
  qrCode?: string;
  imageCache: Map<string, string>;
  bgImage?: string;
}

function PortfolioPage({ portfolio, series, qrCode, imageCache, bgImage }: PortfolioPageProps) {
  const { user } = portfolio;
  const profileUrl = `grafikarsa.com/${user.username}`;

  const avatarBase64 = user.avatar_url ? imageCache.get(user.avatar_url) : undefined;
  const thumbnailBase64 = portfolio.thumbnail_url
    ? imageCache.get(portfolio.thumbnail_url)
    : undefined;

  return (
    <Page size="A4" style={s.page} wrap>
      <PageBackground src={bgImage} />
      <View style={s.portHeader}>
        <Text style={s.portHeaderBrand}>grafikarsa.com</Text>
      </View>
      <View style={s.portfolioPageContent}>

        {/* Portfolio title */}
        <View style={s.portTitleBox}>
          <Text style={s.portTitleLabel}>Judul Portofolio</Text>
          <Text style={s.portTitle}>{portfolio.judul}</Text>
          <Text style={s.portDate}>Dibuat: {formatDate(portfolio.created_at)}</Text>
        </View>

        {/* Profile section: avatar + info + thumbnail right */}
        <View style={s.profileSection}>
          {avatarBase64 ? (
            <Image src={avatarBase64} style={s.avatar} />
          ) : (
            <View style={s.avatarPlaceholder}>
              <Text style={s.avatarInitial}>{getInitial(user.nama)}</Text>
            </View>
          )}
          <View style={s.profileInfo}>
            <ProfileRow label="Nama" value={user.nama} />
            <ProfileRow label="Username" value={`@${user.username}`} />
            {user.kelas_nama && <ProfileRow label="Kelas" value={user.kelas_nama} />}
            {user.jurusan_nama && <ProfileRow label="Jurusan" value={user.jurusan_nama} />}
            {user.nisn && <ProfileRow label="NISN" value={user.nisn} />}
            {user.nis && <ProfileRow label="NIS" value={user.nis} />}
            <ProfileRow label="Profil" value={profileUrl} isLink />
          </View>
          {/* Thumbnail on right side */}
          {thumbnailBase64 ? (
            <View style={s.thumbnailContainer}>
              <Image src={thumbnailBase64} style={s.thumbnail} />
            </View>
          ) : portfolio.thumbnail_url ? (
            <View style={s.thumbnailContainer}>
              <View style={s.thumbnailPlaceholder}>
                <Text style={{ fontSize: 6, color: C.white }}>Thumbnail tidak tersedia</Text>
              </View>
            </View>
          ) : null}
        </View>

        {/* Content blocks */}
        <Text style={s.contentHeader}>Konten Portofolio</Text>
        <View style={s.contentDivider} />

        {portfolio.content_blocks
          .sort((a, b) => a.block_order - b.block_order)
          .map((block) => (
            <ContentBlock
              key={block.id}
              block={block}
              imageCache={imageCache}
            />
          ))}

      </View>

      {/* Footer — page numbers added by pdf-lib post-processing */}
      <View style={s.footer} fixed>
        <Text style={s.footerText}>grafikarsa.com</Text>
      </View>
    </Page>
  );
}

// ── Profile Row ─────────────────────────────────────────────────────────────
function ProfileRow({
  label,
  value,
  isLink,
}: {
  label: string;
  value: string;
  isLink?: boolean;
}) {
  return (
    <View style={s.profileRow}>
      <Text style={s.profileLabel}>{label}</Text>
      <Text style={isLink ? s.profileValueLink : s.profileValue}>{value}</Text>
    </View>
  );
}

// ── Two-Column Blocks Layout ────────────────────────────────────────────────
const FULL_WIDTH_TYPES = new Set(['table']);

interface TwoColumnBlocksProps {
  blocks: PortfolioExportItem['content_blocks'];
  imageCache: Map<string, string>;
}

function TwoColumnBlocks({ blocks, imageCache }: TwoColumnBlocksProps) {
  const sorted = [...blocks].sort((a, b) => a.block_order - b.block_order);
  const rows: React.ReactNode[] = [];
  let pending: PortfolioExportItem['content_blocks'][0][] = [];

  const flushPair = () => {
    if (pending.length === 0) return;
    rows.push(
      <View key={`pair-${rows.length}`} style={s.twoColRow}>
        {pending.map((block) => (
          <View key={block.id} style={s.twoColCell}>
            <ContentBlock
              block={block}
              imageCache={imageCache}
            />
          </View>
        ))}
        {pending.length === 1 && <View style={s.twoColCell} />}
      </View>
    );
    pending = [];
  };

  for (const block of sorted) {
    if (FULL_WIDTH_TYPES.has(block.block_type)) {
      flushPair();
      rows.push(
        <View key={block.id} style={{ marginBottom: 4 }}>
          <ContentBlock
            block={block}
            imageCache={imageCache}
          />
        </View>
      );
    } else {
      pending.push(block);
      if (pending.length === 2) flushPair();
    }
  }
  flushPair();

  return <>{rows}</>;
}

// ── Content Block ───────────────────────────────────────────────────────────
interface ContentBlockProps {
  block: PortfolioExportItem['content_blocks'][0];
  imageCache: Map<string, string>;
}

function ContentBlock({ block, imageCache }: ContentBlockProps) {
  const payload = block.payload as Record<string, unknown>;

  return (
    <View style={s.blockCard}>
      <BlockContent blockType={block.block_type} payload={payload} imageCache={imageCache} />
    </View>
  );
}

// ── Block Content Renderer ──────────────────────────────────────────────────
function BlockContent({
  blockType,
  payload,
  imageCache,
}: {
  blockType: string;
  payload: Record<string, unknown>;
  imageCache: Map<string, string>;
}) {
  switch (blockType) {
    case 'text':
      return <TextBlock payload={payload} />;
    case 'image':
      return <ImageBlock payload={payload} imageCache={imageCache} />;
    case 'youtube':
      return <YouTubeBlock payload={payload} />;
    case 'button':
      return <ButtonBlock payload={payload} />;
    case 'link':
      return <LinkBlock payload={payload} />;
    case 'table':
      return <TableBlock payload={payload} />;
    case 'figma':
    case 'canva':
    case 'ppt':
    case 'pdf':
    case 'doc':
    case 'embed':
      return <EmbedBlock blockType={blockType} payload={payload} />;
    default:
      return (
        <Text style={{ fontSize: 7, color: C.white, fontStyle: 'italic' }}>
          Tipe konten: {blockType}
        </Text>
      );
  }
}

// ── Text Block ──────────────────────────────────────────────────────────────
function TextBlock({ payload }: { payload: Record<string, unknown> }) {
  const content = String(payload.content || '');
  return <Text style={s.textContent}>{truncate(content, 300)}</Text>;
}

// ── Image Block ─────────────────────────────────────────────────────────────
function ImageBlock({
  payload,
  imageCache,
}: {
  payload: Record<string, unknown>;
  imageCache: Map<string, string>;
}) {
  const url = String(payload.url || '');
  const caption = String(payload.caption || '');
  const base64 = url ? imageCache.get(url) : undefined;

  return (
    <View style={s.imageBlock}>
      {base64 ? (
        <Image src={base64} style={s.imageBlockImg} />
      ) : (
        <View style={s.imageBlockPlaceholder}>
          <Text style={{ fontSize: 8, color: C.white }}>Gambar tidak tersedia</Text>
        </View>
      )}
      {caption && <Text style={s.imageCaption}>{caption}</Text>}
    </View>
  );
}

// ── YouTube Block ───────────────────────────────────────────────────────────
function YouTubeBlock({ payload }: { payload: Record<string, unknown> }) {
  const title = String(payload.title || 'Video YouTube');
  const videoId = String(payload.video_id || '');

  return (
    <View>
      <Text style={s.linkBadge}>YouTube</Text>
      <Text style={s.linkTitle}>{title}</Text>
      <Text style={s.linkUrl}>youtube.com/watch?v={videoId}</Text>
    </View>
  );
}

// ── Button Block ────────────────────────────────────────────────────────────
function ButtonBlock({ payload }: { payload: Record<string, unknown> }) {
  const text = String(payload.text || 'Link');
  const url = String(payload.url || '');

  return (
    <View>
      <Text style={s.linkBadge}>Tombol</Text>
      <Text style={s.linkTitle}>{text}</Text>
      <Text style={s.linkUrl}>{url}</Text>
    </View>
  );
}

// ── Link Block ──────────────────────────────────────────────────────────────
function LinkBlock({ payload }: { payload: Record<string, unknown> }) {
  const title = String(payload.title || String(payload.text || 'Link'));
  const url = String(payload.url || '');

  return (
    <View>
      <Text style={s.linkBadge}>Tautan</Text>
      <Text style={s.linkTitle}>{title}</Text>
      <Text style={s.linkUrl}>{url}</Text>
    </View>
  );
}

// ── Table Block ─────────────────────────────────────────────────────────────
function TableBlock({ payload }: { payload: Record<string, unknown> }) {
  const headers = (payload.headers as string[]) || [];
  const rows = (payload.rows as string[][]) || [];
  const maxRows = 5;
  const visibleRows = rows.slice(0, maxRows);
  const hasMore = rows.length > maxRows;

  return (
    <View>
      <View style={s.table}>
        {headers.length > 0 && (
          <View style={[s.tableRow, s.tableHeader]}>
            {headers.map((header, i) => (
              <Text
                key={i}
                style={i === headers.length - 1 ? s.tableCellHeaderLast : s.tableCellHeader}
              >
                {header}
              </Text>
            ))}
          </View>
        )}
        {visibleRows.map((row, rowIdx) => (
          <View
            key={rowIdx}
            style={rowIdx === visibleRows.length - 1 && !hasMore ? s.tableRowLast : s.tableRow}
          >
            {row.map((cell, cellIdx) => (
              <Text
                key={cellIdx}
                style={cellIdx === row.length - 1 ? s.tableCellLast : s.tableCell}
              >
                {truncate(String(cell), 80)}
              </Text>
            ))}
          </View>
        ))}
      </View>
      {hasMore && <Text style={s.tableMore}>... dan {rows.length - maxRows} baris lainnya</Text>}
    </View>
  );
}

// ── Embed Block ─────────────────────────────────────────────────────────────
function EmbedBlock({
  blockType,
  payload,
}: {
  blockType: string;
  payload: Record<string, unknown>;
}) {
  const title = String(
    payload.title || payload.name || blockType.charAt(0).toUpperCase() + blockType.slice(1)
  );
  const url = String(payload.url || payload.embed_url || '');

  const labels: Record<string, string> = {
    figma: 'Figma',
    canva: 'Canva',
    ppt: 'PowerPoint',
    pdf: 'PDF',
    doc: 'Document',
    embed: 'Embed',
  };

  return (
    <View>
      <Text style={s.linkBadge}>{labels[blockType] || blockType}</Text>
      <Text style={s.linkTitle}>{title}</Text>
      {url && <Text style={s.linkUrl}>{truncate(url, 60)}</Text>}
    </View>
  );
}
