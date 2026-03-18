import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { apiClient } from '@/shared/services/api-client';

export interface CurrencyInfo {
  code: string;
  name: string;
  symbol: string;
  decimalPlaces: number;
}

interface CurrencyState {
  code: string;
  symbol: string;
  decimals: number;
  rates: Record<string, number>;
  ratesLastFetched: number;
  currencies: CurrencyInfo[];
  detected: boolean;
}

interface CurrencyActions {
  setCurrency: (code: string) => void;
  fetchRates: () => Promise<void>;
  fetchCurrencies: () => Promise<void>;
  detectCurrency: () => Promise<void>;
  initFromProfile: (preferredCurrency: string) => void;
}

const RATES_TTL = 12 * 60 * 60 * 1000; // 12 hours in ms

export const useCurrencyStore = create<CurrencyState & CurrencyActions>()(
  persist(
    (set, get) => ({
      code: 'INR',
      symbol: '₹',
      decimals: 2,
      rates: { INR: 1 },
      ratesLastFetched: 0,
      currencies: [],
      detected: false,

      setCurrency: (code: string) => {
        const currencies = get().currencies;
        const match = currencies.find((c) => c.code === code);
        if (match) {
          set({
            code: match.code,
            symbol: match.symbol,
            decimals: match.decimalPlaces,
          });
        } else {
          set({ code });
        }
      },

      fetchRates: async () => {
        const now = Date.now();
        if (now - get().ratesLastFetched < RATES_TTL && Object.keys(get().rates).length > 1) {
          return;
        }
        try {
          const data = await apiClient.get<{ base: string; rates: Record<string, number> }>(
            '/currencies/rates',
          );
          set({ rates: data.rates, ratesLastFetched: now });
        } catch {
          // Keep existing rates
        }
      },

      fetchCurrencies: async () => {
        try {
          const data = await apiClient.get<CurrencyInfo[]>('/currencies');
          set({ currencies: data });
        } catch {
          // Keep existing list
        }
      },

      detectCurrency: async () => {
        if (get().detected) return;
        try {
          const data = await apiClient.get<{
            countryCode: string;
            currencyCode: string;
            currencySymbol: string;
          }>('/currencies/detect');
          set({
            code: data.currencyCode,
            symbol: data.currencySymbol,
            detected: true,
          });
          // Find decimal places from currencies list
          const currencies = get().currencies;
          const match = currencies.find((c) => c.code === data.currencyCode);
          if (match) {
            set({ decimals: match.decimalPlaces });
          }
        } catch {
          set({ detected: true }); // Don't retry
        }
      },

      initFromProfile: (preferredCurrency: string) => {
        if (!preferredCurrency || preferredCurrency === get().code) return;
        const currencies = get().currencies;
        const match = currencies.find((c) => c.code === preferredCurrency);
        if (match) {
          set({
            code: match.code,
            symbol: match.symbol,
            decimals: match.decimalPlaces,
            detected: true,
          });
        } else {
          set({ code: preferredCurrency, detected: true });
        }
      },
    }),
    {
      name: 'homechef-currency',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        code: state.code,
        symbol: state.symbol,
        decimals: state.decimals,
        rates: state.rates,
        ratesLastFetched: state.ratesLastFetched,
        detected: state.detected,
      }),
    },
  ),
);
