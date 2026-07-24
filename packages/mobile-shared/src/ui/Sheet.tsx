import { forwardRef, type ReactNode } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { theme } from '../theme/tokens';
import { Button } from './Button';
import { SheetBase, type SheetHandle } from './SheetBase';

export type { SheetHandle };

interface SheetProps {
  title: string;
  body?: string;
  children?: ReactNode;
  /** Optional primary action. When provided alongside a destructive flag,
   *  the button renders in red. */
  primaryLabel?: string;
  primaryDestructive?: boolean;
  onPrimaryPress?: () => void;
  /** Cancel / dismiss label. Defaults to "Cancel". */
  cancelLabel?: string;
  /** Accepted for call-site compatibility with the previous gorhom-backed
   *  implementation, which sized the sheet to fixed snap-point percentages.
   *  The Modal-based shell is content-height with an internal max-height
   *  scroll instead, so this value is accepted and intentionally ignored —
   *  every existing call site (including ones passing e.g. `['70%']`)
   *  continues to compile and renders a sensibly-sized, scrollable sheet. */
  snapPoints?: string[] | number[];
  /** Passed straight through to SheetBase — turn off when `children` already
   *  renders its own scroll container, to avoid nesting two ScrollViews. */
  scrollable?: boolean;
}

/**
 * <Sheet> — the canonical confirmation sheet.
 *
 * Use this instead of `Alert.alert` for any flow where:
 *   - The user is confirming a destructive action (delete menu item,
 *     cancel order, archive)
 *   - The user must make a binary choice (accept / reject)
 *   - The decision benefits from supporting copy beyond a single line
 *
 * Imperative API via ref to match the BottomSheetModal pattern:
 *   const sheetRef = useRef<SheetHandle>(null);
 *   ...
 *   <Pressable onPress={() => sheetRef.current?.present()} />
 *   <Sheet ref={sheetRef} title="Delete item?" ... />
 *
 * Built on <SheetBase> — a plain React Native Modal + Animated
 * implementation with no @gorhom/bottom-sheet dependency (that library
 * silently no-ops `.present()`/`.expand()` on this app's gorhom 5.2.8 +
 * reanimated 4.4.1 pairing, confirmed on-device). The public API here is
 * unchanged from the previous gorhom-backed version.
 */
export const Sheet = forwardRef<SheetHandle, SheetProps>(function Sheet(
  {
    title,
    body,
    children,
    primaryLabel,
    primaryDestructive = false,
    onPrimaryPress,
    cancelLabel = 'Cancel',
    scrollable,
  }: SheetProps,
  ref,
) {
  // `ref` is always a plain RefObject in every call site in this codebase
  // (`useRef<SheetHandle>(null)`), but guard the callback-ref case too —
  // mirrors the exact guard FilterSheet/AddressSwitcherSheet already use.
  const dismiss = () => {
    if (ref && 'current' in ref && ref.current) ref.current.dismiss();
  };

  return (
    <SheetBase ref={ref} scrollable={scrollable}>
      <View style={styles.content}>
        <Text style={styles.title}>{title}</Text>
        {body ? <Text style={styles.body}>{body}</Text> : null}
        {children}
        <View style={styles.actions}>
          {primaryLabel && onPrimaryPress ? (
            <Button
              label={primaryLabel}
              variant={primaryDestructive ? 'destructive' : 'primary'}
              onPress={() => {
                onPrimaryPress();
                dismiss();
              }}
            />
          ) : null}
          <Button label={cancelLabel} variant="ghost" onPress={dismiss} />
        </View>
      </View>
    </SheetBase>
  );
});

const styles = StyleSheet.create({
  content: {
    padding: theme.spacing[5],
    gap: theme.spacing[3],
  },
  title: {
    fontFamily: 'Geist',
    fontSize: theme.typography.size.h2.size,
    color: theme.colors.ink.DEFAULT,
  },
  body: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    lineHeight:
      theme.typography.size.body.size * theme.typography.size.body.lineHeight,
    color: theme.colors.ink.soft,
  },
  actions: {
    marginTop: theme.spacing[3],
    gap: theme.spacing[2],
  },
});
