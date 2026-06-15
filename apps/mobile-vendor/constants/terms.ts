export const VENDOR_TERMS_TEXT = `By joining HomeChef as a vendor you agree to maintain food hygiene standards per FSSAI regulations, ensure accurate menu descriptions, prepare orders within your stated prep time, and comply with all applicable local food safety laws. HomeChef reserves the right to suspend accounts that receive repeated hygiene complaints or fail document verification. Full terms available at fe3dr.com/vendor-terms.`;

export const CANCELLATION_POLICY_OPTIONS = [
  { label: 'No cancellations after order accepted', value: 'no_cancellations' },
  { label: 'Up to 1 hour before prep start', value: 'up_to_1_hour' },
  { label: 'Up to 30 mins before prep start', value: 'up_to_30_mins' },
] as const;

export type CancellationPolicy = (typeof CANCELLATION_POLICY_OPTIONS)[number]['value'];
