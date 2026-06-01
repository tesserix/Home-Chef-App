import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import {
  Animated,
  Easing,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { theme } from '../theme/tokens';

interface UndoInput {
  message: string;
  /** Called when the user taps "Undo" before the timer expires. */
  onUndo: () => void;
  /** Called when the timer expires without Undo being tapped. Use this to
   *  perform the *actual* destructive operation (POST /delete, etc) —
   *  the snackbar is the user's window to back out. */
  onCommit: () => void;
  /** Default 5000ms — Material guidance is 4–10s; 5s is the sweet spot
   *  for vendor / customer apps. Driver app should override to 7s. */
  durationMs?: number;
}

interface UndoContextValue {
  show: (input: UndoInput) => void;
}

const UndoContext = createContext<UndoContextValue | null>(null);

/**
 * <UndoSnackbarProvider> — every destructive action that is reversible
 * within a short window (delete menu item, archive order, dismiss
 * notification) goes through this.
 *
 * The pattern:
 *   1. Optimistically remove the item from the UI.
 *   2. Call `show({ onUndo, onCommit })` — the snackbar appears.
 *   3. If the user taps Undo → restore the item, call `onUndo`.
 *   4. If the timer expires → call `onCommit` which performs the actual
 *      destructive API call.
 *
 * Why this beats a confirm dialog: zero friction for the 99% case, full
 * recovery for the 1% mistake case. .impeccable.md #3: confidence
 * through restraint.
 */
export function UndoSnackbarProvider({ children }: { children: ReactNode }) {
  const [current, setCurrent] = useState<UndoInput | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;
  const translateY = useRef(new Animated.Value(20)).current;
  const progress = useRef(new Animated.Value(1)).current;
  const insets = useSafeAreaInsets();
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const dismissedRef = useRef(false);

  const hide = useCallback(() => {
    Animated.parallel([
      Animated.timing(opacity, {
        toValue: 0,
        duration: theme.motion.duration.micro,
        useNativeDriver: true,
      }),
      Animated.timing(translateY, {
        toValue: 20,
        duration: theme.motion.duration.micro,
        useNativeDriver: true,
      }),
    ]).start(() => setCurrent(null));
  }, [opacity, translateY]);

  const show = useCallback(
    (input: UndoInput) => {
      // If a previous undo snackbar is still up, commit it before showing
      // the new one — never queue / never silently drop the prior.
      if (current && !dismissedRef.current) {
        current.onCommit();
      }
      dismissedRef.current = false;
      setCurrent(input);
      opacity.setValue(0);
      translateY.setValue(20);
      progress.setValue(1);

      const duration = input.durationMs ?? 5000;
      Animated.parallel([
        Animated.timing(opacity, {
          toValue: 1,
          duration: theme.motion.duration.default,
          easing: Easing.bezier(...theme.motion.easing.entrance),
          useNativeDriver: true,
        }),
        Animated.timing(translateY, {
          toValue: 0,
          duration: theme.motion.duration.default,
          easing: Easing.bezier(...theme.motion.easing.entrance),
          useNativeDriver: true,
        }),
        Animated.timing(progress, {
          toValue: 0,
          duration,
          easing: Easing.linear,
          useNativeDriver: false,
        }),
      ]).start();

      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      timeoutRef.current = setTimeout(() => {
        if (!dismissedRef.current) {
          input.onCommit();
          hide();
        }
      }, duration);
    },
    [current, opacity, translateY, progress, hide],
  );

  const handleUndo = useCallback(() => {
    if (!current) return;
    dismissedRef.current = true;
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    current.onUndo();
    hide();
  }, [current, hide]);

  useEffect(
    () => () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    },
    [],
  );

  const value = useMemo(() => ({ show }), [show]);

  return (
    <UndoContext.Provider value={value}>
      {children}
      {current ? (
        <Animated.View
          pointerEvents="box-none"
          style={[
            styles.wrap,
            { bottom: insets.bottom + theme.spacing[4], opacity, transform: [{ translateY }] },
          ]}
        >
          <View style={styles.snackbar}>
            <Text style={styles.message} numberOfLines={1}>
              {current.message}
            </Text>
            <Pressable onPress={handleUndo} hitSlop={8}>
              <Text style={styles.undo}>Undo</Text>
            </Pressable>
          </View>
          <Animated.View
            style={[
              styles.progress,
              {
                width: progress.interpolate({
                  inputRange: [0, 1],
                  outputRange: ['0%', '100%'],
                }),
              },
            ]}
          />
        </Animated.View>
      ) : null}
    </UndoContext.Provider>
  );
}

export function useUndoSnackbar(): UndoContextValue {
  const ctx = useContext(UndoContext);
  if (!ctx) {
    throw new Error('useUndoSnackbar() requires a <UndoSnackbarProvider> ancestor');
  }
  return ctx;
}

const styles = StyleSheet.create({
  wrap: {
    position: 'absolute',
    left: theme.spacing[4],
    right: theme.spacing[4],
    borderRadius: theme.radius.md,
    overflow: 'hidden',
    ...theme.shadow[3],
  },
  snackbar: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: theme.colors.ink.DEFAULT,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    gap: theme.spacing[3],
  },
  message: {
    flex: 1,
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
  undo: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.herb.tint,
    letterSpacing: 0.5,
  },
  progress: {
    height: 2,
    backgroundColor: theme.colors.herb.DEFAULT,
  },
});
