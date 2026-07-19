import { forwardRef } from 'react';
import { ScrollView, type ScrollViewProps } from 'react-native';

/**
 * ScrollView preconfigured so the on-screen keyboard never hides the focused
 * field: it insets the scroll by the keyboard height and scrolls the focused
 * input into view (iOS), keeps taps working while the keyboard is open, and lets
 * the user swipe the keyboard down. Use this for ANY scrollable form instead of a
 * bare ScrollView with repeated keyboard props. All defaults are overridable via
 * props.
 */
export const KeyboardAwareScrollView = forwardRef<ScrollView, ScrollViewProps>(
  function KeyboardAwareScrollView(props, ref) {
    return (
      <ScrollView
        ref={ref}
        keyboardShouldPersistTaps="handled"
        keyboardDismissMode="interactive"
        automaticallyAdjustKeyboardInsets
        contentInsetAdjustmentBehavior="automatic"
        {...props}
      />
    );
  },
);
