// SheetBase — the low-level bottom-sheet shell shared by every bottom sheet
// in the app: the branded confirmation <Sheet>, and any app-level custom
// sheet (filters, address switcher, etc.) that needs full control over its
// own content.
//
// Why this exists: @gorhom/bottom-sheet's BottomSheetModal (and the raw,
// non-modal BottomSheet) silently no-ops on this app's gorhom 5.2.8 +
// reanimated 4.4.1 pairing — confirmed on-device (AddressSwitcherSheet's
// `.expand()` did nothing when tapped; apps/mobile-customer/app/cart.tsx
// documents the same landmine from an earlier migration: "Replaces the old
// @gorhom/bottom-sheet CartSheet, which silently failed to open"). This is a
// plain React Native `Modal` + `Animated` implementation with ZERO gorhom or
// reanimated dependency, so it cannot hit that version-pairing bug.
//
// Public contract mirrors gorhom's BottomSheetModal closely enough that
// every existing call site only needed a rename (`.expand()`/`.close()` →
// `.present()`/`.dismiss()`), never a rewrite:
//   const ref = useRef<SheetHandle>(null);
//   <Pressable onPress={() => ref.current?.present()} />
//   <SheetBase ref={ref}>{...}</SheetBase>

import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import {
  AccessibilityInfo,
  Animated,
  Easing,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  View,
  useWindowDimensions,
  type StyleProp,
  type ViewStyle,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { theme } from '../theme/tokens';

export interface SheetHandle {
  present: () => void;
  dismiss: () => void;
}

export interface SheetBaseProps {
  children: ReactNode;
  /** Fraction of window height the panel may grow to before its content
   *  scrolls internally. Default 0.9 (spec: maxHeight ~90%). */
  maxHeightRatio?: number;
  /** Extra style merged onto the panel — override background/shadow if the
   *  caller wants something other than the default bone/shadow[3] look
   *  (radius stays fixed at theme.radius.lg per the spec). */
  panelStyle?: StyleProp<ViewStyle>;
  /** Wraps `children` in a ScrollView (default true). Turn off when the
   *  caller already renders its own scroll container, to avoid nesting two
   *  vertical ScrollViews. */
  scrollable?: boolean;
  /** Shows the small drag-affordance pill at the top of the panel. Default
   *  true — matches the handle indicator every sheet had under gorhom. */
  showHandle?: boolean;
}

// Surface entrance/exit per THE SPEC §3.5: entrance uses the brand curve at
// the "surface" duration band (300–400ms); exits accelerate and stay short
// (~150ms) — never a slow, attention-grabbing exit.
const ENTER_DURATION = theme.motion.duration.page; // 400ms
const EXIT_DURATION = theme.motion.duration.micro; // 150ms
const ENTER_EASING = Easing.bezier(...theme.motion.easing.entrance);
const EXIT_EASING = Easing.bezier(...theme.motion.easing.state);

export const SheetBase = forwardRef<SheetHandle, SheetBaseProps>(function SheetBase(
  { children, maxHeightRatio = 0.9, panelStyle, scrollable = true, showHandle = true },
  ref,
) {
  const insets = useSafeAreaInsets();
  const { height: windowHeight } = useWindowDimensions();
  const [mounted, setMounted] = useState(false);
  const [visible, setVisible] = useState(false);
  const [reduceMotion, setReduceMotion] = useState(false);
  // Starts off-screen (not 0/open) — the Modal's very first paint happens
  // before the mount effect below has a chance to run `runEnter()`, so an
  // initial value of 0 rendered one full-open frame (panel fully in place)
  // before snapping off-screen and animating back in. windowHeight is
  // already known synchronously from useWindowDimensions above, so this
  // needs no layout measurement round-trip.
  const translateY = useRef(new Animated.Value(windowHeight)).current;
  const backdropOpacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    let alive = true;
    AccessibilityInfo.isReduceMotionEnabled()
      .then((enabled) => {
        if (alive) setReduceMotion(enabled);
      })
      .catch(() => {});
    const sub = AccessibilityInfo.addEventListener('reduceMotionChanged', (enabled) => {
      setReduceMotion(enabled);
    });
    return () => {
      alive = false;
      sub.remove();
    };
  }, []);

  const runEnter = useCallback(() => {
    if (reduceMotion) {
      translateY.setValue(0);
      backdropOpacity.setValue(1);
      return;
    }
    translateY.setValue(windowHeight);
    backdropOpacity.setValue(0);
    Animated.parallel([
      Animated.timing(translateY, {
        toValue: 0,
        duration: ENTER_DURATION,
        easing: ENTER_EASING,
        useNativeDriver: true,
      }),
      Animated.timing(backdropOpacity, {
        toValue: 1,
        duration: ENTER_DURATION,
        useNativeDriver: true,
      }),
    ]).start();
  }, [reduceMotion, translateY, backdropOpacity, windowHeight]);

  const runExit = useCallback(
    (onDone: () => void) => {
      if (reduceMotion) {
        translateY.setValue(windowHeight);
        backdropOpacity.setValue(0);
        onDone();
        return;
      }
      Animated.parallel([
        Animated.timing(translateY, {
          toValue: windowHeight,
          duration: EXIT_DURATION,
          easing: EXIT_EASING,
          useNativeDriver: true,
        }),
        Animated.timing(backdropOpacity, {
          toValue: 0,
          duration: EXIT_DURATION,
          useNativeDriver: true,
        }),
      ]).start(({ finished }) => {
        if (finished) onDone();
      });
    },
    [reduceMotion, translateY, backdropOpacity, windowHeight],
  );

  const present = useCallback(() => {
    setMounted(true);
    setVisible(true);
  }, []);

  const dismiss = useCallback(() => {
    runExit(() => {
      setVisible(false);
      setMounted(false);
    });
  }, [runExit]);

  useImperativeHandle(ref, () => ({ present, dismiss }), [present, dismiss]);

  // Kick the entrance animation once the Modal has actually mounted+visible.
  useEffect(() => {
    if (visible) runEnter();
  }, [visible, runEnter]);

  // Nothing to render (and no Modal in the native tree) until presented —
  // matches gorhom's index:-1 "closed, unmounted" resting state.
  if (!mounted) return null;

  // maxHeight lives on the ScrollView itself (RN's standard "grow to content,
  // cap at N, then scroll" recipe) — NOT on an ancestor View, which would
  // require a matching flex/height on the ScrollView to actually bound it.
  const maxContentHeight = windowHeight * maxHeightRatio;
  const content = scrollable ? (
    <ScrollView
      style={{ maxHeight: maxContentHeight }}
      contentContainerStyle={{ paddingBottom: insets.bottom }}
      showsVerticalScrollIndicator={false}
      keyboardShouldPersistTaps="handled"
    >
      {children}
    </ScrollView>
  ) : (
    <View style={{ maxHeight: maxContentHeight, paddingBottom: insets.bottom }}>{children}</View>
  );

  return (
    <Modal
      visible={visible}
      transparent
      animationType="none"
      statusBarTranslucent
      onRequestClose={dismiss}
    >
      <View style={styles.root} accessibilityViewIsModal>
        <Animated.View style={[styles.backdrop, { opacity: backdropOpacity }]}>
          <Pressable
            style={StyleSheet.absoluteFill}
            onPress={dismiss}
            accessibilityRole="button"
            accessibilityLabel="Dismiss"
          />
        </Animated.View>
        <KeyboardAvoidingView
          behavior={Platform.OS === 'ios' ? 'padding' : undefined}
          style={styles.kbFlex}
          pointerEvents="box-none"
        >
          {/* Shadow lives on this outer Animated.View (the transform also
              applies here) — overflow:hidden for the rounded corners lives
              on a separate inner View. iOS drops the shadow if both live on
              the same node (the codebase's documented shadow+clip gotcha).
              `panelStyle` (background/radius overrides) applies to the inner
              clip view, where background/radius belong. */}
          <Animated.View style={[styles.panelShadow, { transform: [{ translateY }] }]}>
            <View style={[styles.panelClip, panelStyle]}>
              {showHandle ? <View style={styles.handle} /> : null}
              {content}
            </View>
          </Animated.View>
        </KeyboardAvoidingView>
      </View>
    </Modal>
  );
});

const styles = StyleSheet.create({
  root: { flex: 1 },
  backdrop: {
    ...StyleSheet.absoluteFill,
    backgroundColor: 'rgba(26, 26, 24, 0.35)',
  },
  kbFlex: { flex: 1, justifyContent: 'flex-end' },
  panelShadow: {
    ...theme.shadow[3],
  },
  panelClip: {
    backgroundColor: theme.colors.bone,
    borderTopLeftRadius: theme.radius.lg,
    borderTopRightRadius: theme.radius.lg,
    overflow: 'hidden',
  },
  handle: {
    alignSelf: 'center',
    width: 40,
    height: 4,
    borderRadius: 2,
    backgroundColor: theme.colors.mist.strong,
    marginTop: theme.spacing[2],
    marginBottom: theme.spacing[1],
  },
});
