'use client';

import {
  Document,
  Page,
  Text,
  View,
  Image,
  StyleSheet,
} from '@react-pdf/renderer';
import { SeriesExportResponse, PortfolioExportItem } from '@/lib/api/admin';

// Condensed styles for 1-page layout
const styles = StyleSheet.create({
  page: {
    padding: 25,
    fontFamily: 'Helvetica',
    fontSize: 8,
    color: 'black',
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 10,
    paddingBottom: 8,
    borderBottomWidth: 1,
    borderBottomColor: 'lightgray',
  },
  headerLeft: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  logo: {
    width: 24,
    height: 24,
    marginRight: 6,
  },
  brandText: {
    fontSize: 12,
    fontFamily: 'Helvetica-Bold',
    color: 'black',
  },
  brandSubtext: {
    fontSize: 7,
    color: 'gray',
  },
  qrCode: {
    width: 40,
    height: 40,
  },
  seriesTitle: {
    fontSize: 14,
    fontFamily: 'Helvetica-Bold',
    color: 'black',
    marginBottom: 8,
    textAlign: 'center',
  },
  mainContent: {
    flexDirection: 'row',
    gap: 10,
  },
  leftColumn: {
    width: '35%',
  },
  rightColumn: {
    width: '65%',
  },
  profileSection: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 8,
  },
  avatar: {
    width: 40,
    height: 40,
    borderRadius: 20,
    marginRight: 8,
  },
  avatarPlaceholder: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'lightgray',
    marginRight: 8,
    justifyContent: 'center',
    alignItems: 'center',
  },
  avatarInitial: {
    fontSize: 16,
    color: 'gray',
  },
  profileInfo: {
    flex: 1,
  },
  userName: {
    fontSize: 11,
    fontFamily: 'Helvetica-Bold',
  },
  userUsername: {
    fontSize: 8,
    color: 'gray',
  },
  infoBox: {
    backgroundColor: '#f5f5f5',
    padding: 6,
    marginBottom: 6,
  },
  infoTitle: {
    fontSize: 7,
    fontFamily: 'Helvetica-Bold',
    color: 'gray',
    marginBottom: 4,
  },
  infoRow: {
    flexDirection: 'row',
    marginBottom: 2,
  },
  infoLabel: {
    width: 45,
    fontSize: 7,
    color: 'gray',
  },
  infoValue: {
    flex: 1,
    fontSize: 7,
  },
  portfolioTitle: {
    fontSize: 10,
    fontFamily: 'Helvetica-Bold',
    marginBottom: 4,
  },
  portfolioDate: {
    fontSize: 7,
    color: 'gray',
    marginBottom: 6,
  },
  thumbnail: {
    width: '100%',
    height: 80,
    marginBottom: 6,
  },
  thumbnailPlaceholder: {
    width: '100%',
    height: 80,
    backgroundColor: 'lightgray',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 6,
  },
  contentTitle: {
    fontSize: 8,
    fontFamily: 'Helvetica-Bold',
    color: 'gray',
    marginBottom: 4,
  },
  blockCard: {
    marginBottom: 4,
    borderWidth: 1,
    borderColor: 'lightgray',
  },
  blockHeader: {
    backgroundColor: '#fff3cd',
    padding: 4,
  },
  blockInstruksi: {
    fontSize: 7,
    color: 'brown',
  },
  blockContent: {
    padding: 4,
  },
  textContent: {
    fontSize: 7,
    lineHeight: 1.3,
  },
  imageBlock: {
    height: 50,
    backgroundColor: 'lightgray',
    justifyContent: 'center',
    alignItems: 'center',
  },
  imageCaption: {
    fontSize: 6,
    color: 'gray',
    marginTop: 2,
    textAlign: 'center',
  },
  youtubeText: {
    fontSize: 7,
  },
  youtubeUrl: {
    fontSize: 6,
    color: 'gray',
  },
  buttonText: {
    fontSize: 7,
    fontFamily: 'Helvetica-Bold',
  },
  buttonUrl: {
    fontSize: 6,
    color: 'gray',
  },
  table: {
    borderWidth: 1,
    borderColor: 'lightgray',
  },
  tableRow: {
    flexDirection: 'row',
    borderBottomWidth: 1,
    borderBottomColor: 'lightgray',
  },
  tableRowLast: {
    flexDirection: 'row',
  },
  tableRowHeader: {
    backgroundColor: '#f0f0f0',
  },
  tableCell: {
    flex: 1,
    padding: 3,
    fontSize: 6,
    borderRightWidth: 1,
    borderRightColor: 'lightgray',
  },
  tableCellLast: {
    flex: 1,
    padding: 3,
    fontSize: 6,
  },
  tableCellHeader: {
    fontFamily: 'Helvetica-Bold',
  },
  footer: {
    position: 'absolute',
    bottom: 15,
    left: 25,
    right: 25,
    textAlign: 'center',
    fontSize: 7,
    color: 'gray',
  },
});

