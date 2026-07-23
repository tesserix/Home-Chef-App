import { Platform, Pressable, StyleSheet, Text, View } from 'react-native';
import Animated, {
  useReducedMotion,
  useSharedValue,
  withSequence,
  withTiming,
  useAnimatedStyle,
} from 'react-native-reanimated';
import { Image } from 'expo-image';
import { router } from 'expo-router';
import { Heart, UtensilsCrossed } from 'lucide-react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useFavorites, useToggleFavorite } from '../../hooks/useFavorites';
import type { Chef } from '../../types/customer';

// Android ripple tints — translucent colours derived from existing tokens
// (never a new literal colour), matching the primitive Button's `withAlpha`
// convention: card body gets an ink tint, the heart (on a dark photo scrim)
// gets a white tint.
const CARD_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const HEART_RIPPLE = `${customerColors.canvas}59`;

interface ChefCardProps {
  chef: Chef;
}

export function ChefCard({ chef }: ChefCardProps) {
  const reduceMotion = useReducedMotion();

  // Derive favorited state from the favorites list by chefId.
  const { data: favoritesData } = useFavorites();
  const isFavorited =
    (favoritesData?.data ?? []).some((entry) => entry.chefId === chef.id) ??
    false;

  const toggleFavorite = useToggleFavorite();

  // Heart scale-pop: 1 → 1.2 → 1 in 150ms, gated by useReducedMotion.
  const heartScale = useSharedValue(1);
  const heartAnimStyle = useAnimatedStyle(() => ({
    transform: [{ scale: heartScale.value }],
  }));

  function handleToggleFavorite() {
    if (!reduceMotion) {
      heartScale.value = withSequence(
        withTiming(1.25, { duration: 75 }),
        withTiming(1, { duration: 75 }),
      );
    }
    toggleFavorite.mutate({ chefId: chef.id, isFavorited });
  }

  function handlePress() {
    router.push(`/chef/${chef.id}`);
  }

  const hasImage = Boolean(chef.imageUrl);

  return (
    // Outer View: shadow lives here so it's not clipped.
    // iOS requires shadow + overflow on separate Views.
    <View style={styles.outerShadow}>
      {/* Inner clip: rounds corners + clips image without killing the shadow. */}
      <View style={styles.innerClip}>
        <Pressable
          onPress={handlePress}
          accessibilityLabel={`View ${chef.name}`}
          accessibilityRole="button"
          android_ripple={{ color: CARD_RIPPLE, borderless: false }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.card,
                pressed && Platform.OS === 'ios' && styles.cardPressed,
              ]}
            >
              {/* --- Photo area: 4:3, full-width --- */}
              <View style={styles.photoContainer}>
                {hasImage ? (
                  <Image
                    source={{ uri: chef.imageUrl }}
                    style={styles.photo}
                    contentFit="cover"
                    placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
                    transition={150}
                    accessibilityElementsHidden
                  />
                ) : (
                  // Missing image → surface-soft placeholder + utensil glyph
                  // per rubric R2 — required, not optional. No grey void, no
                  // letter-avatar substitute on a photo surface.
                  <View style={styles.photoPlaceholder}>
                    <UtensilsCrossed
                      size={32}
                      color={customerColors.charcoal.soft}
                      accessibilityElementsHidden
                    />
                  </View>
                )}

                {/* Subtle charcoal scrim for heart legibility */}
                <View style={styles.photoScrim} pointerEvents="none" />

                {/* Heart toggle — top-right over the photo.
                    Wrapped in a View with pointerEvents so its touch area
                    doesn't bubble to the outer card Pressable. */}
                <View
                  style={styles.heartTouchable}
                  // Intercept touches here so the card press doesn't fire.
                  onStartShouldSetResponder={() => true}
                >
                  <Pressable
                    onPress={handleToggleFavorite}
                    hitSlop={8}
                    accessibilityRole="togglebutton"
                    accessibilityLabel={
                      isFavorited ? `Remove ${chef.name} from saved` : `Save ${chef.name}`
                    }
                    accessibilityState={{ checked: isFavorited }}
                    android_ripple={{ color: HEART_RIPPLE, borderless: true, radius: 20 }}
                  >
                    <Animated.View style={[styles.heartButton, heartAnimStyle]}>
                      <Heart
                        size={18}
                        color={isFavorited ? customerColors.coral.DEFAULT : customerColors.canvas}
                        fill={isFavorited ? customerColors.coral.DEFAULT : 'transparent'}
                      />
                    </Animated.View>
                  </Pressable>
                </View>
              </View>

              {/* --- Info block: on white below the photo --- */}
              <View style={styles.info}>
                {/* Chef name + inline rating on the same line */}
                <View style={styles.nameRatingRow}>
                  <Text style={styles.chefName} numberOfLines={1}>
                    {chef.name}
                  </Text>
                  {chef.reviewCount === 0 ? (
                    // R1 — never render "★ 0.0 (0)". Zero reviews reads as
                    // "New" instead, so a fresh chef isn't shown as a bad one.
                    <View style={styles.newChip}>
                      <Text style={styles.newChipText}>New</Text>
                    </View>
                  ) : (
                    <View style={styles.ratingBlock}>
                      {/* Star is charcoal, NOT gold per spec */}
                      <Text style={styles.ratingStar}>★</Text>
                      <Text style={styles.ratingValue}>
                        {chef.rating.toFixed(1)}
                      </Text>
                      <Text style={styles.ratingCount}>
                        ({chef.reviewCount})
                      </Text>
                    </View>
                  )}
                </View>

                {/* Cuisine line */}
                <Text style={styles.cuisine} numberOfLines={1}>
                  {chef.cuisine}
                </Text>

                {/* Hygiene / food-safety badge (#35): verified, non-expired FSSAI.
                    Restrained text, in keeping with the card's chrome-light style. */}
                {chef.foodSafetyBadge && (
                  <Text style={styles.foodSafe} numberOfLines={1}>
                    ✓ Food safety verified
                  </Text>
                )}

                {/* One meta line: open-state dot + delivery time · min-order.
                    Open/Closed folds in here instead of its own row — tighter
                    card, same information. */}
                <View style={styles.metaRow}>
                  <View
                    style={[
                      styles.openDot,
                      chef.isOpen ? styles.openDotOpen : styles.openDotClosed,
                    ]}
                  />
                  <Text style={styles.meta} numberOfLines={1}>
                    {[
                      chef.isOpen ? 'Open' : 'Closed',
                      chef.deliveryTime,
                      chef.minimumOrder != null
                        ? `Min ₹${chef.minimumOrder}`
                        : undefined,
                    ]
                      .filter(Boolean)
                      .join(' · ')}
                  </Text>
                </View>

                {/* Outside this chef's delivery range — they're listed because
                    they offer pickup. Restrained text, matching the card's
                    chrome-light style. Only shown when the app knows the
                    customer's location (deliverableToYou is defined). */}
                {chef.deliverableToYou === false && (
                  <Text style={styles.pickupOnly} numberOfLines={1}>
                    Pickup only · outside delivery area
                  </Text>
                )}

              </View>
            </View>
          )}
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  // Shadow on the outer wrapper; overflow on the inner clip — iOS pattern.
  outerShadow: {
    flex: 1,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 6,
    elevation: 2,
  },
  innerClip: {
    flex: 1,
    borderRadius: 12, // rounded-xl
    overflow: 'hidden',
    backgroundColor: customerColors.surface.DEFAULT,
  },
  card: {
    flex: 1,
    backgroundColor: customerColors.surface.DEFAULT,
  },
  // iOS-only pressed treatment (Android gets android_ripple instead — see
  // the Pressable above). Per §3.5 motion contract: pressed scale 0.97.
  cardPressed: {
    opacity: 0.95,
    transform: [{ scale: 0.97 }],
  },

  // --- Photo --- 4:3 aspect ratio
  photoContainer: {
    width: '100%',
    aspectRatio: 4 / 3,
    position: 'relative',
  },
  photo: {
    width: '100%',
    height: '100%',
  },
  photoPlaceholder: {
    width: '100%',
    height: '100%',
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
  },
  // Subtle charcoal scrim — only enough to make the white heart legible.
  photoScrim: {
    ...StyleSheet.absoluteFill,
    backgroundColor: 'rgba(0,0,0,0.08)',
  },

  // Positioned touchable wrapper for the heart — absorbs touches so they
  // don't bubble up to the outer card Pressable. Must be absolutely placed
  // over the photo to match the heart button position.
  heartTouchable: {
    position: 'absolute',
    top: 0,
    right: 0,
    // Generous touch zone extends beyond the 32pt circle
    padding: 12,
  },

  // Heart button — white glyph on the scrim; filled coral when saved.
  heartButton: {
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: 'rgba(0,0,0,0.25)',
    alignItems: 'center',
    justifyContent: 'center',
    // Minimum 44pt touch target via hitSlop on the Pressable
  },

  // --- Info block ---
  info: {
    paddingHorizontal: 10,
    paddingTop: 8,
    paddingBottom: 10,
    gap: 3,
  },

  nameRatingRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    gap: 4,
  },
  chefName: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    letterSpacing: -0.1,
  },
  // R1 zero-review state — surface-soft bg + charcoal-soft text, never a
  // gold/coral badge (that's reserved for the accent).
  newChip: {
    flexShrink: 0,
    backgroundColor: customerColors.surface.soft,
    borderRadius: 9999,
    paddingHorizontal: 7,
    paddingVertical: 2,
  },
  newChipText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    color: customerColors.charcoal.soft,
  },
  ratingBlock: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 2,
    flexShrink: 0,
  },
  // Star is charcoal (not gold) per spec §2 point 3
  ratingStar: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: customerColors.charcoal.DEFAULT,
  },
  ratingValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  ratingCount: {
    fontFamily: 'Inter',
    fontSize: 11,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },

  cuisine: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    letterSpacing: 0,
  },
  foodSafe: {
    fontFamily: 'Inter',
    fontSize: 12,
    fontWeight: '600',
    color: customerColors.success.DEFAULT, // functional success green (token; matches the web success badge)
    marginTop: 2,
  },
  metaRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 5,
  },
  openDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
  },
  openDotOpen: {
    backgroundColor: customerColors.success.DEFAULT,
  },
  openDotClosed: {
    backgroundColor: customerColors.charcoal.soft,
    opacity: 0.4,
  },
  meta: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    letterSpacing: 0,
    fontVariant: ['tabular-nums'],
  },
  pickupOnly: {
    fontFamily: 'Inter',
    fontSize: 12,
    fontWeight: '600',
    color: customerColors.charcoal.soft,
    marginTop: 2,
  },
});
