// Uber Eats–style active-order card. Floats above the tab bar on the Home
// screen. Shows only when the customer has an order in a non-terminal status.
// Tapping navigates to the order detail screen.
//
// Design: white card, coral accent for the progress bar, charcoal text.
// Shadow[2] lifts it off the canvas without heavy visual weight.
// iOS Pressable bug: visual styles on an inner View — Pressable is a plain
// wrapper. Opacity dim on press via `pressed` render-prop state.

import { Pressable, StyleSheet, Text, View } from 'react-native';
import { formatMoney } from '../../lib/format';
import { router } from 'expo-router';
import { ChevronRight } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { Order } from '../../types/customer';
import { getStepIndex, getStepLabels, getStatusLine } from '../../lib/orderSteps';
import { StageIcon } from '../status/StageIcon';

interface ActiveOrderCardProps {
  order: Order;
  // Optional tap override. When the card is the front of a collapsed stack, the
  // tap expands the stack instead of navigating to the order. Omitted → opens
  // the order detail (default).
  onPress?: () => void;
}

// ── Sub-components ──────────────────────────────────────────────────────────

interface ProgressBarProps {
  stepIndex: number; // 0–3
  steps: readonly string[];
}

/** Four-segment slim progress bar. Filled segments = coral, pending = hairline. */
function ProgressBar({ stepIndex, steps }: ProgressBarProps) {
  return (
    <View style={pbStyles.track}>
      {steps.map((s, i) => (
        <View
          key={s}
          style={[
            pbStyles.segment,
            i <= stepIndex ? pbStyles.segmentFilled : pbStyles.segmentEmpty,
            i < steps.length - 1 && pbStyles.segmentGap,
          ]}
        />
      ))}
    </View>
  );
}

const pbStyles = StyleSheet.create({
  track: {
    flexDirection: 'row',
    height: 3,
    gap: 3,
    marginTop: 8,
  },
  segment: {
    flex: 1,
    borderRadius: 2,
    height: 3,
  },
  segmentFilled: {
    backgroundColor: customerColors.coral.DEFAULT,
  },
  segmentEmpty: {
    backgroundColor: customerColors.hairline,
  },
  segmentGap: {
    // gap prop on flexDirection row handles this — no extra style needed.
  },
});

// Step label row (aligned with the segments above).
function StepLabels({ stepIndex, steps }: ProgressBarProps) {
  return (
    <View style={slStyles.row}>
      {steps.map((label, i) => (
        <Text
          key={label}
          style={[
            slStyles.label,
            i === stepIndex ? slStyles.labelActive : slStyles.labelInactive,
          ]}
          numberOfLines={1}
        >
          {label}
        </Text>
      ))}
    </View>
  );
}

const slStyles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    marginTop: 4,
  },
  label: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 10,
    textAlign: 'center',
  },
  labelActive: {
    color: customerColors.coral.DEFAULT,
    fontFamily: 'Inter-SemiBold',
  },
  labelInactive: {
    color: customerColors.charcoal.soft,
  },
});

// ── Main card ───────────────────────────────────────────────────────────────

export function ActiveOrderCard({ order, onPress }: ActiveOrderCardProps) {
  const steps = getStepLabels(order.fulfillmentType);
  const label = getStatusLine(order.status, order.fulfillmentType);
  // Clamp -1 (pending) to 0 so the first segment reads as active.
  const step = Math.max(0, getStepIndex(order.status, order.fulfillmentType));

  const itemCount = order.items.reduce((sum, item) => sum + item.quantity, 0);

  function handlePress() {
    if (onPress) {
      onPress();
      return;
    }
    router.push(`/order/${order.id}`);
  }

  return (
    // iOS Pressable bug mitigation: no function-style returning array on style.
    // The rendered-function form with `pressed` as argument is intentional here
    // so we can dim on press — but visual flex/bg/padding is on the inner View.
    <Pressable
      onPress={handlePress}
      accessibilityRole="button"
      accessibilityLabel={`Active order${order.chef?.name ? ` from ${order.chef.name}` : ''}. ${label}. Tap to track.`}
    >
      {({ pressed }) => (
        <View style={[styles.cardOuter, pressed && styles.cardPressed]}>
          <View style={styles.cardInner}>
            {/* Top row: chef name + chevron */}
            <View style={styles.topRow}>
              <View style={styles.leftCol}>
                {/* Every active stage gets its icon, not just preparing —
                    same status→icon mapping as the detail hero and the
                    tracker. StageIcon returns null for terminal statuses, and
                    this card only renders for non-terminal orders anyway. */}
                <View style={styles.stageIcon}>
                  <StageIcon
                    status={order.status}
                    size={16}
                    color={customerColors.coral.DEFAULT}
                  />
                </View>
                <View style={styles.textCol}>
                  <Text style={styles.chefName} numberOfLines={1}>
                    {order.chef?.name ?? 'Your order'}
                  </Text>
                  <Text style={styles.statusLabel} numberOfLines={1}>
                    {label}
                  </Text>
                </View>
              </View>
              <View style={styles.rightCol}>
                <View style={styles.metaBadge}>
                  <Text style={styles.metaBadgeText}>
                    {itemCount} {itemCount === 1 ? 'item' : 'items'}{' '}
                    · {formatMoney(order.totalAmount)}
                  </Text>
                </View>
                <ChevronRight
                  size={18}
                  color={customerColors.charcoal.soft}
                  accessibilityElementsHidden
                />
              </View>
            </View>

            {/* Progress bar + step labels */}
            <ProgressBar stepIndex={step} steps={steps} />
            <StepLabels stepIndex={step} steps={steps} />
          </View>
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  // Shadow lives on the outer View — clip (overflow+radius) on inner.
  cardOuter: {
    marginHorizontal: 16,
    marginBottom: 8,
    borderRadius: 16,
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    // shadow[2] — lifted, editorial
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.10,
    shadowRadius: 12,
    elevation: 4,
  },
  cardPressed: {
    opacity: 0.94,
  },
  cardInner: {
    borderRadius: 16,
    overflow: 'hidden',
    paddingHorizontal: 16,
    paddingVertical: 14,
  },

  // Top row: left (icon + text) / right (badge + chevron)
  topRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  leftCol: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    marginRight: 8,
  },
  stageIcon: {
    marginTop: -2,
  },
  textCol: {
    flex: 1,
  },
  chefName: {
    fontFamily: 'Geist',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    letterSpacing: -0.1,
    marginBottom: 2,
  },
  statusLabel: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
  },

  rightCol: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  metaBadge: {
    backgroundColor: customerColors.surface.soft,
    borderRadius: 9999,
    paddingHorizontal: 10,
    paddingVertical: 4,
  },
  metaBadgeText: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
});