interface PdfDocumentProps {
  data: SeriesExportResponse;
  qrCodes: Map<string, string>;
  imageCache: Map<string, string>; // url -> base64
  logoBase64?: string;
}

export function PdfDocument({ data, qrCodes, imageCache, logoBase64 }: PdfDocumentProps) {
  const { series, portfolios } = data;

  return (
    <Document>
      {portfolios.map((portfolio, index) => (
        <PortfolioPage
          key={portfolio.id}
          portfolio={portfolio}
          series={series}
          qrCode={qrCodes.get(portfolio.user.username)}
          imageCache={imageCache}
          logoBase64={logoBase64}
          pageNumber={index + 1}
          totalPages={portfolios.length}
        />
      ))}
    </Document>
  );
}

interface PortfolioPageProps {
  portfolio: PortfolioExportItem;
  series: SeriesExportResponse['series'];
  qrCode?: string;
  imageCache: Map<string, string>;
  logoBase64?: string;
  pageNumber: number;
  totalPages: number;
}

function PortfolioPage({ portfolio, series, qrCode, imageCache, logoBase64, pageNumber, totalPages }: PortfolioPageProps) {
  const { user } = portfolio;
  const profileUrl = `grafikarsa.com/${user.username}`;

  const getInstruksi = (blockOrder: number): string | undefined => {
    const seriesBlock = series.blocks?.find((b) => b.block_order === blockOrder);
    return seriesBlock?.instruksi;
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' });
  };

  const avatarBase64 = user.avatar_url ? imageCache.get(user.avatar_url) : undefined;
  const thumbnailBase64 = portfolio.thumbnail_url ? imageCache.get(portfolio.thumbnail_url) : undefined;

  return (
    <Page size="A4" style={styles.page}>
      {/* Header */}
      <View style={styles.header}>
        <View style={styles.headerLeft}>
          {logoBase64 && <Image src={logoBase64} style={styles.logo} />}
          <View>
            <Text style={styles.brandText}>Grafikarsa</Text>
            <Text style={styles.brandSubtext}>Katalog Portofolio Digital</Text>
          </View>
        </View>
        {qrCode && <Image src={qrCode} style={styles.qrCode} />}
      </View>

      {/* Series Title as Main Heading */}
      <Text style={styles.seriesTitle}>{series.nama}</Text>

      {/* Two Column Layout */}
      <View style={styles.mainContent}>
        {/* Left Column - Profile & Info */}
        <View style={styles.leftColumn}>
          {/* Profile */}
          <View style={styles.profileSection}>
            {avatarBase64 ? (
              <Image src={avatarBase64} style={styles.avatar} />
            ) : (
              <View style={styles.avatarPlaceholder}>
                <Text style={styles.avatarInitial}>{user.nama.charAt(0).toUpperCase()}</Text>
              </View>
            )}
            <View style={styles.profileInfo}>
              <Text style={styles.userName}>{user.nama}</Text>
              <Text style={styles.userUsername}>@{user.username}</Text>
            </View>
          </View>

          {/* Student Data */}
          <View style={styles.infoBox}>
            <Text style={styles.infoTitle}>DATA SISWA</Text>
            {user.kelas_nama && (
              <View style={styles.infoRow}>
                <Text style={styles.infoLabel}>Kelas</Text>
                <Text style={styles.infoValue}>{user.kelas_nama}</Text>
              </View>
            )}
            {user.jurusan_nama && (
              <View style={styles.infoRow}>
                <Text style={styles.infoLabel}>Jurusan</Text>
                <Text style={styles.infoValue}>{user.jurusan_nama}</Text>
              </View>
            )}
            {user.nisn && (
              <View style={styles.infoRow}>
                <Text style={styles.infoLabel}>NISN</Text>
                <Text style={styles.infoValue}>{user.nisn}</Text>
              </View>
            )}
            {user.nis && (
              <View style={styles.infoRow}>
                <Text style={styles.infoLabel}>NIS</Text>
                <Text style={styles.infoValue}>{user.nis}</Text>
              </View>
            )}
            <View style={styles.infoRow}>
              <Text style={styles.infoLabel}>Profil</Text>
              <Text style={styles.infoValue}>{profileUrl}</Text>
            </View>
          </View>

          {/* Portfolio Info */}
          <View style={styles.infoBox}>
            <Text style={styles.portfolioTitle}>{portfolio.judul}</Text>
            <Text style={styles.portfolioDate}>{formatDate(portfolio.created_at)}</Text>
          </View>

          {/* Thumbnail */}
          {thumbnailBase64 ? (
            <Image src={thumbnailBase64} style={styles.thumbnail} />
          ) : portfolio.thumbnail_url ? (
            <View style={styles.thumbnailPlaceholder}>
              <Text style={{ fontSize: 7, color: 'gray' }}>Thumbnail</Text>
            </View>
          ) : null}
        </View>

        {/* Right Column - Content Blocks */}
        <View style={styles.rightColumn}>
          <Text style={styles.contentTitle}>KONTEN PORTOFOLIO</Text>
          {portfolio.content_blocks.map((block) => (
            <ContentBlock
              key={block.id}
              block={block}
              instruksi={getInstruksi(block.block_order)}
              imageCache={imageCache}
            />
          ))}
        </View>
      </View>

      {/* Footer */}
      <Text style={styles.footer}>Halaman {pageNumber} dari {totalPages}</Text>
    </Page>
  );
}

