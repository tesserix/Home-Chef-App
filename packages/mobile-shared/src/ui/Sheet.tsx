import {
  forwardRef,
  useCallback,
  useImperativeHandle,
  useMemo,
  useRef,
  type ReactNode,
} from 'react';
import { Animated, Pressable, StyleSheet, Text, View } from 'react-native';
import { BottomSheetModal as _BottomSheetModal } from '@gorhom/bottom-sheet';

// The @gorhom/bottom-sheet types drift against the duplicated @types/react
// in this monorepo (the workspace has two copies — one in mobile-shared/
// node_modules and one at the root). Casting to any sidesteps the JSX
// element-class incompatibility. Runtime behavior is identical.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const BottomSheetModal = _BottomSheetModal as any;
import { theme } from '../theme/tokens';
import { Button } from './Button';

// Type-loose backdrop component prop. The full BottomSheetBackdropProps
// type isn't re-exported from the package root in some versions; we only
// need `animatedIndex` for opacity interpolation.
interface BackdropComponentProps {
  animatedIndex: Animated.Value;
  style: object;
}

export interface SheetHandle {
  present: () => void;
  dismiss: () => void;
}

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
  /** Snap points for the gorhom bottom sheet. Defaults sized for a single
   *  title + body + 2 action buttons. */
  snapPoints?: string[] | number[];
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
    snapPoints,
  }: SheetProps,
  ref,
) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const modalRef = useRef<any>(null);
  const snaps = useMemo(() => snapPoints ?? ['40%'], [snapPoints]);

  useImperativeHandle(
    ref,
    () => ({
      present: () => modalRef.current?.present(),
      dismiss: () => modalRef.current?.dismiss(),
    }),
    [],
  );

  // Inline custom backdrop — tap to dismiss, ink at 35% opacity.
  // Avoids depending on @gorhom/bottom-sheet's BottomSheetBackdrop export,
  // which isn't re-exported from the package root in some workspace
  // configurations.
  const renderBackdrop = useCallback(
    (props: BackdropComponentProps) => (
      <Pressable
        accessibilityLabel="Dismiss"
        onPress={() => modalRef.current?.dismiss()}
        style={[props.style, { backgroundColor: 'rgba(26, 26, 24, 0.35)' }]}
      />
    ),
    [],
  );

  return (
    <BottomSheetModal
      ref={modalRef}
      snapPoints={snaps}
      backdropComponent={renderBackdrop}
      handleIndicatorStyle={styles.handle}
      backgroundStyle={styles.background}
      enableDynamicSizing={!snapPoints}
    >
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
                modalRef.current?.dismiss();
              }}
            />
          ) : null}
          <Button
            label={cancelLabel}
            variant="ghost"
            onPress={() => modalRef.current?.dismiss()}
          />
        </View>
      </View>
    </BottomSheetModal>
  );
});

const styles = StyleSheet.create({
  background: {
    backgroundColor: theme.colors.bone,
    borderTopLeftRadius: theme.radius.lg,
    borderTopRightRadius: theme.radius.lg,
  },
  handle: {
    backgroundColor: theme.colors.mist.strong,
    width: 40,
  },
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
