import { useRef, useState } from 'react';
import { PanResponder, View, type LayoutChangeEvent } from 'react-native';

import { customerColors } from '@homechef/mobile-shared/theme';

// CreditSlider — a drag control for dialling a credit rail down from its maximum.
//
// Built on PanResponder rather than a slider package deliberately: every RN slider
// library is a NATIVE module, which would mean a lockfile change and a dev-client
// rebuild for what is a horizontal drag. This keeps it pure JS and lets the track
// use the brand accent directly.

interface CreditSliderProps {
  value: number;
  max: number;
  /** Fires continuously during the drag so the caller can debounce its own re-quote. */
  onChange: (value: number) => void;
  /** Fires once when the finger lifts — the moment to commit an expensive update. */
  onCommit?: (value: number) => void;
  /** Quantise to whole units (points must be integers; rupees need not be). */
  step?: number;
  disabled?: boolean;
  accessibilityLabel: string;
}

export function CreditSlider({
  value,
  max,
  onChange,
  onCommit,
  step = 1,
  disabled = false,
  accessibilityLabel,
}: CreditSliderProps) {
  const [width, setWidth] = useState(0);
  // Refs, not state: the PanResponder is created once and its callbacks would
  // otherwise close over the first render's values forever.
  const widthRef = useRef(0);
  const maxRef = useRef(max);
  const latest = useRef(value);
  // Page-x of the track, so a drag maps to an absolute finger position rather
  // than drifting by the card's own inset.
  const trackX = useRef(0);
  maxRef.current = max;
  latest.current = value;

  const quantise = (raw: number): number => {
    const clamped = Math.max(0, Math.min(maxRef.current, raw));
    return step > 0 ? Math.round(clamped / step) * step : clamped;
  };

  const responder = useRef(
    PanResponder.create({
      onStartShouldSetPanResponder: () => true,
      onMoveShouldSetPanResponder: () => true,
      onPanResponderGrant: (evt) => {
        if (widthRef.current <= 0 || maxRef.current <= 0) return;
        const next = quantise((evt.nativeEvent.locationX / widthRef.current) * maxRef.current);
        latest.current = next;
        onChange(next);
      },
      onPanResponderMove: (_evt, gesture) => {
        if (widthRef.current <= 0 || maxRef.current <= 0) return;
        const next = quantise(((gesture.moveX - trackX.current) / widthRef.current) * maxRef.current);
        latest.current = next;
        onChange(next);
      },
      onPanResponderRelease: () => onCommit?.(latest.current),
      onPanResponderTerminate: () => onCommit?.(latest.current),
    }),
  ).current;

  function handleLayout(e: LayoutChangeEvent) {
    const w = e.nativeEvent.layout.width;
    widthRef.current = w;
    setWidth(w);
  }

  const pct = max > 0 ? Math.max(0, Math.min(1, value / max)) : 0;
  const thumbLeft = Math.max(0, width * pct - 11);

  return (
    <View
      // 44px of vertical hit area around a 4px track — the visual weight stays
      // light without making the control hard to grab.
      className="h-11 justify-center"
      onLayout={handleLayout}
      accessible
      accessibilityRole="adjustable"
      accessibilityLabel={accessibilityLabel}
      accessibilityValue={{ min: 0, max: Math.round(max), now: Math.round(value) }}
      {...(disabled ? {} : responder.panHandlers)}
      ref={(node) => {
        node?.measure?.((_x, _y, _w, _h, pageX) => {
          trackX.current = pageX;
        });
      }}
    >
      <View
        className="h-1 rounded-full"
        style={{ backgroundColor: customerColors.hairline }}
      >
        <View
          className="h-1 rounded-full"
          style={{
            width: `${pct * 100}%`,
            backgroundColor: disabled ? customerColors.charcoal.soft : customerColors.coral.DEFAULT,
          }}
        />
      </View>
      {!disabled && (
        <View
          className="absolute h-[22px] w-[22px] rounded-full border-2"
          style={{
            left: thumbLeft,
            backgroundColor: customerColors.canvas,
            borderColor: customerColors.coral.DEFAULT,
          }}
        />
      )}
    </View>
  );
}
