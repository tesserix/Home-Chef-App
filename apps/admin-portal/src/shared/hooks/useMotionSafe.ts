import { useReducedMotion } from 'framer-motion';

/**
 * Returns `true` when motion is OK to play, `false` when the user has set
 * `prefers-reduced-motion: reduce`.
 *
 * Use this in components that orchestrate motion outside framer-motion's
 * variant system — e.g., setInterval-driven counters, IntersectionObserver
 * scroll choreography, manual classList toggling.
 *
 * For framer-motion `motion.*` components, the root `<MotionConfig
 * reducedMotion="user">` already handles this — you don't need to check here.
 *
 * @example
 * const canAnimate = useMotionSafe();
 * useEffect(() => {
 *   if (!canAnimate) return;
 *   const id = setInterval(tickCounter, 16);
 *   return () => clearInterval(id);
 * }, [canAnimate]);
 */
export function useMotionSafe(): boolean {
  const shouldReduce = useReducedMotion();
  return !shouldReduce;
}
