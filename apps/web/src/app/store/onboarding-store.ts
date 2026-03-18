import { create } from 'zustand';
import { persist } from 'zustand/middleware';

const TOTAL_STEPS = 3;

export interface OnboardingData {
  // Step 1: Basic Info
  firstName: string;
  lastName: string;
  phone: string;
  dateOfBirth: string;
  // Step 2: Preferences
  dietaryPreferences: string[];
  foodAllergies: string[];
  cuisinePreferences: string[];
  spiceTolerance: string;
  householdSize: string;
  // Step 3: Address
  addressLabel: string;
  addressLine1: string;
  addressLine2: string;
  addressCity: string;
  addressState: string;
  addressPostalCode: string;
  addressCountry: string;
}

interface OnboardingState {
  currentStep: number;
  data: OnboardingData;
  totalSteps: number;
  setStep: (step: number) => void;
  nextStep: () => void;
  prevStep: () => void;
  updateData: (partial: Partial<OnboardingData>) => void;
  reset: () => void;
}

const initialData: OnboardingData = {
  firstName: '',
  lastName: '',
  phone: '',
  dateOfBirth: '',
  dietaryPreferences: [],
  foodAllergies: [],
  cuisinePreferences: [],
  spiceTolerance: '',
  householdSize: '',
  addressLabel: 'Home',
  addressLine1: '',
  addressLine2: '',
  addressCity: '',
  addressState: '',
  addressPostalCode: '',
  addressCountry: 'IN',
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

      reset: () => {
        localStorage.removeItem('customer-onboarding');
        set({ currentStep: 0, data: { ...initialData } });
      },
    }),
    {
      name: 'customer-onboarding',
      partialize: (state) => ({ currentStep: state.currentStep, data: state.data }),
    }
  )
);
