// Special Role types

export interface SpecialRole {
  id: string;
  nama: string;
  description?: string;
  color: string;
  capabilities: string[];
  is_active: boolean;
  user_count?: number;
  created_at: string;
}

export interface SpecialRoleUser {
  id: string;
  username: string;
  nama: string;
  avatar_url?: string;
  kelas_nama?: string;
  assigned_at: string;
  assigned_by?: string;
}

export interface SpecialRoleDetail extends SpecialRole {
  users?: SpecialRoleUser[];
}

export interface CapabilityInfo {
  key: string;
  label: string;
  group: string;
}

export interface CreateSpecialRoleRequest {
  nama: string;
  description?: string;
  color: string;
  capabilities: string[];
  is_active?: boolean;
}

export interface UpdateSpecialRoleRequest {
  nama?: string;
  description?: string;
  color?: string;
  capabilities?: string[];
  is_active?: boolean;
}

export interface AssignUsersRequest {
  user_ids: string[];
}

export interface UserSpecialRolesRequest {
  special_role_ids: string[];
}

// Capability groups for UI
export const capabilityGroups = [
  {
    title: 'Overview',
    capabilities: ['dashboard'],
  },
  {
    title: 'Konten',
    capabilities: ['portfolios', 'moderation', 'assessments', 'assessment_metrics', 'tags', 'series'],
  },
  {
    title: 'Pengguna',
    capabilities: ['users', 'special_roles'],
  },
  {
    title: 'Akademik',
    capabilities: ['majors', 'classes', 'academic_years'],
  },
  {
    title: 'Lainnya',
    capabilities: ['feedback'],
  },
];

// Preset colors for color picker
export const presetColors = [
  { name: 'Indigo', value: '#6366f1' },
  { name: 'Blue', value: '#3b82f6' },
  { name: 'Cyan', value: '#06b6d4' },
  { name: 'Teal', value: '#14b8a6' },
  { name: 'Green', value: '#22c55e' },
  { name: 'Lime', value: '#84cc16' },
  { name: 'Yellow', value: '#eab308' },
  { name: 'Orange', value: '#f97316' },
  { name: 'Red', value: '#ef4444' },
  { name: 'Pink', value: '#ec4899' },
  { name: 'Purple', value: '#a855f7' },
  { name: 'Violet', value: '#8b5cf6' },
];

// Utility function to generate lighter background color
export function generateBgColor(baseColor: string, opacity: number = 0.15): string {
  const r = parseInt(baseColor.slice(1, 3), 16);
  const g = parseInt(baseColor.slice(3, 5), 16);
  const b = parseInt(baseColor.slice(5, 7), 16);
  return `rgba(${r}, ${g}, ${b}, ${opacity})`;
}
