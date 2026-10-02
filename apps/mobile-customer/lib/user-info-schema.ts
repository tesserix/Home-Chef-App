import { z } from 'zod';
import { isValidPhone } from '@homechef/mobile-shared/validation/phone';

export type PhoneCountry = 'IN' | 'AU' | 'NZ';

// Mirrors services.InvalidPhoneMessage so the app and API say the same thing.
const PHONE_ERRORS: Record<PhoneCountry, string> = {
  IN: 'Enter a valid 10-digit Indian mobile number',
  AU: 'Enter a valid 9-digit mobile number, without the leading 0',
  NZ: 'Enter a valid NZ mobile number, without the leading 0',
};

export function buildUserInfoSchema(country: PhoneCountry) {
  return z.object({
    firstName: z.string().min(2, 'First name must be at least 2 characters'),
    lastName: z.string().min(2, 'Last name must be at least 2 characters'),
    phone: z.string().refine((v) => isValidPhone(v, country), PHONE_ERRORS[country]),
  });
}

export type UserInfoForm = z.infer<ReturnType<typeof buildUserInfoSchema>>;
