/**
 * Framer Motion variants for Home Chef.
 *
 * The canonical easing curve is `ease-out-quart` — cubic-bezier(0.22, 1, 0.36, 1).
 * No bounce, no elastic, no overshoot. Source of truth: /.impeccable.md.
 *
 * All entrance distances are intentionally short (8–16px). Exit durations are
 * 75% of entrance per the motion guide. Layout-affecting properties (height,
 * top, left) are avoided where possible — use `transform` and `opacity`.
 *
 * For `prefers-reduced-motion` users, wrap your app root in
 *   <MotionConfig reducedMotion="user"> — framer-motion will then strip
 *   non-essential motion automatically across every motion.* call.
 */

const EASE_OUT_QUART = [0.22, 1, 0.36, 1] as const;
const DURATION = {
  fast: 0.15,
  default: 0.25,
  slow: 0.4,
} as const;
const EXIT = {
  fast: 0.12,    // 75% of fast
  default: 0.18, // 75% of default
  slow: 0.3,     // 75% of slow
} as const;

const baseTransition = {
  duration: DURATION.default,
  ease: EASE_OUT_QUART,
} as const;

const exitTransition = {
  duration: EXIT.default,
  ease: EASE_OUT_QUART,
} as const;

// ---- Fade ----------------------------------------------------
export const fadeIn = {
  hidden: { opacity: 0 },
  visible: { opacity: 1, transition: baseTransition },
  exit: { opacity: 0, transition: exitTransition },
};

export const fadeInUp = {
  hidden: { opacity: 0, y: 12 },
  visible: { opacity: 1, y: 0, transition: baseTransition },
  exit: { opacity: 0, y: -8, transition: exitTransition },
};

export const fadeInDown = {
  hidden: { opacity: 0, y: -12 },
  visible: { opacity: 1, y: 0, transition: baseTransition },
  exit: { opacity: 0, y: 8, transition: exitTransition },
};

export const fadeInLeft = {
  hidden: { opacity: 0, x: -12 },
  visible: { opacity: 1, x: 0, transition: baseTransition },
  exit: { opacity: 0, x: 12, transition: exitTransition },
};

export const fadeInRight = {
  hidden: { opacity: 0, x: 12 },
  visible: { opacity: 1, x: 0, transition: baseTransition },
  exit: { opacity: 0, x: -12, transition: exitTransition },
};

// ---- Scale ---------------------------------------------------
export const scaleIn = {
  hidden: { opacity: 0, scale: 0.96 },
  visible: { opacity: 1, scale: 1, transition: baseTransition },
  exit: { opacity: 0, scale: 0.96, transition: exitTransition },
};

// ---- Staggered children -------------------------------------
export const staggerContainer = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.08, delayChildren: 0.05 },
  },
};

export const staggerContainerFast = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.04 },
  },
};

export const staggerContainerSlow = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.12 },
  },
};

// ---- Sheets / drawers (intentional 100% offset) -------------
export const slideUp = {
  hidden: { y: '100%' },
  visible: { y: 0, transition: { duration: DURATION.slow, ease: EASE_OUT_QUART } },
  exit: { y: '100%', transition: { duration: EXIT.slow, ease: EASE_OUT_QUART } },
};

export const slideDown = {
  hidden: { y: '-100%' },
  visible: { y: 0, transition: { duration: DURATION.slow, ease: EASE_OUT_QUART } },
  exit: { y: '-100%', transition: { duration: EXIT.slow, ease: EASE_OUT_QUART } },
};

export const slideLeft = {
  hidden: { x: '100%' },
  visible: { x: 0, transition: { duration: DURATION.slow, ease: EASE_OUT_QUART } },
  exit: { x: '100%', transition: { duration: EXIT.slow, ease: EASE_OUT_QUART } },
};

export const slideRight = {
  hidden: { x: '-100%' },
  visible: { x: 0, transition: { duration: DURATION.slow, ease: EASE_OUT_QUART } },
  exit: { x: '-100%', transition: { duration: EXIT.slow, ease: EASE_OUT_QUART } },
};

