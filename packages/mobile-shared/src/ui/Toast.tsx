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

type ToastTone = 'success' | 'info' | 'error';

interface ToastInput {
  message: string;
  tone?: ToastTone;
  /** Optional action button label. When provided, displays a button on
   *  the right edge of the toast. */
  action?: { label: string; onPress: () => void };
  /** Auto-dismiss timeout in ms. Default 3500. Pass `null` to require
   *  manual dismissal via the action button. */
  durationMs?: number | null;
}

interface ToastContextValue {
  show: (input: ToastInput) => void;
  dismiss: () => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

/**
 * <ToastProvider> — mount at the root, beneath the AuthProvider and
 * QueryClientProvider, above the navigation Stack so toasts render
 * over every route.
 *
 * Usage:
 *   const { show } = useToast();
 *   show({ message: 'Profile saved', tone: 'success' });
 *
 * Use this for non-blocking confirmations. For blocking confirmations
 * (delete? cancel? confirm?), use <Sheet> instead. Alert.alert is the
 * fallback when neither toast nor sheet fits — usually error states the
 * user must acknowledge.
 */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<ToastInput | null>(null);
  const opacity = useRef(new Animated.Value(0)).current;
  const translateY = useRef(new Animated.Value(20)).current;
  const insets = useSafeAreaInsets();
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const dismiss = useCallback(() => {
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
    ]).start(() => setToast(null));
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
  }, [opacity, translateY]);

  const show = useCallback(
    (input: ToastInput) => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
      setToast(input);
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
      ]).start();
      const duration = input.durationMs ?? 3500;
      if (duration !== null) {
        timeoutRef.current = setTimeout(dismiss, duration);
      }
    },
    [opacity, translateY, dismiss],
  );

  useEffect(
    () => () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    },
    [],
  );

  const value = useMemo(() => ({ show, dismiss }), [show, dismiss]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      {toast ? (
        <Animated.View
          pointerEvents="box-none"
          style={[
            styles.wrap,
            { bottom: insets.bottom + theme.spacing[4], opacity, transform: [{ translateY }] },
          ]}
        >
          <Pressable
            onPress={dismiss}
            accessibilityRole="alert"
            accessibilityLiveRegion="polite"
            style={[styles.toast, toneStyles[toast.tone ?? 'info']]}
          >
            <Text style={styles.message} numberOfLines={2}>
              {toast.message}
            </Text>
            {toast.action ? (
              <Pressable
                onPress={() => {
                  toast.action?.onPress();
                  dismiss();
                }}
                hitSlop={8}
              >
                <Text style={styles.actionLabel}>{toast.action.label}</Text>
              </Pressable>
            ) : null}
          </Pressable>
        </Animated.View>
      ) : null}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error('useToast() requires a <ToastProvider> ancestor');
  }
  return ctx;
}

const styles = StyleSheet.create({
  wrap: {
    position: 'absolute',
    left: theme.spacing[4],
    right: theme.spacing[4],
    alignItems: 'center',
  },
  toast: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    borderRadius: theme.radius.md,
    minWidth: '100%',
    ...theme.shadow[3],
  },
  message: {
    flex: 1,
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
  actionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
  },
});

const toneStyles: Record<ToastTone, { backgroundColor: string }> = {
  success: { backgroundColor: theme.colors.success.DEFAULT },
  info: { backgroundColor: theme.colors.ink.DEFAULT },
  error: { backgroundColor: theme.colors.destructive.DEFAULT },
};
