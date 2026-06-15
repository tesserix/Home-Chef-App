// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).

import type { Metadata } from 'next';
import { LegalPage, type LegalSection } from '@/components/legal-page';
import { LEGAL_LAST_UPDATED, LEGAL_OPERATOR, LEGAL_SUPPORT_EMAIL } from '@/lib/site';

export const metadata: Metadata = {
  title: 'End User Licence Agreement',
  description: 'The licence terms for using the Fe3dr mobile apps.',
  alternates: { canonical: '/eula/' },
};

const SUMMARY =
  'This End User Licence Agreement (EULA) covers your right to use the Fe3dr mobile apps. We grant you a limited, personal licence to use the apps; we keep ownership of the software. You agree not to copy, resell, or reverse-engineer the apps. Where you install through the Apple App Store or Google Play, those stores terms also apply.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Licence grant',
    paragraphs: [
      `${LEGAL_OPERATOR} ("Fe3dr", "we", "us") grants you a limited, non-exclusive, non-transferable, revocable licence to download and use the Fe3dr mobile apps on devices you own or control, for your personal, non-commercial use, subject to this EULA and to our Terms of Service.`,
    ],
  },
  {
    heading: '2. Ownership and intellectual property',
    paragraphs: [
      'The Fe3dr apps, and all software, design, text, graphics, logos, and other content we provide, are owned by Fe3dr or our licensors and are protected by intellectual-property law. This EULA grants you a licence to use the apps; it does not transfer any ownership to you. All rights not expressly granted are reserved.',
    ],
  },
  {
    heading: '3. Restrictions',
    paragraphs: [
      'You agree not to:',
      '• copy, modify, distribute, sell, sublicense, or lease the apps or any part of them;',
      '• reverse-engineer, decompile, or disassemble the apps, or attempt to derive their source code, except to the extent this restriction is prohibited by law;',
      '• remove or alter any proprietary notices in the apps;',
      '• use the apps to build a competing product, or scrape or harvest data through automated means;',
      '• use the apps in any unlawful way or in breach of our Terms of Service.',
    ],
  },
  {
    heading: '4. Third-party app stores',
    paragraphs: [
      'If you download the apps from the Apple App Store or Google Play, your use is also subject to those stores terms. Apple standard Licensed Application End User Licence Agreement applies as a fallback where this EULA is silent, and where there is a conflict for an App Store download, Apple terms govern the relevant point. The app stores are not responsible for the apps or for support, which Fe3dr provides.',
    ],
  },
  {
    heading: '5. Updates',
    paragraphs: [
      'We may release updates, upgrades, and patches to the apps from time to time. This EULA covers those updates unless they come with separate terms. Some updates may be required for the apps to keep working, and we may stop supporting older versions.',
    ],
  },
  {
    heading: '6. Termination',
    paragraphs: [
      'This licence continues until terminated. It ends automatically if you breach this EULA, and we may suspend or revoke it where required by law or to protect the platform. On termination you must stop using and delete the apps. Sections that by their nature should survive — ownership, restrictions, disclaimers, and limitation of liability — survive termination.',
    ],
  },
  {
    heading: '7. Disclaimer of warranties',
    paragraphs: [
      'The apps are provided "as is" and "as available". To the extent permitted by law, we do not warrant that the apps will be uninterrupted, error-free, or free of harmful components. Nothing in this EULA limits any consumer guarantee or right that cannot be excluded under applicable law.',
    ],
  },
  {
    heading: '8. Limitation of liability',
    paragraphs: [
      'To the extent permitted by law, Fe3dr is not liable for indirect or consequential loss arising from your use of the apps, and our total liability in connection with the apps is limited as set out in our Terms of Service. This section does not limit liability that cannot be limited under applicable law.',
    ],
  },
  {
    heading: '9. Governing law and contact',
    paragraphs: [
      // TODO(counsel): confirm IP-ownership entity and governing law for the EULA (see COUNSEL-REVIEW.md).
      'This EULA is read alongside our Terms of Service and Privacy Policy and relates to a platform operated in India.',
      `For any question about this EULA, contact ${LEGAL_SUPPORT_EMAIL}.`,
    ],
  },
];

export default function EulaPage() {
  return (
    <LegalPage
      title="End User Licence Agreement"
      summary={SUMMARY}
      lastUpdated={LEGAL_LAST_UPDATED}
      sections={SECTIONS}
    />
  );
}
