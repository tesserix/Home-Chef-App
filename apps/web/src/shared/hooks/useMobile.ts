import { useState, useEffect, useCallback } from 'react';

// Breakpoints matching Tailwind CSS
const BREAKPOINTS = {
  sm: 640,
  md: 768,
  lg: 1024,
  xl: 1280,
  '2xl': 1536,
} as const;

type Breakpoint = keyof typeof BREAKPOINTS;

/**
 * Hook to detect if the current viewport is mobile
 */
export function useIsMobile(breakpoint: Breakpoint = 'md'): boolean {
  const [isMobile, setIsMobile] = useState(() => {
    if (typeof window === 'undefined') return true;
    return window.innerWidth < BREAKPOINTS[breakpoint];
  });

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < BREAKPOINTS[breakpoint]);
    };

    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, [breakpoint]);

  return isMobile;
}

/**
 * Hook to get the current breakpoint
 */
export function useBreakpoint(): Breakpoint | 'xs' {
  const [breakpoint, setBreakpoint] = useState<Breakpoint | 'xs'>(() => {
    if (typeof window === 'undefined') return 'xs';
    return getBreakpoint(window.innerWidth);
  });

  useEffect(() => {
    const handleResize = () => {
      setBreakpoint(getBreakpoint(window.innerWidth));
    };

    handleResize();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  return breakpoint;
}

function getBreakpoint(width: number): Breakpoint | 'xs' {
  if (width >= BREAKPOINTS['2xl']) return '2xl';
  if (width >= BREAKPOINTS.xl) return 'xl';
  if (width >= BREAKPOINTS.lg) return 'lg';
  if (width >= BREAKPOINTS.md) return 'md';
  if (width >= BREAKPOINTS.sm) return 'sm';
  return 'xs';
}

/**
 * Hook to detect touch device
 */
export function useIsTouchDevice(): boolean {
  const [isTouch, setIsTouch] = useState(false);

  useEffect(() => {
    setIsTouch(
      'ontouchstart' in window ||
      navigator.maxTouchPoints > 0 ||
      // @ts-expect-error - msMaxTouchPoints for older IE
      navigator.msMaxTouchPoints > 0
    );
  }, []);

  return isTouch;
}

/**
 * Hook to detect if app is running as PWA
 */
export function useIsPWA(): boolean {
  const [isPWA, setIsPWA] = useState(false);

  useEffect(() => {
    const isStandalone = window.matchMedia('(display-mode: standalone)').matches;
    // @ts-expect-error - navigator.standalone for iOS
    const isIOSStandalone = window.navigator.standalone === true;
    setIsPWA(isStandalone || isIOSStandalone);
  }, []);

  return isPWA;
}

/**
 * Hook to detect device orientation
 */
export function useOrientation(): 'portrait' | 'landscape' {
  const [orientation, setOrientation] = useState<'portrait' | 'landscape'>(() => {
    if (typeof window === 'undefined') return 'portrait';
    return window.innerHeight > window.innerWidth ? 'portrait' : 'landscape';
  });

  useEffect(() => {
    const handleOrientationChange = () => {
      setOrientation(window.innerHeight > window.innerWidth ? 'portrait' : 'landscape');
    };

    window.addEventListener('resize', handleOrientationChange);
    window.addEventListener('orientationchange', handleOrientationChange);

    return () => {
      window.removeEventListener('resize', handleOrientationChange);
      window.removeEventListener('orientationchange', handleOrientationChange);
    };
  }, []);

  return orientation;
}

/**
 * Hook to detect safe area insets (for notched devices)
 */
export function useSafeAreaInsets() {
  const [insets, setInsets] = useState({
    top: 0,
    bottom: 0,
    left: 0,
    right: 0,
  });

  useEffect(() => {
    const updateInsets = () => {
      const computedStyle = getComputedStyle(document.documentElement);
      setInsets({
        top: parseInt(computedStyle.getPropertyValue('--sat') || '0', 10),
        bottom: parseInt(computedStyle.getPropertyValue('--sab') || '0', 10),
        left: parseInt(computedStyle.getPropertyValue('--sal') || '0', 10),
        right: parseInt(computedStyle.getPropertyValue('--sar') || '0', 10),
      });
    };

    // Set CSS variables for safe area insets
    document.documentElement.style.setProperty('--sat', 'env(safe-area-inset-top)');
    document.documentElement.style.setProperty('--sab', 'env(safe-area-inset-bottom)');
    document.documentElement.style.setProperty('--sal', 'env(safe-area-inset-left)');
    document.documentElement.style.setProperty('--sar', 'env(safe-area-inset-right)');

    updateInsets();
    window.addEventListener('resize', updateInsets);
    return () => window.removeEventListener('resize', updateInsets);
  }, []);

  return insets;
}

/**
 * Hook for haptic feedback (vibration)
 */
export function useHaptics() {
  const vibrate = useCallback((pattern: number | number[] = 10) => {
    if ('vibrate' in navigator) {
      navigator.vibrate(pattern);
    }
  }, []);

  const lightImpact = useCallback(() => vibrate(10), [vibrate]);
  const mediumImpact = useCallback(() => vibrate(20), [vibrate]);
  const heavyImpact = useCallback(() => vibrate(30), [vibrate]);
  const success = useCallback(() => vibrate([10, 50, 10]), [vibrate]);
  const warning = useCallback(() => vibrate([30, 50, 30]), [vibrate]);
  const error = useCallback(() => vibrate([50, 100, 50]), [vibrate]);

  return {
    vibrate,
    lightImpact,
    mediumImpact,
    heavyImpact,
    success,
    warning,
    error,
  };
}

/**
 * Hook to prevent body scroll (useful for modals on mobile)
 */
export function useLockBodyScroll(lock: boolean = true) {
  useEffect(() => {
    if (!lock) return;

    const originalStyle = window.getComputedStyle(document.body).overflow;
    const scrollY = window.scrollY;

    document.body.style.overflow = 'hidden';
    document.body.style.position = 'fixed';
    document.body.style.top = `-${scrollY}px`;
    document.body.style.width = '100%';

    return () => {
      document.body.style.overflow = originalStyle;
      document.body.style.position = '';
      document.body.style.top = '';
      document.body.style.width = '';
      window.scrollTo(0, scrollY);
    };
  }, [lock]);
}

/**
 * Hook to detect online/offline status
 */
export function useOnlineStatus(): boolean {
  const [isOnline, setIsOnline] = useState(() => {
    if (typeof navigator === 'undefined') return true;
    return navigator.onLine;
  });

  useEffect(() => {
    const handleOnline = () => setIsOnline(true);
    const handleOffline = () => setIsOnline(false);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  return isOnline;
}
