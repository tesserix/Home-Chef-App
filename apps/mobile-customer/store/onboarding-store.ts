import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import AsyncStorage from '@react-native-async-storage/async-storage';

// Customer onboarding draft. Previously the three onboarding screens threaded
// their values forward through expo-router navigation params, which lived only
// in in-memory navigation state — so backgrounding the app or a cold start
// mid-onboarding dropped everything and sent the user back to a blank step 1.
// This store persists the partially-filled draft to AsyncStorage so it
// survives, mirroring the vendor/web onboarding stores.
interface OnboardingDraft {
  // Step 1 — basic info
  firstName: string;
  lastName: string;
  phone: string;
  emailVerified: boolean;
  // Step 2 — delivery address
  label: string;
  addressLine1: string;
  addressLine2: string;
  city: string;
  state: string;
  pincode: string;
  // Geocoded from the address autocomplete pick; null when typed manually
  // (server then uses a flat fee + skips delivery-zone checks).
  latitude: number | null;
  longitude: number | null;
  // Step 3 — taste preferences
  cuisinePreferences: string[];
}

interface CustomerOnboardingState extends OnboardingDraft {
  update: (data: Partial<OnboardingDraft>) => void;
  reset: () => void;
}

const initialDraft: OnboardingDraft = {
  firstName: '',
  lastName: '',
  phone: '',
  emailVerified: false,
  label: 'Home',
  addressLine1: '',
  addressLine2: '',
  city: '',
  state: '',
  pincode: '',
  latitude: null,
  longitude: null,
  cuisinePreferences: [],
};

export const useCustomerOnboardingStore = create<CustomerOnboardingState>()(
  persist(
    (set) => ({
      ...initialDraft,
      update: (data) => set(data),
      reset: () => set({ ...initialDraft }),
    }),
    {
      name: 'customer-onboarding-draft',
      storage: createJSONStorage(() => AsyncStorage),
      // Persist only the entered draft, not the action methods.
      partialize: (state) => ({
        firstName: state.firstName,
        lastName: state.lastName,
        phone: state.phone,
        emailVerified: state.emailVerified,
        label: state.label,
        addressLine1: state.addressLine1,
        addressLine2: state.addressLine2,
        city: state.city,
        state: state.state,
        pincode: state.pincode,
        latitude: state.latitude,
        longitude: state.longitude,
        cuisinePreferences: state.cuisinePreferences,
      }),
    },
  ),
);
