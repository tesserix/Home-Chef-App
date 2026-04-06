import { create } from 'zustand';

type DayHours = { open: string; close: string; closed: boolean };

interface PersonalInfo {
  fullName: string;
  phone: string;
  email: string;
}

interface KitchenDetails {
  businessName: string;
  cuisines: string[];
  description: string;
}

interface Operations {
  operatingHours: Record<string, DayHours>;
  prepTime: string;
  serviceRadius: number;
}

interface Documents {
  idProofUri: string | null;
  idProofType: 'image' | 'pdf' | null;
  fssaiUri: string | null;
  fssaiType: 'image' | 'pdf' | null;
}

interface Policies {
  acceptedTerms: boolean;
  cancellationPolicy: string;
}

interface VendorOnboardingState {
  currentStep: number;
  personalInfo: PersonalInfo;
  kitchenDetails: KitchenDetails;
  operations: Operations;
  documents: Documents;
  policies: Policies;
  setStep: (step: number) => void;
  updatePersonalInfo: (data: Partial<PersonalInfo>) => void;
  updateKitchenDetails: (data: Partial<KitchenDetails>) => void;
  updateOperations: (data: Partial<Operations>) => void;
  updateDocuments: (data: Partial<Documents>) => void;
  updatePolicies: (data: Partial<Policies>) => void;
  reset: () => void;
}

const DEFAULT_HOURS: Record<string, DayHours> = {
  monday: { open: '09:00', close: '21:00', closed: false },
  tuesday: { open: '09:00', close: '21:00', closed: false },
  wednesday: { open: '09:00', close: '21:00', closed: false },
  thursday: { open: '09:00', close: '21:00', closed: false },
  friday: { open: '09:00', close: '21:00', closed: false },
  saturday: { open: '09:00', close: '21:00', closed: false },
  sunday: { open: '09:00', close: '21:00', closed: false },
};

const initialState = {
  currentStep: 1,
  personalInfo: { fullName: '', phone: '', email: '' },
  kitchenDetails: { businessName: '', cuisines: [], description: '' },
  operations: {
    operatingHours: DEFAULT_HOURS,
    prepTime: '30min',
    serviceRadius: 10,
  },
  documents: {
    idProofUri: null,
    idProofType: null,
    fssaiUri: null,
    fssaiType: null,
  },
  policies: { acceptedTerms: false, cancellationPolicy: '' },
};

export const useVendorOnboardingStore = create<VendorOnboardingState>((set) => ({
  ...initialState,

  setStep: (step) => set({ currentStep: step }),

  updatePersonalInfo: (data) =>
    set((state) => ({
      personalInfo: { ...state.personalInfo, ...data },
    })),

  updateKitchenDetails: (data) =>
    set((state) => ({
      kitchenDetails: { ...state.kitchenDetails, ...data },
    })),

  updateOperations: (data) =>
    set((state) => ({
      operations: { ...state.operations, ...data },
    })),

  updateDocuments: (data) =>
    set((state) => ({
      documents: { ...state.documents, ...data },
    })),

  updatePolicies: (data) =>
    set((state) => ({
      policies: { ...state.policies, ...data },
    })),

  reset: () => set({ ...initialState }),
}));
