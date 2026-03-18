import React, { useRef, useCallback, useEffect } from 'react';

interface SwipeConfig {
  onSwipeLeft?: () => void;
  onSwipeRight?: () => void;
  onSwipeUp?: () => void;
  onSwipeDown?: () => void;
  threshold?: number;
  preventDefaultOnSwipe?: boolean;
}

interface SwipeState {
  startX: number;
  startY: number;
  endX: number;
  endY: number;
  swiping: boolean;
}

/**
 * Hook for handling swipe gestures
 */
export function useSwipe<T extends HTMLElement = HTMLElement>(config: SwipeConfig) {
  const {
    onSwipeLeft,
    onSwipeRight,
    onSwipeUp,
    onSwipeDown,
    threshold = 50,
    preventDefaultOnSwipe = false,
  } = config;

  const ref = useRef<T>(null);
  const state = useRef<SwipeState>({
    startX: 0,
    startY: 0,
    endX: 0,
    endY: 0,
    swiping: false,
  });

  const handleTouchStart = useCallback((e: TouchEvent) => {
    const touch = e.touches[0];
    if (!touch) return;
    state.current = {
      startX: touch.clientX,
      startY: touch.clientY,
      endX: touch.clientX,
      endY: touch.clientY,
      swiping: true,
    };
  }, []);

  const handleTouchMove = useCallback((e: TouchEvent) => {
    if (!state.current.swiping) return;
    const touch = e.touches[0];
    if (!touch) return;

    state.current.endX = touch.clientX;
    state.current.endY = touch.clientY;

    if (preventDefaultOnSwipe) {
      const diffX = Math.abs(state.current.endX - state.current.startX);
      const diffY = Math.abs(state.current.endY - state.current.startY);
      if (diffX > diffY && diffX > 10) {
        e.preventDefault();
      }
    }
  }, [preventDefaultOnSwipe]);

  const handleTouchEnd = useCallback(() => {
    if (!state.current.swiping) return;

    const { startX, startY, endX, endY } = state.current;
    const diffX = endX - startX;
    const diffY = endY - startY;
    const absDiffX = Math.abs(diffX);
    const absDiffY = Math.abs(diffY);

    // Determine if swipe was horizontal or vertical
    if (absDiffX > absDiffY && absDiffX > threshold) {
      if (diffX > 0) {
        onSwipeRight?.();
      } else {
        onSwipeLeft?.();
      }
    } else if (absDiffY > absDiffX && absDiffY > threshold) {
      if (diffY > 0) {
        onSwipeDown?.();
      } else {
        onSwipeUp?.();
      }
    }

    state.current.swiping = false;
  }, [onSwipeLeft, onSwipeRight, onSwipeUp, onSwipeDown, threshold]);

  useEffect(() => {
    const element = ref.current;
    if (!element) return;

    element.addEventListener('touchstart', handleTouchStart, { passive: true });
    element.addEventListener('touchmove', handleTouchMove, { passive: !preventDefaultOnSwipe });
    element.addEventListener('touchend', handleTouchEnd, { passive: true });

    return () => {
      element.removeEventListener('touchstart', handleTouchStart);
      element.removeEventListener('touchmove', handleTouchMove);
      element.removeEventListener('touchend', handleTouchEnd);
    };
  }, [handleTouchStart, handleTouchMove, handleTouchEnd, preventDefaultOnSwipe]);

  return ref;
}

/**
 * Hook for horizontal swipeable carousel/list
 */
