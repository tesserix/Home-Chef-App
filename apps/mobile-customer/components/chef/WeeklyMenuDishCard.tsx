// WeeklyMenuDishCard — the ONE photo-forward dish card shared by the weekly-menu
// surfaces: the read-only "This week's menu" preview on the chef detail screen
// and the selectable cells on the book-meal-plan screen. Photo on top (with the
// app's standard blurhash + utensil fallback), FSSAI DietIcon meta row, name,
// muted description, tabular price. When `selectable`, tapping toggles it with a
// coral ring + check badge on the photo — logic stays in the caller; this file
// is purely presentational.

import { Pressable, StyleSheet, Text, View } from 'react-native';
import { Image } from 'expo-image';
import * as Haptics from 'expo-haptics';
import { Check, UtensilsCrossed } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { DietIcon } from '@homechef/mobile-shared/ui';
import { comboLabel } from '../../lib/combo-label';

// Same blurhash placeholder the chef-detail MenuItemCard uses — one photo
// language across the app.
const PHOTO_BLURHASH = 'L6PZfSi_.AyE_3t7t7R**0o#DgR4';

const WEEKDAY_NAMES = [
  'Sunday',
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday',
];

/** Today's weekday index (0=Sun..6=Sat) in IST — never trust the device tz. */
export function istTodayWeekday(): number {
  const name = new Date().toLocaleDateString('en-US', {
    weekday: 'long',
    timeZone: 'Asia/Kolkata',
  });
  return WEEKDAY_NAMES.indexOf(name);
}

/** Today's date as YYYY-MM-DD in IST (en-CA formats as ISO). */
export function istTodayIso(): string {
  return new Date().toLocaleDateString('en-CA', { timeZone: 'Asia/Kolkata' });
}

export interface WeeklyMenuDishCardProps {
  name: string;
  slot: 'lunch' | 'dinner';
  /** API variant value — 'veg' maps to the green FSSAI mark, anything else to non-veg. */
  variant: 'veg' | 'nonveg';
  price: number;
  imageUrl?: string;
  description?: string;
  isCombo?: boolean;
  comboComponents?: string[];
  /** When true the card is a toggle (booking); otherwise a static preview card. */
  selectable?: boolean;
  selected?: boolean;
  onPress?: () => void;
}

export function WeeklyMenuDishCard({
  name,
  slot,
  variant,
  price,
  imageUrl,
  description,
  isCombo,
  comboComponents,
  selectable = false,
  selected = false,
  onPress,
}: WeeklyMenuDishCardProps) {
  const isVeg = variant === 'veg';
  const slotLabel = slot === 'lunch' ? 'Lunch' : 'Dinner';
  const variantLabel = isVeg ? 'Veg' : 'Non-veg';
  const metaLabel = `${slotLabel} · ${variantLabel}${isCombo ? ` · ${comboLabel()}` : ''}`;
  const priceLabel = `₹${Math.round(price).toLocaleString('en-IN')}`;

  const body = (
    <>
      {/* PHOTO — the appetite carrier. Constant 2px frame border (transparent
          when idle) so selection never shifts layout. */}
      <View style={[styles.photoFrame, selected && styles.photoFrameSelected]}>
        {imageUrl ? (
          <Image
            source={{ uri: imageUrl }}
            style={styles.photo}
            contentFit="cover"
            placeholder={{ blurhash: PHOTO_BLURHASH }}
            transition={200}
            accessibilityLabel={`Photo of ${name}`}
          />
        ) : (
          <View style={[styles.photo, styles.photoPlaceholder]}>
            <UtensilsCrossed
              size={24}
              color={customerColors.charcoal.soft}
              strokeWidth={1.5}
            />
          </View>
        )}
        {selected ? (
          <View style={styles.checkBadge}>
            <Check size={13} color={customerColors.canvas} strokeWidth={3} />
          </View>
        ) : null}
      </View>

      {/* META — FSSAI mark + slot·variant(&combo) caption. */}
      <View style={styles.metaRow}>
        <DietIcon kind={isVeg ? 'veg' : 'non-veg'} size={12} />
        <Text style={styles.metaText} numberOfLines={1}>
          {metaLabel}
        </Text>
      </View>

      <Text style={styles.name} numberOfLines={2}>
        {name}
      </Text>

      {isCombo && comboComponents && comboComponents.length > 0 ? (
        <Text style={styles.description} numberOfLines={1}>
          {comboComponents.join(' · ')}
        </Text>
      ) : description ? (
        <Text style={styles.description} numberOfLines={1}>
          {description}
        </Text>
      ) : null}

      {price > 0 ? <Text style={styles.price}>{priceLabel}</Text> : null}
    </>
  );

  if (!selectable) {
    return <View style={styles.card}>{body}</View>;
  }

  const handlePress = () => {
    void Haptics.selectionAsync();
    onPress?.();
  };

  return (
    // Plain Pressable wrapper — visual + layout styles live on the inner View
    // (iOS drops flex/bg/padding returned from a Pressable style function).
    <Pressable
      onPress={handlePress}
      accessibilityRole="button"
      accessibilityState={{ selected }}
      accessibilityLabel={`${name}, ${metaLabel}, ${priceLabel}`}
    >
      {({ pressed }) => (
        <View style={[styles.card, pressed && styles.cardPressed]}>{body}</View>
      )}
    </Pressable>
  );
}

