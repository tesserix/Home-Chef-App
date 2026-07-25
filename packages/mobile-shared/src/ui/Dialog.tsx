import { useCallback, useEffect, useRef } from 'react';
import {
  Animated,
  BackHandler,
  Easing,
  Modal,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  View,
  useWindowDimensions,
} from 'react-native';

import { theme } from '../theme/tokens';

// Dialog — the branded centre-screen popup.
//
// Replaces Alert.alert for short blocking decisions. The stock alert renders in
// each platform's own chrome: on Android that means Material corners and TEAL
// action text, so the most consequential moments in the app — cancel this plan,
// delete your account — are the ones that look least like the app.
//
// This is a plain Modal + Animated implementation, deliberately: SheetBase
// documents that this project's @gorhom/bottom-sheet + reanimated pairing
// silently no-ops, and a confirmation that fails to appear is worse than an ugly
// one. No gorhom, no reanimated, nothing to version-pair.
//
// Use <Sheet> instead when the content is long, scrollable, or a picker — a
// bottom sheet carries that comfortably and a centred box does not.

const ENTER_MS = 180;
const EXIT_MS = 140;
// .impeccable.md's entrance curve. No bounce, no overshoot — a confirmation
// should feel settled the instant it lands, not springy.
const EASE_OUT = Easing.bezier(0.22, 1, 0.36, 1);

export interface DialogAction {
  label: string;
  onPress?: () => void;
  /** Renders in the destructive colour. For the action that cannot be undone. */
  destructive?: boolean;
  /** The quiet, dismissing choice. Rendered before the primary action. */
  cancel?: boolean;
}

export interface DialogProps {
  visible: boolean;
  title: string;
  /** Supporting copy. Say what happens and whether it can be undone. */
  message?: string;
  actions: DialogAction[];
  /** Called on backdrop tap and Android back. Omit to make the dialog modal
   *  in the strict sense — a decision the user cannot sidestep. */
  onDismiss?: () => void;
  /** Accent for the primary action. Mirrors <Button accentColor> — the shared
   *  palette is persimmon while the customer app runs coral, so the accent is
   *  injected per app rather than baked in here. Defaults to ink. */
  accentColor?: string;
}

export function Dialog({
  visible,
  title,
  message,
  actions,
  onDismiss,
  accentColor,
}: DialogProps) {
  const { width } = useWindowDimensions();
  // Two animated values rather than one: the backdrop only fades, while the
  // card fades AND lifts. Driving both from one value would tie the card's
  // travel to the backdrop's opacity curve.
  const progress = useRef(new Animated.Value(0)).current;
  const mounted = useRef(false);

  useEffect(() => {
    // Skip the entrance animation on first mount when already visible, so a
    // dialog restored with the screen doesn't animate in unprompted.
    const duration = mounted.current ? (visible ? ENTER_MS : EXIT_MS) : 0;
    mounted.current = true;
    Animated.timing(progress, {
      toValue: visible ? 1 : 0,
      duration,
      easing: EASE_OUT,
      useNativeDriver: true,
    }).start();
  }, [visible, progress]);

  // Android hardware back dismisses, matching what the platform alert does.
  useEffect(() => {
    if (!visible || !onDismiss) return;
    const sub = BackHandler.addEventListener('hardwareBackPress', () => {
      onDismiss();
      return true;
    });
    return () => sub.remove();
  }, [visible, onDismiss]);

  const handleAction = useCallback((action: DialogAction) => {
    action.onPress?.();
  }, []);

  // Cancel-style actions read first, so the eye lands on the way out before the
  // irreversible one — the same ordering the platform alerts use.
  const ordered = [...actions].sort((a, b) => Number(!!b.cancel) - Number(!!a.cancel));

  return (
    <Modal
      visible={visible}
      transparent
      animationType="none"
      statusBarTranslucent
      onRequestClose={onDismiss}
    >
      <Animated.View style={[styles.backdrop, { opacity: progress }]}>
        <Pressable
          style={StyleSheet.absoluteFill}
          onPress={onDismiss}
          // Only announce a dismiss target when there actually is one.
          accessibilityRole={onDismiss ? 'button' : undefined}
          accessibilityLabel={onDismiss ? 'Dismiss' : undefined}
          disabled={!onDismiss}
        />
        <Animated.View
          accessibilityViewIsModal
          accessibilityRole="alert"
          style={[
            styles.card,
            {
              // Cap the width so the dialog never spans a tablet edge to edge.
              width: Math.min(width - theme.spacing[6] * 2, 420),
              opacity: progress,
              transform: [
                {
                  translateY: progress.interpolate({
                    inputRange: [0, 1],
                    outputRange: [12, 0],
                  }),
                },
                {
                  // A restrained rise, not a zoom: 0.97 reads as "settling",
                  // where a 0.8 start reads as a bounce.
                  scale: progress.interpolate({
                    inputRange: [0, 1],
                    outputRange: [0.97, 1],
                  }),
                },
              ],
            },
          ]}
        >
          <View style={styles.body}>
            <Text style={styles.title}>{title}</Text>
            {message ? <Text style={styles.message}>{message}</Text> : null}
          </View>

          <View style={styles.divider} />

          <View style={styles.actions}>
            {ordered.map((action, i) => (
              <Pressable
                key={action.label}
                onPress={() => handleAction(action)}
                accessibilityRole="button"
                accessibilityLabel={action.label}
                android_ripple={{ color: `${theme.colors.ink.DEFAULT}14` }}
                style={styles.actionHit}
              >
                {({ pressed }) => (
                  <Text
                    style={[
                      styles.actionLabel,
                      action.destructive && styles.actionDestructive,
                      // The last action is the primary one; earlier ones are quiet.
                      !action.destructive &&
                        i === ordered.length - 1 &&
                        (accentColor ? { color: accentColor } : styles.actionPrimary),
                      action.cancel && styles.actionCancel,
                      pressed && Platform.OS === 'ios' && styles.actionPressed,
                    ]}
                  >
                    {action.label}
                  </Text>
                )}
              </Pressable>
            ))}
          </View>
        </Animated.View>
      </Animated.View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#00000066',
    paddingHorizontal: theme.spacing[6],
  },
  card: {
    backgroundColor: theme.colors.paper,
    // 12px: crisper than the platform alert's rounding, without drifting into
    // the pill shapes .impeccable.md rules out.
    borderRadius: theme.radius.md,
    overflow: 'hidden',
    // The modal step of the three-step scale — elevation, never decoration.
    ...theme.shadow[3],
  },
  body: {
    paddingHorizontal: theme.spacing[5],
    paddingTop: theme.spacing[5],
    paddingBottom: theme.spacing[4],
  },
  title: {
    fontSize: 18,
    fontWeight: '600',
    color: theme.colors.ink.DEFAULT,
    letterSpacing: -0.2,
  },
  message: {
    marginTop: theme.spacing[2],
    fontSize: 15,
    lineHeight: 21,
    color: theme.colors.ink.soft,
  },
  divider: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
  },
  actions: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[2],
  },
  actionHit: {
    // 48px: a destructive confirmation is the last place to make a target tight.
    minHeight: 48,
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[4],
  },
  actionLabel: {
    fontSize: 15,
    fontWeight: '600',
    color: theme.colors.ink.DEFAULT,
  },
  actionPrimary: { color: theme.colors.ink.DEFAULT },
  actionDestructive: { color: theme.colors.destructive.DEFAULT },
  actionCancel: { color: theme.colors.ink.soft, fontWeight: '500' },
  actionPressed: { opacity: 0.6 },
});
