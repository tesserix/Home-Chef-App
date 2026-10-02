export type MarketCode = 'IN' | 'AU' | 'NZ';

export interface Market {
  code: MarketCode;
  name: string;
  currency: 'INR' | 'AUD' | 'NZD';
  postcodeLabel: string;
  postcodeLength: number;
  regionLabel: string;
  /** Fixed region list; null where states come from the API (India). */
  regions: string[] | null;
  licenceDocType: 'fssai_license' | 'food_safety_cert';
  /** 'bank' collects IFSC details in-app; 'stripe' hands off to Stripe Connect. */
  payoutRail: 'bank' | 'stripe';
  businessNumberLabel: string | null;
  /** 0-based month the tax year starts: April (3) or July (6). */
  taxYearStartMonth: number;
}

const MARKETS: Record<MarketCode, Market> = {
  IN: {
    code: 'IN',
    name: 'India',
    currency: 'INR',
    postcodeLabel: 'PIN code',
    postcodeLength: 6,
    regionLabel: 'State',
    regions: null,
    licenceDocType: 'fssai_license',
    payoutRail: 'bank',
    businessNumberLabel: null,
    taxYearStartMonth: 3,
  },
  AU: {
    code: 'AU',
    name: 'Australia',
    currency: 'AUD',
    postcodeLabel: 'Postcode',
    postcodeLength: 4,
    regionLabel: 'State or territory',
    regions: [
      'Australian Capital Territory',
      'New South Wales',
      'Northern Territory',
      'Queensland',
      'South Australia',
      'Tasmania',
      'Victoria',
      'Western Australia',
    ],
    licenceDocType: 'food_safety_cert',
    payoutRail: 'stripe',
    businessNumberLabel: 'ABN',
    taxYearStartMonth: 6,
  },
  NZ: {
    code: 'NZ',
    name: 'New Zealand',
    currency: 'NZD',
    postcodeLabel: 'Postcode',
    postcodeLength: 4,
    regionLabel: 'Region',
    regions: [
      'Auckland',
      'Bay of Plenty',
      'Canterbury',
      'Gisborne',
      "Hawke's Bay",
      'Manawatū-Whanganui',
      'Marlborough',
      'Nelson',
      'Northland',
      'Otago',
      'Southland',
      'Taranaki',
      'Tasman',
      'Waikato',
      'Wellington',
      'West Coast',
    ],
    licenceDocType: 'food_safety_cert',
    payoutRail: 'stripe',
    businessNumberLabel: 'NZBN',
    taxYearStartMonth: 3,
  },
};

export const MARKET_CODES: MarketCode[] = ['IN', 'AU', 'NZ'];

/** Drafts saved before the country field existed are Indian kitchens. */
export function getMarket(code: string | undefined): Market {
  return MARKETS[(code?.toUpperCase() ?? '') as MarketCode] ?? MARKETS.IN;
}

export function isValidPostcode(value: string, code: string | undefined): boolean {
  const { postcodeLength } = getMarket(code);
  return new RegExp(`^\\d{${postcodeLength}}$`).test(value.trim());
}

export function taxYearStart(market: Market, d: Date = new Date()): number {
  return d.getMonth() >= market.taxYearStartMonth ? d.getFullYear() : d.getFullYear() - 1;
}
