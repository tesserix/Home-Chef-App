'use client';

import { useEffect, useState } from 'react';
import { cn } from '@tesserix/web';
import { Wordmark } from '@/components/wordmark';

/**
 * Sticky top navigation. Transparent-on-white at rest; a hairline and
 * faint shadow appear once the page scrolls.
 */
export function SiteNav() {
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
          aria-label="Home Chef — home"
          className="flex items-center gap-2.5 rounded"
        >
          <Wordmark />
        </a>

        <div className="flex items-center gap-2 sm:gap-7">
          <a
            href="#how-it-works"
            className="hidden text-[15px] font-medium text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal sm:block"
          >
            How it works
          </a>
          <a
            href="#whats-cooking"
            className="hidden text-[15px] font-medium text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal md:block"
          >
            What&rsquo;s cooking
          </a>
          <a
            href="#for-chefs"
            className="hidden text-[15px] font-medium text-charcoal-soft transition-colors duration-micro ease-state hover:text-charcoal sm:block"
          >
            For chefs
          </a>
          <a
            href="#get-the-app"
            className="inline-flex h-11 items-center rounded-full bg-coral px-5 text-[15px] font-semibold text-white transition-colors duration-micro ease-state hover:bg-coral-pressed"
          >
            Get the app
          </a>
        </div>
      </nav>
    </header>
  );
}
