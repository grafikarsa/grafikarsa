import api from './client';
import type {
  Changelog,
  ChangelogListItem,
  ChangelogUnreadCount,
  CreateChangelogRequest,
  UpdateChangelogRequest,
} from '../types/changelog';
import type { ApiResponse, PaginatedResponse } from '../types/api';

// ============================================================================
// PUBLIC API
// ============================================================================

export async function getChangelogs(page = 1, limit = 10) {
  return api.get<PaginatedResponse<Changelog[]>>(
    `/changelogs?page=${page}&limit=${limit}`
  );
}

export async function getChangelogById(id: string) {
  return api.get<ApiResponse<Changelog>>(`/changelogs/${id}`);
}

export async function getLatestChangelog() {
  return api.get<ApiResponse<Changelog>>('/changelogs/latest');
}

export async function getUnreadCount() {
  return api.get<ApiResponse<ChangelogUnreadCount>>('/changelogs/unread-count');
}

export async function markChangelogAsRead(id: string) {
  return api.post<ApiResponse<null>>(`/changelogs/${id}/mark-read`);
}

export async function markAllChangelogsAsRead() {
  return api.post<ApiResponse<null>>('/changelogs/mark-all-read');
}

// ============================================================================
// ADMIN API
// ============================================================================

export async function getAdminChangelogs(page = 1, limit = 20, search = '') {
  const params = new URLSearchParams({
    page: page.toString(),
    limit: limit.toString(),
  });
  if (search) params.append('search', search);
  
  return api.get<PaginatedResponse<ChangelogListItem[]>>(
    `/admin/changelogs?${params.toString()}`
  );
}

export async function getAdminChangelogById(id: string) {
  return api.get<ApiResponse<Changelog>>(`/admin/changelogs/${id}`);
}

export async function createChangelog(data: CreateChangelogRequest) {
  return api.post<ApiResponse<Changelog>>('/admin/changelogs', data);
}

export async function updateChangelog(id: string, data: UpdateChangelogRequest) {
  return api.patch<ApiResponse<Changelog>>(`/admin/changelogs/${id}`, data);
}

export async function deleteChangelog(id: string) {
  return api.delete<ApiResponse<null>>(`/admin/changelogs/${id}`);
}

export async function publishChangelog(id: string) {
  return api.post<ApiResponse<null>>(`/admin/changelogs/${id}/publish`);
}

export async function unpublishChangelog(id: string) {
  return api.post<ApiResponse<null>>(`/admin/changelogs/${id}/unpublish`);
}
