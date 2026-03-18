import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { OnboardingData, KitchenAddress, OperatingHours, DocumentUpload } from '@/shared/types';
import { createTTLStorage } from '@/shared/hooks/useDraftForm';

const TOTAL_STEPS = 5;

const DEFAULT_HOURS: OperatingHours = {
  monday: { open: '09:00', close: '21:00' },
  tuesday: { open: '09:00', close: '21:00' },
  wednesday: { open: '09:00', close: '21:00' },
  thursday: { open: '09:00', close: '21:00' },
  friday: { open: '09:00', close: '21:00' },
  saturday: { open: '09:00', close: '21:00' },
  sunday: { open: '09:00', close: '21:00' },
};

interface ServerProfile {
  businessName?: string;
  description?: string;
  cuisines?: string[];
  specialties?: string[];
  profileImage?: string;
  prepTime?: string;
  serviceRadius?: number;
  minimumOrder?: number;
  deliveryFee?: number;
  addressLine1?: string;
  addressLine2?: string;
  city?: string;
  state?: string;
  postalCode?: string;
  country?: string;
  fullName?: string;
  email?: string;
  phone?: string;
}

interface OnboardingState {
  currentStep: number;
  data: OnboardingData;
  totalSteps: number;
  setStep: (step: number) => void;
  nextStep: () => void;
  prevStep: () => void;
  updateData: (partial: Partial<OnboardingData>) => void;
  updateAddress: (partial: Partial<KitchenAddress>) => void;
  updateHours: (day: keyof OperatingHours, hours: { open: string; close: string } | undefined) => void;
  addDocument: (doc: DocumentUpload) => void;
  removeDocument: (type: DocumentUpload['type']) => void;
  hydrateFromServer: (step: number, profile: ServerProfile) => void;
  reset: () => void;
}

const initialData: OnboardingData = {
  fullName: '',
  phone: '',
  email: '',
  kitchenAddress: { line1: '', country: 'IN', city: '', state: '', postalCode: '' },
  businessName: '',
  description: '',
  kitchenType: 'home_kitchen',
  cuisines: [],
  specialties: [],
  yearsOfExperience: '',
  mealsPerDay: '',
  prepTime: '30-45 min',
  serviceRadius: 5,
  minimumOrder: 0,
  deliveryFee: 30,
  operatingHours: DEFAULT_HOURS,
  documents: [],
  acceptedTerms: false,
  acceptedHygienePolicy: false,
  acceptedCancellationPolicy: false,
};

export const useOnboardingStore = create<OnboardingState>()(
  persist(
    (set) => ({
      currentStep: 0,
      data: { ...initialData },
      totalSteps: TOTAL_STEPS,

      setStep: (step) => set({ currentStep: Math.max(0, Math.min(step, TOTAL_STEPS - 1)) }),
      nextStep: () => set((s) => ({ currentStep: Math.min(s.currentStep + 1, TOTAL_STEPS - 1) })),
      prevStep: () => set((s) => ({ currentStep: Math.max(s.currentStep - 1, 0) })),

      updateData: (partial) =>
        set((s) => ({ data: { ...s.data, ...partial } })),

      updateAddress: (partial) =>
        set((s) => ({
          data: { ...s.data, kitchenAddress: { ...s.data.kitchenAddress, ...partial } },
        })),

      updateHours: (day, hours) =>
        set((s) => ({
          data: {
            ...s.data,
            operatingHours: { ...s.data.operatingHours, [day]: hours },
          },
        })),

      addDocument: (doc) =>
        set((s) => ({
          data: {
            ...s.data,
            documents: [...s.data.documents.filter((d) => d.type !== doc.type), doc],
          },
        })),

      removeDocument: (type) =>
        set((s) => ({
          data: {
            ...s.data,
            documents: s.data.documents.filter((d) => d.type !== type),
          },
        })),

      hydrateFromServer: (step, profile) =>
        set((s) => {
          // Only skip hydration if local data matches the server profile (same user)
          // If the emails differ, the local data is from a different user - reset and hydrate
          if (profile.email && s.data.email && s.data.email !== profile.email) {
            // Different user - reset local data first
            localStorage.removeItem('vendor-onboarding');
          } else {
            // Same user - only hydrate if local data is empty
            const localHasData = s.data.businessName?.trim() !== '' && s.data.cuisines.length > 0;
            if (localHasData) return s;
          }

          const merged: Partial<OnboardingData> = {};
          if (profile.fullName) merged.fullName = profile.fullName;
          if (profile.phone) merged.phone = profile.phone;
          if (profile.email) merged.email = profile.email;
          if (profile.businessName) merged.businessName = profile.businessName;
          if (profile.description) merged.description = profile.description;
          if (profile.cuisines?.length) merged.cuisines = profile.cuisines;
          if (profile.specialties?.length) merged.specialties = profile.specialties;
          if (profile.profileImage) merged.profileImage = profile.profileImage;
          if (profile.prepTime) merged.prepTime = profile.prepTime;
          if (profile.serviceRadius) merged.serviceRadius = profile.serviceRadius;
          if (profile.minimumOrder) merged.minimumOrder = profile.minimumOrder;
          if (profile.deliveryFee) merged.deliveryFee = profile.deliveryFee;

          const addr: Partial<KitchenAddress> = {};
          if (profile.addressLine1) addr.line1 = profile.addressLine1;
          if (profile.addressLine2) addr.line2 = profile.addressLine2;
          if (profile.city) addr.city = profile.city;
          if (profile.state) addr.state = profile.state;
          if (profile.postalCode) addr.postalCode = profile.postalCode;
          if (profile.country) addr.country = profile.country;

          return {
            currentStep: Math.max(s.currentStep, step),
            data: {
              ...s.data,
              ...merged,
              kitchenAddress: { ...s.data.kitchenAddress, ...addr },
            },
          };
        }),

      reset: () => {
        localStorage.removeItem('vendor-onboarding');
        set({ currentStep: 0, data: { ...initialData } });
      },
    }),
    {
      name: 'vendor-onboarding',
      storage: createTTLStorage(),
      partialize: (state) => ({ currentStep: state.currentStep, data: state.data }),
    }
  )
);
