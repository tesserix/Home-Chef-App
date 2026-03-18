/**
 * Framer Motion animation variants for consistent animations across the app
 */

// Fade animations
export const fadeIn = {
  hidden: { opacity: 0 },
  visible: { opacity: 1 },
};

export const fadeInUp = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0 },
};

export const fadeInDown = {
  hidden: { opacity: 0, y: -20 },
  visible: { opacity: 1, y: 0 },
};

export const fadeInLeft = {
  hidden: { opacity: 0, x: -20 },
  visible: { opacity: 1, x: 0 },
};

export const fadeInRight = {
  hidden: { opacity: 0, x: 20 },
  visible: { opacity: 1, x: 0 },
};

// Scale animations
export const scaleIn = {
  hidden: { opacity: 0, scale: 0.9 },
  visible: { opacity: 1, scale: 1 },
};

export const scaleInBounce = {
  hidden: { opacity: 0, scale: 0.8 },
  visible: {
    opacity: 1,
    scale: 1,
    transition: {
      type: 'spring',
      stiffness: 300,
      damping: 20,
    },
  },
};

// Container for staggered children
export const staggerContainer = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
    },
  },
};

export const staggerContainerFast = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.05,
    },
  },
};

export const staggerContainerSlow = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.15,
    },
  },
};

// Slide animations
export const slideUp = {
  hidden: { y: '100%' },
  visible: { y: 0 },
  exit: { y: '100%' },
};

export const slideDown = {
  hidden: { y: '-100%' },
  visible: { y: 0 },
  exit: { y: '-100%' },
};

export const slideLeft = {
  hidden: { x: '100%' },
  visible: { x: 0 },
  exit: { x: '100%' },
};

export const slideRight = {
  hidden: { x: '-100%' },
  visible: { x: 0 },
  exit: { x: '-100%' },
};

// Expand/collapse
export const expand = {
  hidden: { height: 0, opacity: 0 },
  visible: { height: 'auto', opacity: 1 },
  exit: { height: 0, opacity: 0 },
};

// Page transitions
export const pageTransition = {
  hidden: { opacity: 0, y: 10 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.3,
      ease: [0.16, 1, 0.3, 1], // Premium easing
    },
  },
  exit: {
    opacity: 0,
    y: -10,
    transition: {
      duration: 0.2,
    },
  },
};

// Modal animations
export const modalOverlay = {
  hidden: { opacity: 0 },
  visible: { opacity: 1 },
  exit: { opacity: 0 },
};

export const modalContent = {
  hidden: { opacity: 0, scale: 0.95, y: 10 },
  visible: {
    opacity: 1,
    scale: 1,
    y: 0,
    transition: {
      duration: 0.2,
      ease: [0.16, 1, 0.3, 1],
    },
  },
  exit: {
    opacity: 0,
    scale: 0.95,
    y: 10,
    transition: {
      duration: 0.15,
    },
  },
};

// List item animations
export const listItem = {
  hidden: { opacity: 0, x: -10 },
  visible: { opacity: 1, x: 0 },
};

// Card animations
export const cardHover = {
  rest: {
    y: 0,
    boxShadow: '0 1px 3px 0 rgb(0 0 0 / 0.04), 0 1px 2px -1px rgb(0 0 0 / 0.02)',
  },
  hover: {
    y: -4,
    boxShadow: '0 8px 24px -4px rgb(0 0 0 / 0.1), 0 4px 8px -2px rgb(0 0 0 / 0.04)',
    transition: {
      duration: 0.2,
      ease: [0.16, 1, 0.3, 1],
    },
  },
};

// Button press animation
export const buttonTap = {
  scale: 0.98,
};

// Notification/toast animations
export const toast = {
  hidden: { opacity: 0, y: 50, scale: 0.9 },
  visible: {
    opacity: 1,
    y: 0,
    scale: 1,
    transition: {
      type: 'spring',
      stiffness: 400,
      damping: 25,
    },
  },
  exit: {
    opacity: 0,
    y: 20,
    scale: 0.9,
    transition: {
      duration: 0.2,
    },
  },
};

// Skeleton shimmer animation
export const shimmer = {
  hidden: { x: '-100%' },
  visible: {
    x: '100%',
    transition: {
      duration: 1.5,
      ease: 'linear',
      repeat: Infinity,
    },
  },
};

// Counter/number animation
export const counter = {
  hidden: { opacity: 0, scale: 0.8 },
  visible: {
    opacity: 1,
    scale: [0.8, 1.1, 1],
    transition: {
      duration: 0.3,
    },
  },
};

// Transition configs
export const transitions = {
  default: {
    duration: 0.2,
    ease: [0.16, 1, 0.3, 1],
  },
  fast: {
    duration: 0.15,
    ease: [0.16, 1, 0.3, 1],
  },
  slow: {
    duration: 0.4,
    ease: [0.16, 1, 0.3, 1],
  },
  spring: {
    type: 'spring',
    stiffness: 300,
    damping: 20,
  },
  springBouncy: {
    type: 'spring',
    stiffness: 400,
    damping: 15,
  },
};

// Viewport trigger options
export const viewportOnce = {
  once: true,
  margin: '-100px',
};

export const viewportAlways = {
  once: false,
  margin: '-50px',
};