// ---- Accordion / expand -------------------------------------
// NOTE: animating height is not GPU-accelerated. Prefer the CSS
// `grid-template-rows: 0fr → 1fr` pattern for accordions where possible.
// Kept here for compatibility with existing callers — use sparingly.
export const expand = {
  hidden: { height: 0, opacity: 0 },
  visible: {
    height: 'auto',
    opacity: 1,
    transition: { duration: DURATION.default, ease: EASE_OUT_QUART },
  },
  exit: {
    height: 0,
    opacity: 0,
    transition: { duration: EXIT.default, ease: EASE_OUT_QUART },
  },
};

// ---- Page-level transitions ---------------------------------
export const pageTransition = {
  hidden: { opacity: 0, y: 8 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: DURATION.default, ease: EASE_OUT_QUART },
  },
  exit: {
    opacity: 0,
    y: -8,
    transition: { duration: EXIT.default, ease: EASE_OUT_QUART },
  },
};

// ---- Modal overlay + content -------------------------------
export const modalOverlay = {
  hidden: { opacity: 0 },
  visible: { opacity: 1, transition: { duration: DURATION.fast, ease: EASE_OUT_QUART } },
  exit: { opacity: 0, transition: { duration: EXIT.fast, ease: EASE_OUT_QUART } },
};

export const modalContent = {
  hidden: { opacity: 0, scale: 0.96, y: 8 },
  visible: {
    opacity: 1,
    scale: 1,
    y: 0,
    transition: { duration: DURATION.default, ease: EASE_OUT_QUART },
  },
  exit: {
    opacity: 0,
    scale: 0.96,
    y: 8,
    transition: { duration: EXIT.default, ease: EASE_OUT_QUART },
  },
};

// ---- List items --------------------------------------------
export const listItem = {
  hidden: { opacity: 0, x: -8 },
  visible: { opacity: 1, x: 0, transition: baseTransition },
};

// ---- Button press ------------------------------------------
// Intentionally tiny — feels like a press, not a bounce.
export const buttonTap = {
  scale: 0.98,
};

// ---- Toast notification (slide up + fade, no bounce) -------
export const toast = {
  hidden: { opacity: 0, y: 16 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: DURATION.default, ease: EASE_OUT_QUART },
  },
  exit: {
    opacity: 0,
    y: 16,
    transition: { duration: EXIT.default, ease: EASE_OUT_QUART },
  },
};

// ---- Skeleton shimmer (decorative, paused under reduced motion) --
export const shimmer = {
  hidden: { x: '-100%' },
  visible: {
    x: '100%',
    transition: { duration: 1.4, ease: 'linear', repeat: Infinity },
  },
};

// ---- Counter (smooth scale, no overshoot) ------------------
export const counter = {
  hidden: { opacity: 0, scale: 0.95 },
  visible: {
    opacity: 1,
    scale: 1,
    transition: { duration: DURATION.default, ease: EASE_OUT_QUART },
  },
};

// ---- Transition presets ------------------------------------
export const transitions = {
  default: { duration: DURATION.default, ease: EASE_OUT_QUART },
  fast: { duration: DURATION.fast, ease: EASE_OUT_QUART },
  slow: { duration: DURATION.slow, ease: EASE_OUT_QUART },
  // No spring presets — every shared animation uses tween + ease-out-quart.
  // If a callsite needs spring physics, use { type: 'spring', stiffness: 200, damping: 30 }
  // explicitly (damping ≥ 25 prevents overshoot).
} as const;

// ---- Viewport triggers (scroll-into-view animations) -------
export const viewportOnce = {
  once: true,
  margin: '-100px',
};

export const viewportAlways = {
  once: false,
  margin: '-50px',
};

// ---- Easing constants (exported for inline use) -----------
export const EASING = {
  smooth: EASE_OUT_QUART,
  // ease-in-out for state changes (less common)
  state: [0.4, 0, 0.2, 1] as const,
} as const;

export const DURATIONS = DURATION;
