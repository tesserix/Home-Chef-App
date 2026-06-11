'use client';

import { useEffect, useRef } from 'react';

interface ParallaxProps {
  children: React.ReactNode;
  /**
   * Parallax intensity. Positive values drift the element *against*
   * scroll (it lags, feels further away); negative values lead it.
   * Keep within ±0.2 for subtlety.
   */
  speed?: number;
  className?: string;
}

/**
 * Subtle scroll parallax via translate3d on an inner element.
 * The OUTER element is measured (so the transform never feeds back
 * into the measurement) and the INNER element moves — transform-only,
 * rAF-throttled, fully disabled for prefers-reduced-motion.
 */
export function Parallax({ children, speed = 0.12, className }: ParallaxProps) {
  const outerRef = useRef<HTMLDivElement>(null);
  const innerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const outer = outerRef.current;
    const inner = innerRef.current;
    if (!outer || !inner) return;

    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      return;
    }

    let raf = 0;

    const update = () => {
      raf = 0;
      const rect = outer.getBoundingClientRect();
      const offsetFromCenter =
        rect.top + rect.height / 2 - window.innerHeight / 2;
      const shift = (offsetFromCenter * -speed).toFixed(1);
      inner.style.transform = `translate3d(0, ${shift}px, 0)`;
    };

    const schedule = () => {
      if (!raf) raf = requestAnimationFrame(update);
    };

    update();
    window.addEventListener('scroll', schedule, { passive: true });
    window.addEventListener('resize', schedule, { passive: true });

    return () => {
      window.removeEventListener('scroll', schedule);
      window.removeEventListener('resize', schedule);
      if (raf) cancelAnimationFrame(raf);
    };
  }, [speed]);

  return (
    <div ref={outerRef} className={className}>
      <div ref={innerRef} style={{ willChange: 'transform' }}>
        {children}
      </div>
    </div>
  );
}
