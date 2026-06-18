import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Chef capacity & cutoff controls (#48): per-meal cutoffs + auto-sold-out, and
// per-dish daily caps (the cap counts live on the menu item via useVendorMenu).

export interface CapacitySettings {
  chefId: string;
  cutoffEnabled: boolean;
  lunchCutoff: string; // "HH:MM" IST, "" = none
  dinnerCutoff: string;
  autoSoldOut: boolean;
}

export function useCapacitySettings() {
  return useQuery<CapacitySettings>({
    queryKey: ['chef', 'capacity-settings'],
    queryFn: () =>
      api.get<CapacitySettings>('/chef/capacity-settings').then((r) => r.data),
    staleTime: 60_000,
  });
}

export function useUpdateCapacitySettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Partial<Omit<CapacitySettings, 'chefId'>>) =>
      api.put<CapacitySettings>('/chef/capacity-settings', body).then((r) => r.data),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ['chef', 'capacity-settings'] }),
  });
}

export function useSetItemCapacity() {
  const qc = useQueryClient();
  return useMutation({
    // dailyCapacity null = unlimited.
    mutationFn: (vars: { itemId: string; dailyCapacity: number | null }) =>
      api
        .put(`/chef/menu/items/${vars.itemId}/capacity`, {
          dailyCapacity: vars.dailyCapacity,
        })
        .then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chef', 'menu'] }),
  });
}
