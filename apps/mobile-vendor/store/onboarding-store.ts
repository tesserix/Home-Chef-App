import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import AsyncStorage from '@react-native-async-storage/async-storage';

type DayHours = { open: string; close: string; closed: boolean };

interface PersonalInfo {
  fullName: string;
  phone: string;
  email: string;
  emailVerified?: boolean;
}

interface KitchenDetails {
  businessName: string;
  cuisines: string[];
  description: string;
  addressLine1: string;
  addressLine2: string;
  city: string;
  state: string;
  postalCode: string;
}

interface Operations {
  operatingHours: Record<string, DayHours>;
  prepTime: string;
  serviceRadius: number;
  // How customers get their food. At least one is required to finish onboarding
  // (a kitchen offering neither can't be activated).
  offersPickup: boolean;
  offersSelfDelivery: boolean;
}

interface Documents {
  idProofUri: string | null;
  idProofType: 'image' | 'pdf' | null;
  fssaiUri: string | null;
  fssaiType: 'image' | 'pdf' | null;
  // 14-digit Food Safety license number. Stored alongside the photo
  // upload so admin tooling (and Wave 3 invoicing) can resolve it
  // without re-reading the document image. Optional during partial
  // form fill; validated on submit.
  fssaiLicenseNumber: string;
  // ISO date string (YYYY-MM-DD). Submitted as the `expiryDate`
  // multipart field on the FSSAI doc upload + persisted on
  // ChefDocument.ExpiryDate so the expiry reminder cron fires.
  fssaiExpiryDate: string;
  // 15-character GSTIN. Optional — chefs below the GST threshold
  // (currently ₹20L turnover) don't need one. When set, printed on
  // customer invoices and used by the chef to claim input tax credit.
  gstin: string;
  // Kitchen compliance media — uploaded GCS URLs (not local uris) for the
  // kitchen photos + walkthrough video the admin reviews. At least one
  // photo AND one video are mandatory to finish the documents step. Both
  // kinds are submitted together as the `kitchenPhotos` array on
  // /chef/onboarding (the video is just another URL — no separate field).
  kitchenMedia: Array<{ url: string; type: 'image' | 'video' }>;
}

interface Policies {
  acceptedTerms: boolean;
  cancellationPolicy: string;
}

// Payout (#739). This slice holds ONLY a non-sensitive record that the step
// was completed — never the account number, IFSC or UPI ID. Those are POSTed
// straight to /chef/payout, which stores them in GCP Secret Manager; putting
// them in this AsyncStorage draft would write a chef's bank details to device
// plaintext and undo the reason the backend blanks those columns.
interface Payout {
  /** True once /chef/payout has accepted the details. */
  configured: boolean;
  /** 'bank_transfer' | 'upi' — the selector, which is not sensitive. */
  method: string;
  /** Masked summary for the Review step, e.g. "Bank ••••9012". */
  summary: string;
}

interface VendorOnboardingState {
  currentStep: number;
  personalInfo: PersonalInfo;
  kitchenDetails: KitchenDetails;
  operations: Operations;
  documents: Documents;
  policies: Policies;
  payout: Payout;
  setStep: (step: number) => void;
  updatePersonalInfo: (data: Partial<PersonalInfo>) => void;
  updateKitchenDetails: (data: Partial<KitchenDetails>) => void;
  updateOperations: (data: Partial<Operations>) => void;
  updateDocuments: (data: Partial<Documents>) => void;
  updatePolicies: (data: Partial<Policies>) => void;
  updatePayout: (data: Partial<Payout>) => void;
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
  kitchenDetails: {
    businessName: '',
    cuisines: [],
    description: '',
    addressLine1: '',
    addressLine2: '',
    city: '',
    state: '',
    postalCode: '',
  },
  operations: {
    operatingHours: DEFAULT_HOURS,
    prepTime: '30min',
    serviceRadius: 10,
    offersPickup: true,
    offersSelfDelivery: false,
  },
  documents: {
    idProofUri: null,
    idProofType: null,
    fssaiUri: null,
    fssaiType: null,
    fssaiLicenseNumber: '',
    fssaiExpiryDate: '',
    gstin: '',
    kitchenMedia: [],
  },
  policies: { acceptedTerms: false, cancellationPolicy: '' },
  payout: { configured: false, method: '', summary: '' },
};

export const useVendorOnboardingStore = create<VendorOnboardingState>()(
  persist(
    (set) => ({
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

      updatePayout: (data) =>
        set((state) => ({
          payout: { ...state.payout, ...data },
        })),

      reset: () => set({ ...initialState }),
    }),
    {
      name: 'vendor-onboarding-draft',
      storage: createJSONStorage(() => AsyncStorage),
      // Only persist user-entered draft data — not the action methods.
      // Document URIs are local file paths and may dangle across cold
      // starts; the documents step re-validates them on focus.
      partialize: (state) => ({
        currentStep: state.currentStep,
        personalInfo: state.personalInfo,
        kitchenDetails: state.kitchenDetails,
        operations: state.operations,
        documents: state.documents,
        policies: state.policies,
        // Safe to persist: a masked summary and the method selector only.
        // The sensitive fields never enter this store (see the Payout type).
        payout: state.payout,
      }),
    },
  ),
);
