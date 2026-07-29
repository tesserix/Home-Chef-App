import { ExternalLink, FileText, Scale, Shield } from 'lucide-react';
import { Card } from '@/shared/components/ui/Card';

// Legal hub — the web counterpart of apps/mobile-vendor/app/legal.tsx.
//
// The mobile app bundles each document as its own in-app screen because a phone
// browser hand-off is a poor experience mid-onboarding. On web the customer-
// facing site already publishes the canonical versions, so this links out rather
// than forking the text: two copies of a licence agreement WILL drift, and the
// one a chef agreed to needs to be the one that is actually in force.
//
// The EULA is deliberately absent — it is an APP STORE licence for the mobile
// binary and means nothing for a browser session.

const MARKETING_ORIGIN = 'https://fe3dr.com';

const DOCS = [
  {
    href: `${MARKETING_ORIGIN}/privacy`,
    title: 'Privacy policy',
    caption: 'How we handle your data',
    Icon: Shield,
  },
  {
    href: `${MARKETING_ORIGIN}/terms`,
    title: 'Terms of service',
    caption: 'Using the chef portal',
    Icon: FileText,
  },
  {
    href: `${MARKETING_ORIGIN}/chef-agreement`,
    title: 'Chef agreement',
    caption: 'Commission, payouts, food safety',
    Icon: Scale,
  },
];

export function LegalPage() {
  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="text-2xl font-semibold tracking-tight text-foreground">Legal</h1>
      <p className="mt-1 text-sm text-ink-soft">
        The documents that govern your kitchen on Fe3dr. These open the published versions, which
        are always the ones in force.
      </p>

      <div className="mt-6 flex flex-col gap-3">
        {DOCS.map((d) => {
          const Icon = d.Icon;
          return (
            <a
              key={d.href}
              href={d.href}
              target="_blank"
              rel="noopener noreferrer"
              className="block"
            >
              <Card className="flex items-center gap-4 p-4 transition-colors hover:bg-paper">
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-mist">
                  <Icon className="h-5 w-5 text-ink-soft" aria-hidden="true" />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block font-medium text-foreground">{d.title}</span>
                  <span className="block text-sm text-ink-soft">{d.caption}</span>
                </span>
                <ExternalLink className="h-4 w-4 shrink-0 text-ink-muted" aria-hidden="true" />
              </Card>
            </a>
          );
        })}
      </div>
    </div>
  );
}

export default LegalPage;
