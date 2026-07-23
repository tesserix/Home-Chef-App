import type { ReactNode, RefObject } from 'react';
import {
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
  scrollRef,
}: OnboardingScaffoldProps) {
  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
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
          <Text style={styles.stepLabel} accessibilityLabel={`Step ${step} of ${total}`}>
            Step {step} of {total}
          </Text>
        </View>

        {/* Progress bar — one segment per step: done = ink.soft, current = full
            ink AND a wider pill ("you are here" reads from both weight and
            width, not just position), upcoming = mist. This scaffold is
            vendor-exclusive (only vendor onboarding screens import it), so it
            stays on the monochrome ink palette rather than the retired
            persimmon accent. */}
        <View
          style={styles.progressRow}
          accessibilityRole="progressbar"
          accessibilityValue={{ min: 0, max: total, now: step }}
        >
          {Array.from({ length: total }).map((_, i) => {
            const isDone = i < step - 1;
            const isCurrent = i === step - 1;
            return (
              <View
                key={i}
                style={[
                  styles.dot,
                  isDone && styles.dotDone,
                  isCurrent && styles.dotCurrent,
                ]}
              />
            );
          })}
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
          {subtitle ? <Text style={styles.subtitle}>{subtitle}</Text> : null}
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

  progressRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[1],
    paddingHorizontal: theme.spacing[6],
    paddingBottom: theme.spacing[4],
  },
  dot: {
    flex: 1,
    height: 4,
    borderRadius: 2,
    backgroundColor: theme.colors.mist.DEFAULT,
  },
  // Done — soft ink, distinct from both the upcoming mist and the current
  // step's full ink (the vendor app carries zero persimmon, so "done" vs
  // "current" has to read from ink weight, not a second accent colour).
  dotDone: { backgroundColor: theme.colors.ink.soft },
  // Current step — full ink AND a wider pill than every other segment, so
  // "you are here" is unambiguous even at a glance.
  dotCurrent: { flex: 2.2, backgroundColor: theme.colors.ink.DEFAULT },

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
