// Template legal content — have counsel review before launch (see COUNSEL-REVIEW.md).

import { CONTACT_EMAIL, LEGAL_OPERATOR, LEGAL_SUPPORT_EMAIL } from '@/lib/site';
import { SiteFooter } from '@/components/site-footer';
import { SiteNav } from '@/components/site-nav';

export interface LegalSection {
  heading: string;
  /** Each entry is a paragraph. Prefix with "• " to render as a bullet line. */
  paragraphs: string[];
}

interface LegalPageProps {
  title: string;
  /**
   * Lead paragraph shown under the title. When `sections` is omitted this is
   * the only body content (interim stub mode); when `sections` is present it
   * reads as the policy's plain-language summary.
   */
  summary: string;
  /** Date string for the "Last updated" line. Required for published bodies. */
  lastUpdated?: string;
  /** Full structured policy body. When present, the stub placeholder is dropped. */
  sections?: LegalSection[];
}

export function LegalPage({ title, summary, lastUpdated, sections }: LegalPageProps) {
  const hasBody = Array.isArray(sections) && sections.length > 0;

  return (
    <>
      <SiteNav />
      <main id="main">
        <article className="mx-auto max-w-3xl px-5 py-16 sm:px-8 lg:py-24">
          <h1 className="font-display text-[clamp(1.875rem,4vw,2.5rem)] font-bold tracking-tight text-charcoal">
            {title}
          </h1>

          {hasBody ? (
            <>
              <p className="mt-4 text-sm text-charcoal-soft">
                Fe3dr is a product of {LEGAL_OPERATOR}.
                {lastUpdated ? ` Last updated: ${lastUpdated}.` : ''}
              </p>
              <p className="mt-6 leading-relaxed text-charcoal-soft">{summary}</p>

              <div className="mt-10 space-y-10">
                {sections.map((section, i) => (
                  <section key={`${section.heading}-${i}`}>
                    <h2 className="font-display text-xl font-semibold tracking-tight text-charcoal">
                      {section.heading}
                    </h2>
                    <LegalSectionBody paragraphs={section.paragraphs} />
                  </section>
                ))}
              </div>

              <p className="mt-12 border-t border-charcoal/10 pt-6 text-sm leading-relaxed text-charcoal-soft">
                Questions about this {title.toLowerCase()}? Write to{' '}
                <a
                  href={`mailto:${LEGAL_SUPPORT_EMAIL}`}
                  className="font-medium text-charcoal underline underline-offset-4 transition-colors duration-micro ease-state hover:text-charcoal-soft"
                >
                  {LEGAL_SUPPORT_EMAIL}
                </a>
                .
              </p>
            </>
          ) : (
            <>
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
            </>
          )}
        </article>
      </main>
      <SiteFooter />
    </>
  );
}

/**
 * Renders a section's paragraphs, grouping consecutive "• "-prefixed lines into
 * a single <ul>. Plain paragraphs render as <p>.
 */
function LegalSectionBody({ paragraphs }: { paragraphs: string[] }) {
  const blocks: Array<{ type: 'p'; text: string } | { type: 'ul'; items: string[] }> = [];

  for (const para of paragraphs) {
    if (para.startsWith('• ')) {
      const item = para.slice(2);
      const last = blocks[blocks.length - 1];
      if (last && last.type === 'ul') {
        last.items.push(item);
      } else {
        blocks.push({ type: 'ul', items: [item] });
      }
    } else {
      blocks.push({ type: 'p', text: para });
    }
  }

  return (
    <div className="mt-3 space-y-3">
      {blocks.map((block, i) =>
        block.type === 'ul' ? (
          <ul
            key={i}
            className="ml-5 list-disc space-y-1.5 leading-relaxed text-charcoal-soft marker:text-charcoal-soft"
          >
            {block.items.map((item, j) => (
              <li key={j}>{item}</li>
            ))}
          </ul>
        ) : (
          <p key={i} className="leading-relaxed text-charcoal-soft">
            {block.text}
          </p>
        ),
      )}
    </div>
  );
}
