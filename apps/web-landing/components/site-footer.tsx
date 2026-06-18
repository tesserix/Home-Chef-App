import { CONTACT_EMAIL, INSTAGRAM_URL, X_URL } from '@/lib/site';
import { StoreBadges } from '@/components/store-badges';
import { Wordmark } from '@/components/wordmark';

const EXPLORE_LINKS = [
  { href: '#how-it-works', label: 'How it works' },
  { href: '#whats-cooking', label: "What's cooking" },
  { href: '#for-chefs', label: 'For chefs' },
  { href: '#get-the-app', label: 'Get the app' },
] as const;

const LEGAL_LINKS = [
  { href: '/privacy/', label: 'Privacy Policy' },
  { href: '/terms/', label: 'Terms of Service' },
  { href: '/refund/', label: 'Refund & Cancellation' },
] as const;

export function SiteFooter() {
  return (
    <footer className="border-t border-hairline">
      <div className="mx-auto max-w-6xl px-5 py-14 sm:px-8 lg:py-16">
        <div className="grid gap-10 lg:grid-cols-12">
          <div className="lg:col-span-5">
            <Wordmark />
            <p className="mt-4 max-w-xs text-sm leading-relaxed text-charcoal-soft">
              Real home-cooked food from verified kitchens near you, delivered
              to your door.
            </p>
            <StoreBadges height={40} className="mt-6" />
          </div>

          <nav aria-label="Explore" className="lg:col-span-2 lg:col-start-7">
            <h2 className="text-sm font-semibold text-charcoal">Explore</h2>
            <ul className="mt-4 space-y-3">
              {EXPLORE_LINKS.map((link) => (
                <li key={link.href}>
                  <a
                    href={link.href}
                    className="text-sm text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal"
                  >
                    {link.label}
                  </a>
                </li>
              ))}
            </ul>
          </nav>

          <nav aria-label="Legal" className="lg:col-span-2">
            <h2 className="text-sm font-semibold text-charcoal">Legal</h2>
            <ul className="mt-4 space-y-3">
              {LEGAL_LINKS.map((link) => (
                <li key={link.href}>
                  <a
                    href={link.href}
                    className="text-sm text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal"
                  >
                    {link.label}
                  </a>
                </li>
              ))}
            </ul>
          </nav>

          <div className="lg:col-span-3">
            <h2 className="text-sm font-semibold text-charcoal">Contact</h2>
            <ul className="mt-4 space-y-3">
              <li>
                <a
                  href={`mailto:${CONTACT_EMAIL}`}
                  className="text-sm text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal"
                >
                  {CONTACT_EMAIL}
                </a>
              </li>
              <li className="flex gap-4 pt-1">
                {/* TODO(owner): real social profiles (lib/site.ts). */}
                <a
                  href={INSTAGRAM_URL}
                  aria-label="Fe3dr on Instagram"
                  className="text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal"
                >
                  <InstagramIcon />
                </a>
                <a
                  href={X_URL}
                  aria-label="Fe3dr on X"
                  className="text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal"
                >
                  <XIcon />
                </a>
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-12 flex flex-col gap-2 border-t border-hairline pt-6 text-sm text-charcoal-soft sm:flex-row sm:items-center sm:justify-between">
          <p>&copy; {new Date().getFullYear()} Fe3dr &middot; fe3dr.com</p>
          <p>Made with care, like the food.</p>
        </div>
      </div>
    </footer>
  );
}

function InstagramIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <rect x="3" y="3" width="18" height="18" rx="5" stroke="currentColor" strokeWidth="1.6" />
      <circle cx="12" cy="12" r="4" stroke="currentColor" strokeWidth="1.6" />
      <circle cx="17.2" cy="6.8" r="1.1" fill="currentColor" />
    </svg>
  );
}

function XIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M17.9 3h3l-6.6 7.6L22 21h-6.1l-4.8-6.3L5.6 21h-3l7.1-8.1L2 3h6.3l4.3 5.7L17.9 3Zm-1.1 16.2h1.7L7.4 4.7H5.6l11.2 14.5Z" />
    </svg>
  );
}