export function useSwipeableList<T extends HTMLElement = HTMLElement>(
  itemCount: number,
  initialIndex = 0
) {
  const currentIndex = useRef(initialIndex);
  const containerRef = useRef<T>(null);

  const goTo = useCallback((index: number) => {
    const clampedIndex = Math.max(0, Math.min(index, itemCount - 1));
    currentIndex.current = clampedIndex;

    if (containerRef.current) {
      const itemWidth = containerRef.current.scrollWidth / itemCount;
      containerRef.current.scrollTo({
        left: itemWidth * clampedIndex,
        behavior: 'smooth',
      });
    }

    return clampedIndex;
  }, [itemCount]);

  const goNext = useCallback(() => {
    return goTo(currentIndex.current + 1);
  }, [goTo]);

  const goPrev = useCallback(() => {
    return goTo(currentIndex.current - 1);
  }, [goTo]);

  const swipeRef = useSwipe<T>({
    onSwipeLeft: goNext,
    onSwipeRight: goPrev,
    preventDefaultOnSwipe: true,
  });

  // Merge refs
  const mergedRef = useCallback((node: T | null) => {
    containerRef.current = node;
    (swipeRef as React.MutableRefObject<T | null>).current = node;
  }, [swipeRef]);

  return {
    ref: mergedRef,
    currentIndex: currentIndex.current,
    goTo,
    goNext,
    goPrev,
    isFirst: currentIndex.current === 0,
    isLast: currentIndex.current === itemCount - 1,
  };
}

/**
 * Hook for pull-to-refresh functionality
 */
export function usePullToRefresh(onRefresh: () => Promise<void>) {
  const containerRef = useRef<HTMLDivElement>(null);
  const startY = useRef(0);
  const pullDistance = useRef(0);
  const isRefreshing = useRef(false);

  const handleTouchStart = useCallback((e: TouchEvent) => {
    if (containerRef.current?.scrollTop === 0 && e.touches[0]) {
      startY.current = e.touches[0].clientY;
    }
  }, []);

  const handleTouchMove = useCallback((e: TouchEvent) => {
    if (startY.current === 0 || isRefreshing.current || !e.touches[0]) return;

    const currentY = e.touches[0].clientY;
    pullDistance.current = Math.max(0, currentY - startY.current);

    if (pullDistance.current > 0 && containerRef.current?.scrollTop === 0) {
      e.preventDefault();
      // Apply visual feedback (transform)
      const pullElement = containerRef.current?.querySelector('[data-pull-indicator]');
      if (pullElement instanceof HTMLElement) {
        const progress = Math.min(pullDistance.current / 80, 1);
        pullElement.style.transform = `translateY(${pullDistance.current * 0.5}px) rotate(${progress * 180}deg)`;
        pullElement.style.opacity = String(progress);
      }
    }
  }, []);

  const handleTouchEnd = useCallback(async () => {
    if (pullDistance.current > 80 && !isRefreshing.current) {
      isRefreshing.current = true;
      try {
        await onRefresh();
      } finally {
        isRefreshing.current = false;
      }
    }

    // Reset
    startY.current = 0;
    pullDistance.current = 0;
    const pullElement = containerRef.current?.querySelector('[data-pull-indicator]');
    if (pullElement instanceof HTMLElement) {
      pullElement.style.transform = '';
      pullElement.style.opacity = '0';
    }
  }, [onRefresh]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    container.addEventListener('touchstart', handleTouchStart, { passive: true });
    container.addEventListener('touchmove', handleTouchMove, { passive: false });
    container.addEventListener('touchend', handleTouchEnd, { passive: true });

    return () => {
      container.removeEventListener('touchstart', handleTouchStart);
      container.removeEventListener('touchmove', handleTouchMove);
      container.removeEventListener('touchend', handleTouchEnd);
    };
  }, [handleTouchStart, handleTouchMove, handleTouchEnd]);

  return {
    containerRef,
    isRefreshing: isRefreshing.current,
  };
}

/**
 * Hook for long press gesture
 */
export function useLongPress(
  onLongPress: () => void,
  onClick?: () => void,
  { delay = 500 }: { delay?: number } = {}
) {
  const timerRef = useRef<NodeJS.Timeout | null>(null);
  const isLongPress = useRef(false);

  const start = useCallback(() => {
    isLongPress.current = false;
    timerRef.current = setTimeout(() => {
      isLongPress.current = true;
      onLongPress();
    }, delay);
  }, [onLongPress, delay]);

  const clear = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const handleClick = useCallback(() => {
    if (!isLongPress.current && onClick) {
      onClick();
    }
  }, [onClick]);

  return {
    onMouseDown: start,
    onMouseUp: clear,
    onMouseLeave: clear,
    onTouchStart: start,
    onTouchEnd: clear,
    onClick: handleClick,
  };
}