export interface WeeklyMenuDayHeaderProps {
  title: string;
  isToday?: boolean;
}

/** Day-section header shared by both surfaces: bold day + coral "Today" pill. */
export function WeeklyMenuDayHeader({ title, isToday = false }: WeeklyMenuDayHeaderProps) {
  return (
    <View style={styles.dayHeader}>
      <Text style={styles.dayTitle}>{title}</Text>
      {isToday ? (
        <View style={styles.todayPill}>
          <Text style={styles.todayPillText}>Today</Text>
        </View>
      ) : null}
    </View>
  );
}

const CARD_WIDTH = 150;
const PHOTO_HEIGHT = 110;

const styles = StyleSheet.create({
  // Flat on canvas — the photo is the card; no border soup around the text.
  card: {
    width: CARD_WIDTH,
    gap: 4,
  },
  // Opacity-only press feedback (no scale/bounce; reduced-motion safe).
  cardPressed: {
    opacity: 0.7,
  },

  // ---- Photo ----
  photoFrame: {
    borderRadius: 12,
    borderWidth: 2,
    borderColor: 'transparent',
    marginBottom: 4,
  },
  photoFrameSelected: {
    borderColor: customerColors.coral.DEFAULT,
  },
  photo: {
    width: '100%',
    height: PHOTO_HEIGHT,
    borderRadius: 10, // sits inside the 2px frame → 12 outer
  },
  photoPlaceholder: {
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
  },
  checkBadge: {
    position: 'absolute',
    top: 6,
    right: 6,
    width: 22,
    height: 22,
    borderRadius: 9999,
    backgroundColor: customerColors.coral.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
  },

  // ---- Text ----
  metaRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 5,
  },
  metaText: {
    flex: 1,
    fontFamily: 'Inter-Medium',
    fontSize: 11,
    letterSpacing: 0.2,
    color: customerColors.charcoal.soft,
  },
  name: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    lineHeight: 19,
    color: customerColors.charcoal.DEFAULT,
  },
  description: {
    fontFamily: 'Inter',
    fontSize: 12,
    lineHeight: 16,
    color: customerColors.charcoal.soft,
  },
  price: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
    marginTop: 2,
  },

  // ---- Day header ----
  dayHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 12,
  },
  dayTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
  },
  todayPill: {
    backgroundColor: customerColors.coral.tint,
    borderRadius: 9999,
    paddingHorizontal: 8,
    paddingVertical: 2,
  },
  todayPillText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 0.2,
    color: customerColors.coral.pressed,
  },
});
