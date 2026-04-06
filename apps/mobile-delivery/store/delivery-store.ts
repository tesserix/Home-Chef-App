import { create } from 'zustand';

interface DeliveryTrackingState {
  isTrackingLocation: boolean;
  activeDeliveryId: string | null;
  setTrackingLocation: (
    isTracking: boolean,
    deliveryId?: string | null,
  ) => void;
}

export const useDeliveryStore = create<DeliveryTrackingState>((set) => ({
  isTrackingLocation: false,
  activeDeliveryId: null,

  setTrackingLocation: (
    isTracking: boolean,
    deliveryId: string | null = null,
  ) => {
    set({
      isTrackingLocation: isTracking,
      activeDeliveryId: isTracking ? deliveryId : null,
    });
  },
}));
