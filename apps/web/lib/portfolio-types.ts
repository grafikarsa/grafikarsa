// Tipe data karya yang dipakai bersama antara komponen client (detail) dan
// bagian server (metadata + JSON-LD).
//
// Dipisah supaya halaman tidak mendefinisikan tipe yang sama dua kali —
// ketiganya harus tetap sinkron.

export type Block = {
  id: string;
  block_type: string;
  position: number;
  payload: Record<string, unknown>;
};

export type Portfolio = {
  id: string;
  title: string;
  slug: string;
  status: string;
  blocks: Block[];
  like_count: number;
  comment_count: number;
  is_liked?: boolean;
  author_name?: string;
  author_username?: string;
  description?: string;
  published_at?: string | null;
  updated_at?: string | null;
  thumbnail_media_id?: string | null;
};

export type PortfolioDetail = Portfolio;
