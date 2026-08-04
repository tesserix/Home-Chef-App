// FSSAI registration guide for home chefs.
//
// Every figure and every click on this page is taken from FSSAI's own pages,
// not from a summary site, because a chef acts on it with their own money:
//
//  • the ₹100-a-year fee is the "Registration" column of the FoSCoS fee table;
//  • the ₹1.5 crore ceiling and the "Home Based Canteens / Dabba Wallas"
//    wording are what the FoSCoS eligibility checker returns for that kind of
//    business;
//  • the three documents are quoted verbatim from FSSAI's own
//    "Documents required for Registration Certificate".
//
// The ₹7,500-a-year figure on the same FSSAI fee table is the CENTRAL LICENCE,
// for operations far larger than a home kitchen. It must never be published
// here as "the fee": it would tell a chef to pay 75× what they actually owe.

import type { Metadata } from 'next';
import { LegalPage, type LegalSection } from '@/components/legal-page';
import {
  CHEFS_EMAIL,
  FOSCOS_APPLY_URL,
  FOSCOS_DOCUMENTS_URL,
  FOSCOS_ELIGIBILITY_URL,
  FOSCOS_FEE_URL,
  FOSCOS_HELPDESK_EMAIL,
  FOSCOS_HELPDESK_PHONE,
  FOSCOS_SIGNUP_GUIDE_URL,
  FSSAI_ASSIST_FEE,
  FSSAI_ASSIST_FEE_INC_GST,
  FSSAI_ASSIST_GST_RATE,
  FSSAI_KIND_OF_BUSINESS,
  FSSAI_REGISTRATION_FEE_PER_YEAR,
  FSSAI_REGISTRATION_TURNOVER_CEILING,
} from '@/lib/site';

export const metadata: Metadata = {
  title: 'FSSAI registration for home chefs',
  description:
    'What an FSSAI registration costs a home kitchen in India (₹100 a year), the documents you need, and how to apply on the government FoSCoS portal step by step.',
  alternates: { canonical: '/fssai/' },
};

const SUMMARY =
  `Every home kitchen selling food in India needs an FSSAI registration by law, and you need one before you can list a menu on Fe3dr. For a home kitchen it is the cheapest tier: ₹${FSSAI_REGISTRATION_FEE_PER_YEAR} a year, paid to the government, not to us. You apply yourself on FSSAI's own portal and it takes about twenty minutes. If you would rather not deal with the form, we will file it for you for ₹${FSSAI_ASSIST_FEE} + GST on top of the government fee.`;

