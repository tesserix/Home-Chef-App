// End User Licence Agreement — reachable from Profile > Legal.
// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).

import { LegalScreen, type LegalSection } from '../components/legal/LegalScreen';

const LAST_UPDATED = '11 June 2026';

const INTRO =
  'This End User Licence Agreement (EULA) covers your right to use the Fe3dr customer app, provided by Tesserix Pty Ltd ("Fe3dr", "we", "us"). We grant you a limited licence to use the app; we keep ownership of the software.';

const SECTIONS: LegalSection[] = [
  {
    heading: '1. Licence grant',
    paragraphs: [
      'Fe3dr grants you a limited, non-exclusive, non-transferable, revocable licence to download and use the Fe3dr customer app on devices you own or control, for your personal, non-commercial use, subject to this EULA and to our Terms of Service.',
    ],
  },
  {
    heading: '2. Ownership and intellectual property',
    paragraphs: [
      'The Fe3dr app, and all software, design, text, graphics, and logos we provide, are owned by Fe3dr or our licensors and are protected by intellectual-property law. This EULA grants you a licence to use the app; it does not transfer ownership. All rights not expressly granted are reserved.',
    ],
  },
  {
    heading: '3. Restrictions',
    paragraphs: [
      '• Do not copy, modify, distribute, sell, sublicense, or lease the app or any part of it.',
      '• Do not reverse-engineer, decompile, or disassemble the app except to the extent this restriction is prohibited by law.',
      '• Do not remove or alter any proprietary notices.',
      '• Do not use the app to build a competing product, or scrape or harvest data through automated means.',
      '• Do not use the app unlawfully or in breach of our Terms of Service.',
    ],
  },
  {
    heading: '4. Third-party app stores',
    paragraphs: [
      'If you download the app from the Apple App Store or Google Play, your use is also subject to those stores’ terms. Apple’s standard Licensed Application End User Licence Agreement applies as a fallback where this EULA is silent, and where there is a conflict for an App Store download, Apple’s terms govern the relevant point. The app stores are not responsible for the app or for support, which Fe3dr provides.',
    ],
  },
  {
    heading: '5. Updates and termination',
    paragraphs: [
      'We may release updates, upgrades, and patches, which this EULA covers unless they come with separate terms. Some updates may be required for the app to keep working. This licence continues until terminated; it ends automatically if you breach this EULA, and we may suspend or revoke it where required by law or to protect the platform. On termination you must stop using and delete the app.',
    ],
  },
  {
    heading: '6. Disclaimer and liability',
    paragraphs: [
      'The app is provided "as is" and "as available". To the extent permitted by law, we do not warrant that it will be uninterrupted or error-free, and our total liability in connection with the app is limited as set out in our Terms of Service. Nothing here limits any right that cannot be excluded under applicable law.',
    ],
  },
  {
    heading: '7. Governing law and contact',
    paragraphs: [
      'This EULA is read alongside our Terms of Service and Privacy Policy, and relates to a platform operated in India. For any question about this EULA, contact support@fe3dr.com.',
    ],
  },
];

export default function CustomerEulaScreen() {
  return (
    <LegalScreen
      title="End User Licence"
      lastUpdated={LAST_UPDATED}
      intro={INTRO}
      sections={SECTIONS}
    />
  );
}
