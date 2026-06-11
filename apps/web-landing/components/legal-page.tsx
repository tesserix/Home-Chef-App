import { CONTACT_EMAIL } from '@/lib/site';
import { SiteFooter } from '@/components/site-footer';
import { SiteNav } from '@/components/site-nav';

interface LegalPageProps {
  title: string;
  /**
   * Short interim summary shown until the full legal text lands.
   * TODO(owner): replace these stubs with the real, lawyer-reviewed
   * policy content before public launch.
   */
  summary: string;
}

export function LegalPage({ title, summary }: LegalPageProps) {
  return (
    <>
      <SiteNav />
      <main id="main">
        <article className="mx-auto max-w-3xl px-5 py-16 sm:px-8 lg:py-24">
          <h1 className="font-display text-[clamp(1.875rem,4vw,2.5rem)] font-bold tracking-tight text-charcoal">
            {title}
          </h1>
          <p className="mt-6 leading-relaxed text-charcoal-soft">{summary}</p>
          <p className="mt-4 leading-relaxed text-charcoal-soft">
            The full {title.toLowerCase()} is being finalised and will be
            published here before launch. Questions in the meantime? Write to{' '}
            {/* Charcoal, not coral: small text needs 4.5:1 AA contrast. */}
            <a
              href={`mailto:${CONTACT_EMAIL}`}
              className="font-medium text-charcoal underline underline-offset-4 transition-colors duration-micro ease-state hover:text-charcoal-soft"
            >
              {CONTACT_EMAIL}
            </a>
            .
          </p>
        </article>
      </main>
      <SiteFooter />
    </>
  );
}
