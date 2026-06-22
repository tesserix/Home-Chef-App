import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export type SettingsResponse = Record<string, unknown>;

export function useAdminSettings() {
  return useQuery<SettingsResponse>({
    queryKey: ['admin', 'settings'],
    queryFn: () => api.get<SettingsResponse>('/admin/settings').then((r) => r.data),
  });
}

export function useUpdateSetting() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, value }: { key: string; value: unknown }) =>
      api.put('/admin/settings', { key, value }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'settings'] }),
  });
}