interface ContentBlockProps {
  block: PortfolioExportItem['content_blocks'][0];
  instruksi?: string;
  imageCache: Map<string, string>;
}

function ContentBlock({ block, instruksi, imageCache }: ContentBlockProps) {
  const payload = block.payload as Record<string, unknown>;

  const renderContent = () => {
    switch (block.block_type) {
      case 'text':
        const content = String(payload.content || '');
        // Truncate long text
        const truncated = content.length > 200 ? content.substring(0, 200) + '...' : content;
        return (
          <View style={styles.blockContent}>
            <Text style={styles.textContent}>{truncated}</Text>
          </View>
        );

      case 'image':
        const imageUrl = payload.url ? String(payload.url) : '';
        const imageCaption = payload.caption ? String(payload.caption) : '';
        const imageBase64 = imageUrl ? imageCache.get(imageUrl) : undefined;
        return (
          <View style={styles.blockContent}>
            {imageBase64 ? (
              <Image src={imageBase64} style={{ width: '100%', height: 60 }} />
            ) : (
              <View style={styles.imageBlock}>
                <Text style={{ fontSize: 7, color: 'gray' }}>Gambar</Text>
              </View>
            )}
            {imageCaption && <Text style={styles.imageCaption}>{imageCaption}</Text>}
          </View>
        );

      case 'youtube':
        const videoId = String(payload.video_id || '');
        const videoTitle = String(payload.title || 'Video YouTube');
        return (
          <View style={styles.blockContent}>
            <Text style={styles.youtubeText}>{videoTitle}</Text>
            <Text style={styles.youtubeUrl}>youtube.com/watch?v={videoId}</Text>
          </View>
        );

      case 'button':
        const buttonText = String(payload.text || 'Link');
        const buttonUrl = String(payload.url || '');
        return (
          <View style={styles.blockContent}>
            <Text style={styles.buttonText}>{buttonText}</Text>
            <Text style={styles.buttonUrl}>{buttonUrl}</Text>
          </View>
        );

      case 'table':
        const headers = (payload.headers as string[]) || [];
        const rows = (payload.rows as string[][]) || [];
        return (
          <View style={styles.blockContent}>
            <View style={styles.table}>
              {headers.length > 0 && (
                <View style={[styles.tableRow, styles.tableRowHeader]}>
                  {headers.map((header, i) => (
                    <Text key={i} style={[i === headers.length - 1 ? styles.tableCellLast : styles.tableCell, styles.tableCellHeader]}>
                      {header}
                    </Text>
                  ))}
                </View>
              )}
              {rows.slice(0, 3).map((row, rowIdx) => (
                <View key={rowIdx} style={rowIdx === Math.min(rows.length, 3) - 1 ? styles.tableRowLast : styles.tableRow}>
                  {row.map((cell, cellIdx) => (
                    <Text key={cellIdx} style={cellIdx === row.length - 1 ? styles.tableCellLast : styles.tableCell}>
                      {cell}
                    </Text>
                  ))}
                </View>
              ))}
            </View>
          </View>
        );

      default:
        return null;
    }
  };

  return (
    <View style={styles.blockCard}>
      {instruksi && (
        <View style={styles.blockHeader}>
          <Text style={styles.blockInstruksi}>{instruksi}</Text>
        </View>
      )}
      {renderContent()}
    </View>
  );
}
