/**
 * The sibling sites a partner signs up on. fe3dr.com is the customer storefront;
 * a home chef who lands here has to be told, plainly and from anywhere on the
 * site, that their side of the platform is somewhere else.
 *
 * One constant because the URL was written out at each of the three places that
 * linked to it, and a fourth was about to be added.
 */

/** Where a home chef onboards their kitchen and manages orders. */
export const VENDOR_PORTAL_URL = 'https://vendors.fe3dr.com';

/**
 * The wording every chef entry point uses. "Become a Chef" reads as an
 * aspiration; a person who already cooks needs to know this is where they list
 * their kitchen, and that it leaves this site.
 */
export const VENDOR_CTA_LABEL = 'Add your kitchen';
export const VENDOR_CTA_BLURB =
  'Cook from home? List your kitchen on Fe3dr, take orders in your area and get paid weekly.';

/**
 * What it costs to start, stated in figures we can stand behind.
 *
 * The honest version of "it's free" is: nothing to join, nothing monthly, and
 * nothing at all until an order is paid for. The one real upfront cost is the
 * FSSAI Basic Registration every home kitchen in India needs by law — 100 a year,
 * paid to FSSAI, not to us. The commission is named too: a pitch that implies a
 * chef keeps 100% of every order would be the kind of claim this platform gets
 * held to later.
 *
 * COMMISSION_PCT mirrors services.DefaultCommissionRate (apps/api). It is
 * admin-tunable per the platform settings, so treat this as the advertised
 * standard rate and update both together.
 */
export const VENDOR_COST_POINTS = [
  {
    figure: '₹0',
    label: 'to join and list',
    detail: 'No signup fee, no monthly fee, no charge for your menu or photos.',
  },
  {
    figure: '₹100',
    label: 'FSSAI registration, per year',
    detail:
      'The food safety licence every home kitchen in India needs. Paid to FSSAI, not to us — we walk you through it.',
  },
  {
    figure: '6%',
    label: 'commission per order',
    detail: 'Only on orders you actually sell. Payouts land weekly, tips in full.',
  },
] as const;
