import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import AsyncStorage from '@react-native-async-storage/async-storage';

// Sign-up draft: name/email/phone entered on the register screen survive an
// app kill until the account is actually created or the user cancels.
// Passwords are never persisted.
interface RegisterDraft {
  firstName: string;
  lastName: string;
  email: string;
  phone: string;
}

interface RegisterDraftState extends RegisterDraft {
  update: (data: Partial<RegisterDraft>) => void;
  reset: () => void;
}

const initialDraft: RegisterDraft = {
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
};

export const useRegisterDraftStore = create<RegisterDraftState>()(
  persist(
    (set) => ({
      ...initialDraft,
      update: (data) => set(data),
      reset: () => set({ ...initialDraft }),
    }),
    {
      name: 'customer-register-draft',
      storage: createJSONStorage(() => AsyncStorage),
      partialize: (s) => ({
        firstName: s.firstName,
        lastName: s.lastName,
        email: s.email,
        phone: s.phone,
      }),
    },
  ),
);
