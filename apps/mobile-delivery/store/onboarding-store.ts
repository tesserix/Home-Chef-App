import { create } from 'zustand';

interface PersonalInfo {
  city: string;
  emergencyContact: string;
  emergencyPhone: string;
  vehicleType: 'bike' | 'scooter' | 'car' | 'van';
  dateOfBirth: string;
}

interface VehicleDetails {
  vehicleType: string;
  vehicleMake: string;
  vehicleModel: string;
  vehicleYear: string;
  vehicleColor: string;
  vehicleNumber: string;
  licenseNumber: string;
}

interface Documents {
  drivingLicenseUri: string | null;
  idProofUri: string | null;
  vehicleRcUri: string | null;
}

interface PayoutDetails {
  payoutMethod: 'bank' | 'upi';
  bankAccountNumber: string;
  bankIFSC: string;
  upiId: string;
}

interface SubscriptionInfo {
  selectedPlanId: string | null;
  planName: string;
}

interface DriverOnboardingState {
  currentStep: number;
  personalInfo: PersonalInfo;
  vehicleDetails: VehicleDetails;
  documents: Documents;
  payoutDetails: PayoutDetails;
  subscriptionInfo: SubscriptionInfo;
  setStep: (step: number) => void;
  updatePersonalInfo: (data: Partial<PersonalInfo>) => void;
  updateVehicleDetails: (data: Partial<VehicleDetails>) => void;
  updateDocuments: (data: Partial<Documents>) => void;
  updatePayoutDetails: (data: Partial<PayoutDetails>) => void;
  updateSubscriptionInfo: (data: Partial<SubscriptionInfo>) => void;
  reset: () => void;
}

const defaultPersonalInfo: PersonalInfo = {
  city: '',
  emergencyContact: '',
  emergencyPhone: '',
  vehicleType: 'bike',
  dateOfBirth: '',
};

const defaultVehicleDetails: VehicleDetails = {
  vehicleType: '',
  vehicleMake: '',
  vehicleModel: '',
  vehicleYear: '',
  vehicleColor: '',
  vehicleNumber: '',
  licenseNumber: '',
};

const defaultDocuments: Documents = {
  drivingLicenseUri: null,
  idProofUri: null,
  vehicleRcUri: null,
};

const defaultPayoutDetails: PayoutDetails = {
  payoutMethod: 'bank',
  bankAccountNumber: '',
  bankIFSC: '',
  upiId: '',
};

const defaultSubscriptionInfo: SubscriptionInfo = {
  selectedPlanId: null,
  planName: '',
};

export const useDriverOnboardingStore = create<DriverOnboardingState>((set) => ({
  currentStep: 1,
  personalInfo: { ...defaultPersonalInfo },
  vehicleDetails: { ...defaultVehicleDetails },
  documents: { ...defaultDocuments },
  payoutDetails: { ...defaultPayoutDetails },
  subscriptionInfo: { ...defaultSubscriptionInfo },

  setStep: (step) => set({ currentStep: step }),

  updatePersonalInfo: (data) =>
    set((state) => ({
      personalInfo: { ...state.personalInfo, ...data },
    })),

  updateVehicleDetails: (data) =>
    set((state) => ({
      vehicleDetails: { ...state.vehicleDetails, ...data },
    })),

  updateDocuments: (data) =>
    set((state) => ({
      documents: { ...state.documents, ...data },
    })),

  updatePayoutDetails: (data) =>
    set((state) => ({
      payoutDetails: { ...state.payoutDetails, ...data },
    })),

  updateSubscriptionInfo: (data) =>
    set((state) => ({
      subscriptionInfo: { ...state.subscriptionInfo, ...data },
    })),

  reset: () =>
    set({
      currentStep: 1,
      personalInfo: { ...defaultPersonalInfo },
      vehicleDetails: { ...defaultVehicleDetails },
      documents: { ...defaultDocuments },
      payoutDetails: { ...defaultPayoutDetails },
      subscriptionInfo: { ...defaultSubscriptionInfo },
    }),
}));
