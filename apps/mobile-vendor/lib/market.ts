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
  timeZone: string;
}

const MARKETS: Record<MarketCode, Market> = {
  IN: {
    code: 'IN',
    timeZone: 'Asia/Kolkata',
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
    timeZone: 'Australia/Sydney',
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
    timeZone: 'Pacific/Auckland',
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
  const [year, month] = marketDateISO(market, d).split('-').map(Number);
  return month! > market.taxYearStartMonth ? year! : year! - 1;
}

export function marketDateISO(market: Market, d: Date = new Date(), daysAgo = 0): string {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: market.timeZone, year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(d);
  const value = (type: string) => Number(parts.find((p) => p.type === type)!.value);
  return new Date(Date.UTC(value('year'), value('month') - 1, value('day') - daysAgo)).toISOString().slice(0, 10);
}

export function formatExpenseDate(iso: string, market: Market): string {
  return new Date(iso.length === 10 ? `${iso}T12:00:00Z` : iso).toLocaleDateString(`en-${market.code}`, {
    timeZone: iso.length === 10 ? 'UTC' : market.timeZone, day: '2-digit', month: 'short', year: 'numeric',
  });
}
