'use client';

import { useEffect, useState } from 'react';
import { cn } from '@tesserix/web';
import { Wordmark } from '@/components/wordmark';
import { APP_LOGIN_PATH, BROWSE_PATH, WEB_APP_LIVE } from '@/lib/site';

interface SiteNavProps {
  /**
   * Where the primary CTA goes. Defaults to the home page's badge block;
   * pages that aren't the home page pass a real destination so the CTA is
   * never a dangling anchor.
   */
  ctaHref?: string;
  ctaLabel?: string;
}

/**
 * Sticky top navigation. Transparent-on-white at rest; a hairline and
 * faint shadow appear once the page scrolls.
 */
export function SiteNav({
  // The CTA points at ORDERING, not at the app store. While both listings are
  // still "coming soon", a nav button labelled "Get the app" sends the one
  // visitor who is ready to buy to a page that cannot sell them anything.
  // Browse is guest-accessible, so this is a real destination for a first-time
  // visitor, not a sign-in wall.
  ctaHref = WEB_APP_LIVE ? BROWSE_PATH : '/download/',
  ctaLabel = WEB_APP_LIVE ? 'Order now' : 'Get the app',
}: SiteNavProps = {}) {
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8);
    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  return (
    <header
      className={cn(
        'sticky top-0 z-50 bg-canvas transition-shadow duration-250 ease-state',
        scrolled ? 'border-b border-hairline shadow-1' : 'border-b border-transparent'
      )}
    >
      <nav
        aria-label="Main"
        className="mx-auto flex h-[72px] max-w-6xl items-center justify-between px-5 sm:px-8"
      >
        <a
          href="/"
          aria-label="Fe3dr — home"
          className="flex items-center gap-2.5 rounded"
        >
          <Wordmark />
        </a>

        <div className="flex items-center gap-2 sm:gap-7">
          <a
            href="/#how-it-works"
            className="hidden text-[15px] font-medium text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal sm:block"
          >
            How it works
          </a>
          <a
            href="/#whats-cooking"
            className="hidden text-[15px] font-medium text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal md:block"
          >
            What&rsquo;s cooking
          </a>
          <a
            href="/#for-chefs"
            className="hidden text-[15px] font-medium text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal sm:block"
          >
            For chefs
          </a>
          {WEB_APP_LIVE ? (
            <a
              href={APP_LOGIN_PATH}
              className="text-[15px] font-medium text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal"
            >
              Log in
            </a>
          ) : null}
          <a
            href={ctaHref}
            className="inline-flex h-11 items-center rounded-full bg-coral px-5 text-[15px] font-semibold text-white transition-colors duration-micro ease-state hover:bg-coral-pressed"
          >
            {ctaLabel}
          </a>
        </div>
      </nav>
    </header>
  );
}
