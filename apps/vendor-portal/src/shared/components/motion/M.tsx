/**
 * <M> — Home Chef's motion primitive.
 *
 * Wraps `motion.<tag>` from framer-motion with our canonical defaults:
 *   - ease-out-quart (cubic-bezier(0.22, 1, 0.36, 1))
 *   - 250ms entrance / 180ms exit
 *   - prefers-reduced-motion respected automatically (via root MotionConfig)
 *
 * Use this in NEW code instead of writing `motion.div` directly. Existing
 * `motion.*` callsites still work (covered by the MotionConfig at the App root),
 * but <M> gives you a typed, consistent default surface.
 *
 * Examples:
 *
 *   <M.Div variants={fadeInUp} initial="hidden" animate="visible">…</M.Div>
 *
 *   <M.Section as="section" preset="fade-in-up">…</M.Section>
 *
 *   <M.Stagger>
 *     <M.Item>One</M.Item>
 *     <M.Item>Two</M.Item>
 *   </M.Stagger>
 */

import { motion, type HTMLMotionProps, type Variants } from 'framer-motion';
import { forwardRef } from 'react';
import {
  fadeIn,
  fadeInUp,
  fadeInDown,
  fadeInLeft,
  fadeInRight,
  scaleIn,
  staggerContainer,
  staggerContainerFast,
  staggerContainerSlow,
  listItem,
} from '@/shared/utils/animations';

export type MotionPreset =
  | 'fade'
  | 'fade-in-up'
  | 'fade-in-down'
  | 'fade-in-left'
  | 'fade-in-right'
  | 'scale-in'
  | 'stagger'
  | 'stagger-fast'
  | 'stagger-slow'
  | 'item';

const PRESETS: Record<MotionPreset, Variants> = {
  fade: fadeIn,
  'fade-in-up': fadeInUp,
  'fade-in-down': fadeInDown,
  'fade-in-left': fadeInLeft,
  'fade-in-right': fadeInRight,
  'scale-in': scaleIn,
  stagger: staggerContainer,
  'stagger-fast': staggerContainerFast,
  'stagger-slow': staggerContainerSlow,
  item: listItem,
};

interface MDivProps extends HTMLMotionProps<'div'> {
  /** Apply a predefined motion preset. Equivalent to setting variants + initial="hidden" + animate="visible". */
  preset?: MotionPreset;
}

function applyPreset(
  preset: MotionPreset | undefined,
  rest: Omit<MDivProps, 'preset'>
): HTMLMotionProps<'div'> {
  if (!preset) return rest;
  return {
    variants: PRESETS[preset],
    initial: rest.initial ?? 'hidden',
    animate: rest.animate ?? 'visible',
    exit: rest.exit ?? 'exit',
    ...rest,
  };
}

const Div = forwardRef<HTMLDivElement, MDivProps>(({ preset, ...rest }, ref) => (
  <motion.div ref={ref} {...applyPreset(preset, rest)} />
));
Div.displayName = 'M.Div';

const Section = forwardRef<HTMLElement, MDivProps>(({ preset, ...rest }, ref) => (
  <motion.section
    ref={ref as React.Ref<HTMLElement>}
    {...(applyPreset(preset, rest) as HTMLMotionProps<'section'>)}
  />
));
Section.displayName = 'M.Section';

const Span = forwardRef<HTMLSpanElement, HTMLMotionProps<'span'> & { preset?: MotionPreset }>(
  ({ preset, ...rest }, ref) => (
    <motion.span
      ref={ref}
      {...(applyPreset(preset, rest as Omit<MDivProps, 'preset'>) as HTMLMotionProps<'span'>)}
    />
  )
);
Span.displayName = 'M.Span';

const Li = forwardRef<HTMLLIElement, HTMLMotionProps<'li'> & { preset?: MotionPreset }>(
  ({ preset, ...rest }, ref) => (
    <motion.li
      ref={ref}
      {...(applyPreset(preset, rest as Omit<MDivProps, 'preset'>) as HTMLMotionProps<'li'>)}
    />
  )
);
Li.displayName = 'M.Li';

/** Convenience stagger container that defaults to staggering its children. */
const Stagger = forwardRef<HTMLDivElement, MDivProps>(({ preset = 'stagger', ...rest }, ref) => (
  <motion.div ref={ref} {...applyPreset(preset, rest)} />
));
Stagger.displayName = 'M.Stagger';

/** Convenience stagger child that uses the listItem variant. */
const Item = forwardRef<HTMLDivElement, MDivProps>(({ preset = 'item', ...rest }, ref) => (
  <motion.div ref={ref} {...applyPreset(preset, rest)} />
));
Item.displayName = 'M.Item';

/**
 * `<M>` is a namespace, not a component. Use `<M.Div>`, `<M.Stagger>`, etc.
 */
export const M = {
  Div,
  Section,
  Span,
  Li,
  Stagger,
  Item,
} as const;

export type MNamespace = typeof M;
