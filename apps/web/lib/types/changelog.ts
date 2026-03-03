// Changelog types

export type ChangelogCategory = 'added' | 'updated' | 'removed' | 'fixed';

export interface ChangelogBlock {
  id: string;
  block_type: string;
  payload: Record<string, unknown>;
}

export interface ChangelogSection {
  id: string;
  category: ChangelogCategory;
  blocks: ChangelogBlock[];
}

export interface ChangelogContributor {
  id: string;
  user: {
    id: string;
    username: string;
    nama: string;
    avatar_url?: string;
  };
  contribution: string;
}

export interface Changelog {
  id: string;
  version: string;
  title: string;
  description?: string;
  release_date: string;
  is_published: boolean;
  sections: ChangelogSection[];
  contributors: ChangelogContributor[];
  created_by?: {
    id: string;
    username: string;
    nama: string;
    avatar_url?: string;
  };
  created_at: string;
  updated_at: string;
}

export interface ChangelogListItem {
  id: string;
  version: string;
  title: string;
  description?: string;
  release_date: string;
  is_published: boolean;
  categories: string[];
  created_at: string;
}

// Request types
export interface ChangelogSectionRequest {
  category: ChangelogCategory;
  blocks: {
    block_type: string;
    payload: Record<string, unknown>;
  }[];
}

export interface ChangelogContributorRequest {
  user_id: string;
  contribution: string;
}

export interface CreateChangelogRequest {
  version: string;
  title: string;
  description?: string;
  release_date?: string;
  sections: ChangelogSectionRequest[];
  contributors: ChangelogContributorRequest[];
}

export interface UpdateChangelogRequest {
  version?: string;
  title?: string;
  description?: string;
  release_date?: string;
  sections?: ChangelogSectionRequest[];
  contributors?: ChangelogContributorRequest[];
}

// Response types
export interface ChangelogUnreadCount {
  count: number;
}
