// Scheduled delivery slots (#51) — fetches a chef's offered lunch/dinner
// delivery windows for the checkout picker. Mirrors the API's
// GET /chefs/:id/delivery-slots response (services.SlotAvailability).

import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

export interface DeliverySlot {
  date: string; // "YYYY-MM-DD" IST
  slot: 'lunch' | 'dinner';
  label: string; // "Lunch" | "Dinner"
  window: string; // "12:00–14:00"
  start: string;
  end: string;
  cutoff?: string;
  remaining: number | null; // null = unlimited
  available: boolean; // open (not past cutoff/window) and not full
  scheduledFor: string; // ISO timestamp
}

export interface DeliverySlotsResponse {
  slotsEnabled: boolean;
  slots: DeliverySlot[];
}

export function useDeliverySlots(chefId?: string) {
  return useQuery<DeliverySlotsResponse>({
    queryKey: ['delivery-slots', chefId],
    queryFn: async () => {
      const r = await api.get(`/v1/chefs/${chefId}/delivery-slots`);
      return r.data as DeliverySlotsResponse;
    },
    enabled: !!chefId,
    staleTime: 60_000,
  });
}
