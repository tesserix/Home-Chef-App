import { useEffect, useRef, type ReactNode, type RefObject } from 'react';
import {
  Animated,
  Easing,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { theme } from '../theme/tokens';
import { Button } from './Button';
import { useReducedMotion } from './useReducedMotion';

interface OnboardingScaffoldProps {
  /** Current step (1-indexed). */
  step: number;
  /** Total steps. Used for both progress dots and the "Step N of M" label. */
  total: number;
  /** Short phase name for this step (e.g. "Your details", "Documents"). When
   *  provided, shown as an eyebrow above the title so the user knows which phase
   *  of the journey they're in, not just a number. */
  stepName?: string;
  /** Geist headline — sentence case, no exclamation. */
  title: string;
  /** Inter body subtitle — one sentence explaining *why* this step. */
  subtitle?: string;
  /** Form content — typically a column of Inputs. */
  children: ReactNode;
  /** Primary action label. Default "Continue". */
  primaryLabel?: string;
  onPrimary: () => void;
  primaryLoading?: boolean;
  primaryDisabled?: boolean;
  /** Optional back action. When provided, renders a top-left chevron text
   *  link "Back". */
  onBack?: () => void;
  /** One short line of encouragement for this step — what the chef gets out of
   *  finishing it, not what the form wants. A signup wizard reads as paperwork
   *  without it; this is the line that answers "why bother". */
  encouragement?: string;
  /** Optional ref to the internal form ScrollView. Forms with react-hook-form
   *  validation pass this so an invalid submit can scroll the first errored
   *  field into view (R14) — the ScrollView otherwise isn't reachable from
   *  the screen since this component owns it. */
  scrollRef?: RefObject<ScrollView | null>;
}

/**
 * <OnboardingScaffold> — the shared chrome for every step of the
 * vendor onboarding wizard (and any future wizard with the same shape).
 *
 * Visual structure:
 *   [Back]                     [Step N · Total]   ← top bar
 *   [Progress dots]                               ← ink for done/current, mist for upcoming
 *
 *   <Geist title>
 *   <Inter subtitle>
 *
 *   <form fields>                                 ← scrollable
 *
 *   [Continue]                                    ← sticky bottom CTA
 *
 * The sticky CTA is the big UX improvement: users currently have to
 * scroll past the form to find the button on some screens, which
 * compounds with the iOS keyboard taking ~40% of the screen.
 */
export function OnboardingScaffold({
  step,
  total,
  stepName,
  title,
  subtitle,
  children,
  primaryLabel = 'Continue',
  onPrimary,
  primaryLoading = false,
  primaryDisabled = false,
  onBack,
  encouragement,
  scrollRef,
}: OnboardingScaffoldProps) {
  // Completion counts *finished* steps, so step 1 honestly reads 0% rather than
  // taking credit for work not yet done; the last step lands on 100%.
  const percent = Math.round(((step - 1) / Math.max(total - 1, 1)) * 100);

  const fill = useRef(new Animated.Value(percent)).current;
  const reduceMotion = useReducedMotion();

  useEffect(() => {
    if (reduceMotion) {
      fill.setValue(percent);
      return;
    }
    Animated.timing(fill, {
      toValue: percent,
      duration: 400,
      // Brand easing — decelerate, no overshoot. Width can't use the native
      // driver, hence animating a percentage rather than a transform.
      easing: Easing.bezier(0.22, 1, 0.36, 1),
      useNativeDriver: false,
    }).start();
    // Deliberate behavior improvement over the old ref-based version: the
    // fill animation now correctly reacts if Reduce Motion changes
    // mid-session, which the ref (read once, never causing a re-render)
    // silently ignored.
  }, [percent, fill, reduceMotion]);

  const fillWidth = fill.interpolate({
    inputRange: [0, 100],
    outputRange: ['0%', '100%'],
  });

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      {/* iOS uses 'padding' to lift the sticky CTA above the keyboard. On
          Android, 'height' fights the padding-applying SafeAreaView above —
          the two re-measure each other on mount and loop ("Maximum update
          depth exceeded"). Android's window already resizes for the keyboard
          (adjustResize), so we leave behavior undefined (passthrough). */}
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        style={styles.kav}
      >
        {/* Top bar */}
        <View style={styles.topBar}>
          {onBack ? (
            <Pressable
              onPress={onBack}
              hitSlop={12}
              accessibilityRole="button"
              accessibilityLabel="Back"
              android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
            >
              {({ pressed }) => (
                <Text
                  style={[
                    styles.backLink,
                    pressed && Platform.OS === 'ios' && { opacity: 0.6 },
                  ]}
                >
                  ← Back
                </Text>
              )}
            </Pressable>
          ) : (
            <View />
          )}
          <Text
            style={styles.stepLabel}
            accessibilityLabel={`Step ${step} of ${total}, ${percent} percent complete`}
          >
            Step {step} of {total}
            <Text style={styles.stepDivider}>{'  ·  '}</Text>
            <Text style={styles.percentLabel}>{percent}%</Text>
          </Text>
        </View>

        {/* Progress — one continuous track with an animated ink fill, plus a
            tick per step boundary. The segmented bar showed position but never
            how much was LEFT; the percentage and the growing fill are what make
            progress feel like progress. Monochrome by design: this scaffold is
            vendor-exclusive and carries no persimmon. */}
        <View
          style={styles.progressRow}
          accessibilityRole="progressbar"
          accessibilityValue={{ min: 0, max: 100, now: percent }}
        >
          <View style={styles.track}>
            <Animated.View style={[styles.trackFill, { width: fillWidth }]} />
            <View style={styles.tickRow} pointerEvents="none">
              {Array.from({ length: total - 1 }).map((_, i) => (
                <View key={i} style={styles.tick} />
              ))}
            </View>
          </View>
        </View>

        <ScrollView
          ref={scrollRef}
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          {stepName ? <Text style={styles.eyebrow}>{stepName}</Text> : null}
          <Text style={styles.title}>{title}</Text>
          {subtitle ? (
            <Text style={[styles.subtitle, encouragement && styles.subtitleTight]}>
              {subtitle}
            </Text>
          ) : null}
          {encouragement ? (
            <View style={styles.encouragement}>
              <Text style={styles.encouragementText}>{encouragement}</Text>
            </View>
          ) : null}
          <View style={styles.form}>{children}</View>
        </ScrollView>

        {/* Sticky CTA */}
        <View style={styles.ctaWrap}>
          <Button
            label={primaryLabel}
            onPress={onPrimary}
            loading={primaryLoading}
            disabled={primaryDisabled || primaryLoading}
          />
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  kav: { flex: 1 },
  topBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[6],
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[2],
  },
  backLink: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },
  stepLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.5,
    fontVariant: ['tabular-nums'],
  },
  stepDivider: { color: theme.colors.mist.strong },
  percentLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ['tabular-nums'],
  },

  progressRow: {
    paddingHorizontal: theme.spacing[6],
    paddingBottom: theme.spacing[4],
  },
  track: {
    height: 4,
    borderRadius: 2,
    backgroundColor: theme.colors.mist.DEFAULT,
    overflow: 'hidden',
    justifyContent: 'center',
  },
  trackFill: {
    position: 'absolute',
    left: 0,
    top: 0,
    bottom: 0,
    borderRadius: 2,
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  // Hairline ticks keep the "N discrete steps" reading that the segmented bar
  // gave, without losing the continuous sense of how far is left.
  tickRow: {
    flexDirection: 'row',
    justifyContent: 'space-evenly',
    alignItems: 'center',
    height: '100%',
  },
  tick: {
    width: 1,
    height: '100%',
    backgroundColor: theme.colors.paper,
    opacity: 0.9,
  },

  encouragement: {
    borderLeftWidth: 2,
    borderLeftColor: theme.colors.ink.DEFAULT,
    paddingLeft: theme.spacing[3],
    marginBottom: theme.spacing[5],
  },
  encouragementText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: theme.typography.size.bodySm.size * 1.45,
    color: theme.colors.ink.soft,
  },

  scroll: { flex: 1 },
  scrollContent: {
    paddingHorizontal: theme.spacing[6],
    paddingBottom: theme.spacing[8],
  },
  eyebrow: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    letterSpacing: 0.6,
    textTransform: 'uppercase',
    marginTop: theme.spacing[2],
  },
  title: {
    fontFamily: 'Geist',
    fontSize: theme.typography.size.h1.size,
    lineHeight:
      theme.typography.size.h1.size * theme.typography.size.h1.lineHeight,
    letterSpacing: theme.typography.size.h1.letterSpacing,
    color: theme.colors.ink.DEFAULT,
    marginBottom: theme.spacing[1],
    marginTop: theme.spacing[2],
  },
  subtitle: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    lineHeight:
      theme.typography.size.body.size * theme.typography.size.body.lineHeight,
    color: theme.colors.ink.muted,
    marginBottom: theme.spacing[5],
  },
  // The encouragement block carries the gap when present, so the subtitle
  // tightens rather than leaving a double space.
  subtitleTight: { marginBottom: theme.spacing[3] },
  form: { gap: theme.spacing[1] },

  ctaWrap: {
    paddingHorizontal: theme.spacing[6],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[4],
    borderTopWidth: 1,
    borderTopColor: theme.colors.mist.DEFAULT,
    backgroundColor: theme.colors.paper,
  },
});
