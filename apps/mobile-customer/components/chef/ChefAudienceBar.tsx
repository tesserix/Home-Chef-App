// Like + Subscribe for the chef detail screen.
//
// Deliberately NOT the heart in the header: that one saves to Favorites, a
// curated shortlist capped at 7. A like is an uncapped public signal and a
// subscription is a standing request to hear from the kitchen — three different
// things, so three different controls.
//
// Neither control is coral: the screen's accent budget belongs to Add / Place
// order (§ one accent, used sparingly). These read as hairline pills, and only
// the filled state carries any tint.

import { Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import * as Haptics from 'expo-haptics';
import { Bell, BellRing, Heart } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { HAIRLINE } from '../../lib/hairline';

const PILL_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

export interface ChefAudienceBarProps {
  liked: boolean;
  subscribed: boolean;
  likeCount: number;
  subscriberCount: number;
  onToggleLike: () => void;
  onToggleSubscribe: () => void;
  /** Disabled while either toggle is in flight, so a double tap can't race. */
  busy?: boolean;
  chefName: string;
}

/** Counts read as "1.2k" past a thousand — a five-digit number would push the
 *  pill wider than the label it sits next to. */
function formatCount(n: number): string {
  if (n < 1000) return String(n);
  const k = n / 1000;
  return `${k >= 10 ? Math.round(k) : k.toFixed(1).replace(/\.0$/, '')}k`;
}

export function ChefAudienceBar({
  liked,
  subscribed,
  likeCount,
  subscriberCount,
  onToggleLike,
  onToggleSubscribe,
  busy = false,
  chefName,
}: ChefAudienceBarProps) {
  const press = (fn: () => void) => () => {
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    fn();
  };

  return (
    <View style={styles.row}>
      <Pressable
        onPress={press(onToggleLike)}
        disabled={busy}
        accessibilityRole="button"
        accessibilityState={{ selected: liked, disabled: busy }}
        accessibilityLabel={
          liked
            ? `Unlike ${chefName}. ${likeCount} likes`
            : `Like ${chefName}. ${likeCount} likes`
        }
        android_ripple={{ color: PILL_RIPPLE, borderless: false }}
        style={styles.pressable}
      >
        {({ pressed }) => (
          <View
            style={[
              styles.pill,
              liked && styles.pillLiked,
              pressed && Platform.OS === 'ios' && styles.pillPressed,
            ]}
          >
            <Heart
              size={16}
              strokeWidth={2}
              color={liked ? customerColors.coral.DEFAULT : customerColors.charcoal.soft}
              fill={liked ? customerColors.coral.DEFAULT : 'transparent'}
            />
            <Text style={[styles.pillText, liked && styles.pillTextLiked]}>
              {formatCount(likeCount)}
            </Text>
          </View>
        )}
      </Pressable>

      <Pressable
        onPress={press(onToggleSubscribe)}
        disabled={busy}
        accessibilityRole="button"
        accessibilityState={{ selected: subscribed, disabled: busy }}
        accessibilityLabel={
          subscribed
            ? `Unsubscribe from ${chefName}. You will stop getting their updates`
            : `Subscribe to ${chefName} for new menus, price drops and opening times`
        }
        android_ripple={{ color: PILL_RIPPLE, borderless: false }}
        style={styles.pressable}
      >
        {({ pressed }) => (
          <View
            style={[
              styles.pill,
              subscribed && styles.pillSubscribed,
              pressed && Platform.OS === 'ios' && styles.pillPressed,
            ]}
          >
            {subscribed ? (
              <BellRing size={16} strokeWidth={2} color={customerColors.charcoal.DEFAULT} />
            ) : (
              <Bell size={16} strokeWidth={2} color={customerColors.charcoal.soft} />
            )}
            <Text style={[styles.pillText, subscribed && styles.pillTextSubscribed]}>
              {subscribed ? 'Subscribed' : 'Subscribe'}
            </Text>
            {subscriberCount > 0 ? (
              <Text style={styles.pillCount}>{formatCount(subscriberCount)}</Text>
            ) : null}
          </View>
        )}
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginTop: 12,
  },
  // 44px min target (§ touch targets, customer).
  pressable: { borderRadius: 8, minHeight: 44, justifyContent: 'center' },
  pill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingHorizontal: 12,
    minHeight: 36,
    borderRadius: 8,
    borderWidth: HAIRLINE,
    borderColor: customerColors.charcoal.soft,
  },
  pillPressed: { opacity: 0.6 },
  pillLiked: { borderColor: customerColors.coral.DEFAULT },
  pillSubscribed: {
    borderColor: customerColors.charcoal.DEFAULT,
    backgroundColor: `${customerColors.charcoal.DEFAULT}0D`,
  },
  pillText: {
    fontFamily: 'Inter',
    fontSize: 14,
    fontWeight: '500',
    color: customerColors.charcoal.soft,
    // Counts sit next to a label, so they must not reflow as digits change.
    fontVariant: ['tabular-nums'],
  },
  pillTextLiked: { color: customerColors.coral.DEFAULT },
  pillTextSubscribed: { color: customerColors.charcoal.DEFAULT },
  pillCount: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
});
