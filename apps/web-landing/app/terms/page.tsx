// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).

import type { Metadata } from 'next';
import { LegalPage, type LegalSection } from '@/components/legal-page';
import { LEGAL_LAST_UPDATED, LEGAL_OPERATOR, LEGAL_SUPPORT_EMAIL } from '@/lib/site';

export const metadata: Metadata = {
  title: 'Terms of Service',
  description: 'The terms that govern your use of Home Chef.',
  alternates: { canonical: '/terms/' },
};

const SUMMARY =
  'Fe3dr is a marketplace that connects you with home cooks in your city. We do not cook your food. When you place an order, your contract for the meal is with the chef, and your contract for the ride is with the driver. We hold your payment safely until your order is delivered, and we refund you within seven working days when a refund is due. You read the allergen and ingredient information the chef provides; you do not order what you cannot eat. If something goes wrong, our Grievance Officer is your first port of call.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Acceptance of these terms',
    paragraphs: [
      'These terms form a contract between you and us. They apply every time you visit our website, create an account, or place an order. By using Fe3dr, you agree to these terms. If you do not agree, please do not use the service.',
      'Some sections affect important rights — payments, refunds, food safety, liability, and dispute resolution change what you can recover if something goes wrong. Please read them with care.',
    ],
  },
  {
    heading: '2. What Fe3dr does',
    paragraphs: [
      `Fe3dr, operated by ${LEGAL_OPERATOR}, is a technology platform. We connect customers who want home-cooked food with independent home cooks (each a Chef) and independent delivery partners (each a Driver). We list menus, take payments, and pass orders to the right chef and driver.`,
      'We are a marketplace, not a restaurant. We do not cook your food, store it, package it, or carry it. The chef cooks the food in their own kitchen; the driver picks it up and brings it to you. We verify chefs and drivers before they start and we set service standards, but we do not control the kitchen, the cooking, or the route.',
      'We operate as an intermediary under the Information Technology Act, 2000. Our role and the protection that goes with it is explained in section 11.',
    ],
  },
  {
    heading: '3. Your account',
    paragraphs: [
      'You must be at least 18 years old to use Fe3dr. When you sign up, please give us accurate information — your name, phone number, email, and delivery address — and keep it up to date, because we rely on it to deliver your food.',
      'Keep your password safe. You are responsible for everything that happens through your account unless you tell us someone else has used it. You may close your account at any time from your settings. We may suspend or close your account if you break these terms, place fraudulent orders, or if a court or regulator tells us to — and we will tell you why, unless we are barred by law from doing so.',
    ],
  },
  {
    heading: '4. Placing an order',
    paragraphs: [
      'When you place an Order, you make an offer to buy a meal from the chef on the terms shown at checkout. The chef may accept or decline. If the chef declines, we refund you in full.',
      'Your contract for the food is with the chef. Your contract for the delivery is with the driver, where the driver is independent. We act as the chef and driver collection agent for the money, and as their notice agent for messages about the order. The chef sets the menu, the price, and the cooking style; the driver chooses the route.',
      'We do not guarantee a specific delivery time. Where we display an expected time, we mean it as a typical estimate, not a promise. Traffic, weather, festivals, and chef capacity all affect actual delivery times. Once the order is paid for and accepted, we send a confirmation that sets out what you ordered, what you paid, and the chef allergen notes.',
    ],
  },
  {
    heading: '5. Payment',
    paragraphs: [
      'We accept payment in Indian rupees through our regulated payment partners, currently Razorpay and Stripe. We never store your full card number — the partner you use stores the card details on their systems.',
      'We collect the money for the chef and driver and hold it in escrow until the order is delivered. After delivery, we release the chef share on their settlement cycle and the driver share to the driver. Settlement timings are governed by the Reserve Bank of India Payment Aggregator framework.',
      'Menu prices include the chef cooking charges. We also charge a delivery fee and a small platform fee, both shown at checkout before you pay. Taxes such as GST are shown as a separate line, calculated as required by Indian tax law. When a refund is due, we start it within 24 hours of approval and the money usually reaches your account within seven working days.',
      'If a payment fails, no order is placed. If a payment looks suspicious, we may hold it for review or ask you for more information before we accept the order, to protect you and the chef from fraud.',
    ],
  },
  {
    heading: '6. Cancellations and refunds',
    paragraphs: [
      'You can cancel an order any time before the chef starts cooking and get a full refund. Once the chef has begun preparing the food, the chef chooses whether to cancel and how much to refund, because the ingredients have already been used. Each chef sets a clear cancellation policy that you see at checkout.',
      'If the chef cancels after accepting, or the food is not delivered, you get a full refund. If the food arrives in poor condition, contact us within four hours of delivery with a photo and a short description, and we will work with the chef to put it right. For full detail, see our Refund & Cancellation Policy at fe3dr.com/refund.',
    ],
  },
  {
    heading: '7. Food safety and allergens',
    paragraphs: [
      'The chef cooks the food and is responsible for what is in it. The chef labels each dish with its ingredients and any common allergens. We facilitate that disclosure: we display what the chef tells us on the menu and order screen. We do not test the food or inspect every kitchen for every order.',
      'If you have a food allergy, an intolerance, or a religious or dietary restriction, please read the ingredient list on every dish before you order, and message the chef where you are not sure. Home kitchens often handle many ingredients in the same space, so cross-contact is possible even when an ingredient is not listed. Every chef on Fe3dr must hold the food-business registration or licence required by the Food Safety and Standards Authority of India (FSSAI), which we check before a chef can list a menu.',
    ],
  },
  {
    heading: '8. Reviews and content',
    paragraphs: [
      'You can rate and review a chef after an order. Please keep reviews honest, based on your own experience, and relevant to the order. When you post a review, photo, or comment you keep ownership of it and grant us a worldwide, non-exclusive, royalty-free licence to host, display, and share it on Fe3dr and in our marketing.',
      'Do not post anything false, that names third parties without their consent, that targets a person rather than the food or service, that is unlawful, or that infringes someone else rights. We may take down reviews that break these rules or that a court tells us to remove. Defamation rules apply to reviews: a false statement of fact that damages a chef business may expose you to a civil claim, while honest opinion and fair criticism are protected.',
    ],
  },
  {
    heading: '9. Prohibited use',
    paragraphs: [
      'When you use Fe3dr, please do not:',
      '• place an order you do not intend to pay for or receive;',
      '• pretend to be someone else, use a false name, or use a payment method that is not yours;',
      '• resell the food, including through dark stores, cloud kitchens, or other marketplaces;',
      '• scrape, copy, or republish menu data, prices, or photos for commercial purposes without our written consent;',
      '• interfere with the platform, including by uploading malware, probing for weaknesses, or overloading our systems;',
      '• harass a chef, a driver, or our staff;',
      '• use Fe3dr to break any law, including tax law, food-safety law, and consumer-protection law.',
      'If we believe you have used Fe3dr for any of the above, we may pause your account, refuse future orders, and report you to the relevant authorities.',
    ],
  },
  {
    heading: '10. Liability',
    paragraphs: [
      'Nothing in these terms removes any right you have under Indian consumer-protection law, including the Consumer Protection Act, 2019 and the Consumer Protection (E-Commerce) Rules, 2020, that we cannot remove by contract.',
      'For everything else, our total liability to you for any single order is limited to the platform fees we collected from you for that order. We do not guarantee food quality, taste, or freshness beyond what a reasonable home cook can deliver, or any specific delivery time. We are not liable for indirect or consequential loss, including missed meetings, lost income, or emotional distress, except where Indian law requires otherwise.',
      'We are not responsible for the chef cooking choices, the driver route choices, or any third-party service we rely on. We do our best to keep Fe3dr available, but we may take the service down for maintenance, security work, or to respond to an incident, and we do not promise it will be uninterrupted or error-free.',
    ],
  },
  {
    heading: '11. Intermediary status',
    paragraphs: [
      'We operate Fe3dr as an intermediary under section 79 of the Information Technology Act, 2000 and the IT (Intermediary Guidelines and Digital Media Ethics Code) Rules, 2021. We host content that chefs, drivers, and customers post; we do not author menus, reviews, or driver messages, and we are not responsible for them in the way the author is.',
      'Chefs and drivers are independent. They are not our employees. They are responsible for their own tax, their own insurance, and their own compliance with the law. We follow the takedown process the Intermediary Rules require: if you believe content on Fe3dr is unlawful, report it to our Grievance Officer and we respond within the timelines those rules require.',
    ],
  },
  {
    heading: '12. Governing law and dispute resolution',
    paragraphs: [
      // TODO(counsel): the operating entity carries an Australian suffix while operations, food safety, payments, and tax are India-based — confirm the single governing-law / jurisdiction clause before launch (see COUNSEL-REVIEW.md).
      'These terms relate to a platform operated in India and are read alongside Indian food-safety (FSSAI), payments (RBI), and tax (GST) law. If you are a consumer, nothing here stops you from bringing a complaint to a District Consumer Disputes Redressal Commission with jurisdiction over your place of residence, as the Consumer Protection Act, 2019 allows.',
      'If you have a problem, please tell us first — most problems are sorted out quickly through our Grievance Officer, who acknowledges complaints within 48 hours and aims to resolve them within 30 days. You may also escalate to the National Consumer Helpline, a District Consumer Disputes Redressal Commission, the Data Protection Board of India, or the Reserve Bank of India Integrated Ombudsman Scheme for unresolved payment disputes. For commercial disputes that are not consumer matters, we agree to arbitration in India under the Arbitration and Conciliation Act, 1996, seated in Mumbai, before a single arbitrator, in English.',
    ],
  },
  {
    heading: '13. Privacy and changes',
    paragraphs: [
      'We handle your personal data under the Digital Personal Data Protection Act, 2023. Our Privacy Policy at fe3dr.com/privacy explains what we collect, why, how long we keep it, and how to exercise your rights. By using Fe3dr, you confirm that you have read it.',
      'We may update these terms. When we do, we update the "Last updated" date at the top of this page. For material changes — for example a change to how we handle refunds, how disputes are resolved, or what fees you pay — we give you 30 days notice by email and in the app. We do not change the terms that apply to an order you have already placed.',
    ],
  },
  {
    heading: '14. Contact us',
    paragraphs: [
      `For routine questions, and for complaints, data-protection requests, and notices that need a formal response, contact our support and Grievance Officer at ${LEGAL_SUPPORT_EMAIL}. We acknowledge complaints within 48 hours and aim to resolve them within 30 days, in line with the Consumer Protection (E-Commerce) Rules, 2020 and the Digital Personal Data Protection Act, 2023.`,
    ],
  },
];

export default function TermsPage() {
  return (
    <LegalPage
      title="Terms of Service"
      summary={SUMMARY}
      lastUpdated={LEGAL_LAST_UPDATED}
      sections={SECTIONS}
    />
  );
}
