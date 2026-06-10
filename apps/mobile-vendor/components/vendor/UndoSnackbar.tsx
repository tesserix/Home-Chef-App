import { useEffect, useRef } from "react";
import { Animated, Easing, Pressable, StyleSheet, Text } from "react-native";
import { theme } from "@homechef/mobile-shared/theme";
import type { PendingUndo } from "../../hooks/useVendorOrders";

interface UndoSnackbarProps {
  pendingUndo: PendingUndo | null;
  onUndo: () => void;
  /** When provided, the snackbar surfaces a retry state instead of an
   *  undo state — used to recover from a failed accept/reject mutation. */
  errorMessage?: string | null;
  onRetry?: () => void;
}

/**
 * The post-action transient bar.
 *
 * Two roles:
 *  - **Undo:** ink background, "Order accepted/rejected" + UNDO. The
 *    3s grace window before the mutation commits.
 *  - **Retry:** destructive background, "Could not [action]" + RETRY.
 *    Surfaces when the optimistic action errors out.
 *
 * Motion: pure timing on the entrance/exit, not spring. The brand lock
 * explicitly forbids bounce / overshoot — the prior spring overshot by
 * a few pixels on entry.
 */
export function UndoSnackbar({
  pendingUndo,
  onUndo,
  errorMessage,
  onRetry,
}: UndoSnackbarProps) {
  const translateY = useRef(new Animated.Value(120)).current;
  const isVisible = !!pendingUndo || !!errorMessage;

  useEffect(() => {
    Animated.timing(translateY, {
      toValue: isVisible ? 0 : 120,
      duration: 250,
      easing: Easing.bezier(0.22, 1, 0.36, 1),
      useNativeDriver: true,
    }).start();
  }, [isVisible, translateY]);

  if (!isVisible) return null;

  const isError = !!errorMessage;
  const label = isError
    ? errorMessage
    : pendingUndo?.action === "accepted"
      ? "Order accepted"
      : "Order rejected";
  const actionLabel = isError ? "RETRY" : "UNDO";
  const onPressAction = isError ? onRetry : onUndo;

  return (
    <Animated.View
      style={[
        styles.root,
        isError ? styles.rootError : styles.rootUndo,
        { transform: [{ translateY }] },
      ]}
      accessibilityLiveRegion="polite"
    >
      <Text style={styles.label} numberOfLines={2}>
        {label}
      </Text>
      {onPressAction ? (
        <Pressable
          onPress={onPressAction}
          hitSlop={8}
          style={({ pressed }) => [styles.action, pressed && { opacity: 0.7 }]}
          accessibilityRole="button"
          accessibilityLabel={isError ? "Retry" : "Undo"}
        >
          <Text
            style={[styles.actionLabel, !isError && styles.actionLabelUndo]}
          >
            {actionLabel}
          </Text>
        </Pressable>
      ) : null}
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  root: {
    position: "absolute",
    bottom: theme.spacing[4],
    left: theme.spacing[4],
    right: theme.spacing[4],
    zIndex: 50,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    borderRadius: theme.radius.md,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    gap: theme.spacing[3],
    ...theme.shadow[3],
  },
  rootUndo: {
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  rootError: {
    backgroundColor: theme.colors.destructive.DEFAULT,
  },
  label: {
    flex: 1,
    fontFamily: "Inter-SemiBold",
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
  },
  action: {
    paddingVertical: theme.spacing[1],
    paddingHorizontal: theme.spacing[3],
    borderRadius: theme.radius.sm,
    backgroundColor: "rgba(255, 255, 255, 0.16)",
  },
  actionLabel: {
    fontFamily: "Inter-SemiBold",
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
    letterSpacing: 0.5,
  },
  // Brand-warm UNDO label on the ink bar (UI-V2 Wave B). RETRY keeps
  // paper — brand[200] would not read on the destructive red bg.
  actionLabelUndo: {
    color: theme.colors.brand[200],
  },
});
