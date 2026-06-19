import { useMutation } from '@tanstack/react-query';
import { api } from '../lib/api';

// Promo code validation (#39). Previews the discount for a code against the
// cart subtotal + chef before checkout; the server re-validates and re-computes
// the discount at order time, so this is a preview only — never trusted for the
// final price. Mirrors POST /v1/promo/validate.

export interface PromoValidationInput {
  code: string;
  orderTotal: number; // cart subtotal the discount applies to
  chefId?: string; // scopes chef-funded codes to the cart's chef
}

export interface PromoValidationResult {
  valid: boolean;
  code: string;
  discountType: 'percentage' | 'fixed';
  discountValue: number;
  discount: number;
  description?: string;
  fundingSource?: 'platform' | 'chef';
}

/** Extract the API's error message (string, or {message,...} for min-order). */
export function promoErrorMessage(err: unknown): string {
  const data = (err as { response?: { data?: { error?: unknown } } })?.response?.data?.error;
  if (typeof data === 'string') return data;
  if (data && typeof data === 'object' && 'message' in data) {
    return String((data as { message?: unknown }).message ?? 'Invalid promo code');
  }
  return 'Invalid promo code';
}

export function useValidatePromo() {
  return useMutation<PromoValidationResult, Error, PromoValidationInput>({
    mutationFn: async (input) => {
      const r = await api.post('/v1/promo/validate', {
        code: input.code.trim().toUpperCase(),
        orderTotal: input.orderTotal,
        chefId: input.chefId,
      });
      return r.data as PromoValidationResult;
    },
  });
}
