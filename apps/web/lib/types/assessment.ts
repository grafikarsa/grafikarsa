// Assessment types

export interface AssessmentMetric {
  id: string;
  nama: string;
  deskripsi?: string;
  urutan: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateMetricRequest {
  nama: string;
  deskripsi?: string;
}

export interface UpdateMetricRequest {
  nama?: string;
  deskripsi?: string;
  is_active?: boolean;
}

export interface ReorderMetricsRequest {
  orders: { id: string; urutan: number }[];
}

export interface ScoreInput {
  metric_id: string;
  score: number;
  comment?: string;
}

export interface CreateAssessmentRequest {
  scores: ScoreInput[];
  final_comment?: string;
}

export interface ScoreResponse {
  id: string;
  metric_id: string;
  metric?: AssessmentMetric;
  score: number;
  comment?: string;
  created_at: string;
  updated_at: string;
}

export interface AssessmentResponse {
  id: string;
  portfolio_id: string;
  portfolio?: {
    id: string;
    judul: string;
    slug: string;
    thumbnail_url?: string;
  };
  assessed_by: string;
  assessor?: {
    id: string;
    username: string;
    nama: string;
    avatar_url?: string;
  };
  scores: ScoreResponse[];
  final_comment?: string;
  total_score?: number;
  created_at: string;
  updated_at: string;
}

export interface PortfolioForAssessment {
  id: string;
  judul: string;
  slug: string;
  thumbnail_url?: string;
  published_at?: string;
  user?: {
    id: string;
    username: string;
    nama: string;
    avatar_url?: string;
  };
  assessment?: {
    id: string;
    total_score?: number;
    assessor?: {
      id: string;
      username: string;
      nama: string;
      avatar_url?: string;
    };
    assessed_at: string;
  };
}