const SECTIONS: LegalSection[] = [
  {
    heading: 'What it costs',
    paragraphs: [
      `A home kitchen falls under Registration — the first of FSSAI's three tiers — which is ₹${FSSAI_REGISTRATION_FEE_PER_YEAR} per year. You choose the term when you apply, up to five years, and pay for all of it upfront:`,
      '• 1 year — ₹100',
      '• 2 years — ₹200',
      '• 3 years — ₹300',
      '• 4 years — ₹400',
      '• 5 years — ₹500',
      `That is the whole government fee. Registration applies while your annual turnover is up to ${FSSAI_REGISTRATION_TURNOVER_CEILING} — which is every home kitchen we have ever onboarded. Above that ceiling you would need a State Licence instead, whose fee varies by state.`,
      "You may see a ₹7,500-a-year figure on FSSAI's fee page. That is the Central Licence column, for national operations of a very different size. It is not what a home kitchen pays, and you should not apply for it.",
      'Longer terms are worth taking. The fee per year is the same either way, and a five-year registration is four fewer renewals to remember — and a lapsed registration means we have to hide your menu until it is current again.',
    ],
    links: [{ href: FOSCOS_FEE_URL, label: 'FSSAI fee structure (foscos.fssai.gov.in)' }],
  },
  {
    heading: 'What you need before you start',
    paragraphs: [
      'Three documents. That is the entire list, quoted from FSSAI:',
      '• A passport-style photo of yourself.',
      '• A government-issued photo ID — Aadhaar, PAN, Voter ID or similar.',
      '• Proof of address of the business activity, but only if you will cook somewhere other than the address printed on that photo ID.',
      'If you cook at the address on your Aadhaar, the third one does not apply and you need just the photo and the ID. Have both as clear phone photos or scans before you begin, because the form asks you to upload them partway through and times out if you go hunting.',
    ],
    links: [
      {
        href: FOSCOS_DOCUMENTS_URL,
        label: 'Documents required for a Registration Certificate (FSSAI, PDF)',
      },
    ],
  },
  {
    heading: 'How to apply, step by step',
    paragraphs: [
      'You apply on FoSCoS, the Food Safety Compliance System — the government portal, and the only place a registration can legitimately be obtained. No agent is required.',
      '• 1. Go to the FoSCoS "Apply for New License/Registration" page, linked below.',
      '• 2. Under "Where is the premise of operation?", choose General. The other tiles are for kitchens inside a railway station, airport or seaport, and for the paid fast-track route — none of which apply to a home kitchen.',
      '• 3. Choose the state your kitchen is in.',
      '• 4. Under "Select Kind of Business", open the Food Services group, then Food Vending Establishment.',
      `• 5. Tick ${FSSAI_KIND_OF_BUSINESS} — FSSAI describes it as "an individual or establishment involved in distribution of packed meals (usually packed lunch) from food service establishments such as home based caterer or restaurants to customers". That is a home kitchen selling tiffin.`,
      `• 6. Select "Annual Turnover upto Rs. 1.5Cr [Registration]" and press Proceed. This is the step that puts you on the ₹${FSSAI_REGISTRATION_FEE_PER_YEAR} tier rather than a licence costing many times more.`,
      '• 7. Sign up or sign in, fill in Form A, upload the two or three documents above, choose your term, and pay online.',
      'You get an application reference number immediately. Keep it — it is how you track progress, and how we can chase on your behalf if it stalls.',
      "If you want to check which tier you fall under before starting anything, the eligibility checker asks the same questions and tells you the answer without submitting an application.",
    ],
    links: [
      { href: FOSCOS_APPLY_URL, label: 'Apply on FoSCoS (the government portal)' },
      { href: FOSCOS_ELIGIBILITY_URL, label: 'Check your tier first — FoSCoS eligibility checker' },
      { href: FOSCOS_SIGNUP_GUIDE_URL, label: "FSSAI's own sign-up walkthrough (PDF)" },
    ],
  },
  {
    heading: `Or we will do it for you — ₹${FSSAI_ASSIST_FEE} + GST`,
    paragraphs: [
      `If the form is not how you want to spend your evening, send us your documents and we will complete the application for you. We charge ₹${FSSAI_ASSIST_FEE} + ${FSSAI_ASSIST_GST_RATE}% GST — ₹${FSSAI_ASSIST_FEE_INC_GST} in total — once per application, whatever term you choose. It is one form either way, so it does not go up for a five-year registration.`,
      'The government fee is separate and unchanged. You pay FSSAI their ₹100 a year; our fee is only for doing the paperwork. A five-year registration filed by us is ₹500 to FSSAI plus ₹59 to us.',
      'Email us these and we will take it from there:',
      '• A passport-style photo of yourself.',
      '• A photo or scan of your Aadhaar, PAN or Voter ID.',
      '• Proof of the kitchen address, if you cook somewhere other than the address on that ID — a recent electricity bill, rent agreement or property document.',
      '• Your full name as it should appear on the registration, your mobile number, your email, and the full kitchen address with pincode.',
      '• How many years you want to register for, from one to five.',
      `Send them to ${CHEFS_EMAIL}. We will confirm what we have, tell you what the government fee comes to, and send you the application reference once it is filed. The registration is issued in your name, to you — it is your licence, not ours, and it stays yours if you ever stop cooking with us.`,
      'What we do not do: we cannot make an application succeed that should not, and we will not submit a turnover figure or a kitchen address that is not true. If your kitchen needs a State Licence rather than a Registration, we will tell you and we will not charge you for finding out.',
    ],
  },
  // The FAQ is one section per question rather than a single block of
  // question-and-answer paragraphs: the template renders every paragraph
  // identically, so a question set in the same face as its answer stops reading
  // as a question at all.
  {
    heading: 'Do I really need one to cook from home for a few customers?',
    paragraphs: [
      `Yes. FSSAI registration is required of any food business in India, whatever its size, and Fe3dr verifies it before you can list a menu. It is also what lets a customer trust a kitchen they have never visited — worth considerably more to you than the ₹${FSSAI_REGISTRATION_FEE_PER_YEAR}.`,
    ],
  },
  {
    heading: 'How long does it take to come through?',
    paragraphs: [
      'The application itself takes about twenty minutes. Issuance sits with the state food-safety department, not with us, and typically takes a few working days. We cannot speed that up, and neither can anyone charging you to try.',
    ],
  },
  {
    heading: 'Can I start setting up on Fe3dr while I wait?',
    paragraphs: [
      'Yes. Build your profile, menu, photos and payout details while the registration is in progress. Your menu goes live to customers once we have verified the registration number.',
    ],
  },
  {
    heading: 'Do I already have one?',
    paragraphs: [
      'FSSAI issues one registration per premises, and any number of kinds of business can be endorsed on it. If you already hold a valid registration for the same address — from a catering business, a shop, anything — you do not need a second one to cook with us. Send us the number you have.',
      'If you cook from more than one address, each address needs its own.',
    ],
  },
  {
    heading: 'What if my turnover grows past the ceiling?',
    paragraphs: [
      `Above ${FSSAI_REGISTRATION_TURNOVER_CEILING} a year you move up to a State Licence. That is a good problem to have, and we will help you with the change when you get there.`,
    ],
  },
  {
    heading: 'What happens if it expires?',
    paragraphs: [
      'We remind you before it lapses. If it does lapse we have to hide your menu until it is current again — selling on an expired registration is an offence for you and for us. Renewal is the same fee and the same form, which is the argument for taking five years upfront.',
    ],
  },
  {
    heading: `Is the ₹${FSSAI_REGISTRATION_FEE_PER_YEAR} paid to Fe3dr?`,
    paragraphs: [
      `No. It goes to FSSAI through the government portal. The only thing you would ever pay us is the ₹${FSSAI_ASSIST_FEE} + GST filing fee, and only if you ask us to file for you.`,
    ],
  },
  {
    heading: 'The FoSCoS site is broken, or my application is stuck.',
    paragraphs: [
      `FSSAI runs its own helpdesk: ${FOSCOS_HELPDESK_EMAIL}, or ${FOSCOS_HELPDESK_PHONE}. Tell us as well — if we filed it for you, chasing it is part of what you paid for.`,
    ],
    links: [{ href: '/vendor-terms/', label: 'Chef terms — your food-safety obligations' }],
  },
];

export default function FssaiPage() {
  return (
    <LegalPage
      title="FSSAI registration for home chefs"
      summary={SUMMARY}
      sections={SECTIONS}
      contactPrompt="Stuck on any of this?"
    />
  );
}
