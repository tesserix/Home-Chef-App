'use client';

import { useEffect, useRef } from 'react';
import { cn } from '@tesserix/web';

type RevealVariant = 'up' | 'left' | 'right' | 'scale';

interface RevealProps {
  children: React.ReactNode;
  /** Stagger offset in ms (applied as transition-delay). */
  delay?: number;
  /** Entrance direction — all are opacity + transform only. */
  variant?: RevealVariant;
  className?: string;
}

const VARIANT_CLASS: Record<RevealVariant, string | undefined> = {
  up: undefined,
  left: 'reveal--left',
  right: 'reveal--right',
  scale: 'reveal--scale',
};

/**
 * Scroll-reveal wrapper. Adds `.is-visible` when the element enters the
 * viewport; the transition itself lives in globals.css (opacity +
 * transform only, ease-out-quart). Reduced-motion users — and no-JS
 * visitors via the CSS fallback — get content instantly.
 */
export function Reveal({
  children,
  delay = 0,
  variant = 'up',
  className,
}: RevealProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;

    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      node.classList.add('is-visible');
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[0];
        if (entry?.isIntersecting) {
          node.classList.add('is-visible');
          observer.disconnect();
        }
      },
      { threshold: 0.15, rootMargin: '0px 0px -40px 0px' }
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  return (
    <div
      ref={ref}
      className={cn('reveal', VARIANT_CLASS[variant], className)}
      style={delay > 0 ? { transitionDelay: `${delay}ms` } : undefined}
    >
      {children}
    </div>
  );
}
