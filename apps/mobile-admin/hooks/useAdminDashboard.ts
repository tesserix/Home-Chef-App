import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { AdminStats, Activity, AdminAnalytics } from '../lib/admin-types';

export function useAdminStats() {
  return useQuery<AdminStats>({
    queryKey: ['admin', 'stats'],
    queryFn: () => api.get<AdminStats>('/admin/stats').then((r) => r.data),
    refetchInterval: 60_000,
  });
}

export function useAdminActivities(limit = 20) {
  return useQuery<Activity[]>({
    queryKey: ['admin', 'activities', limit],
    queryFn: () =>
      api
        .get<{ data: Activity[] }>('/admin/activities', { params: { limit } })
        .then((r) => r.data.data ?? []),
  });
}

export function useAdminAnalytics() {
  return useQuery<AdminAnalytics>({
    queryKey: ['admin', 'analytics'],
    queryFn: () => api.get<AdminAnalytics>('/admin/analytics').then((r) => r.data),
  });
}
