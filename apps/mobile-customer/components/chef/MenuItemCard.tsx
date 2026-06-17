import { Alert, Pressable, StyleSheet, Text, View } from 'react-native';
import { Image } from 'expo-image';
import * as Haptics from 'expo-haptics';
import { Minus, Plus, UtensilsCrossed } from 'lucide-react-native';
import { useCartStore } from '../../store/cart-store';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { MenuItem } from '../../types/customer';

interface MenuItemCardProps {
  item: MenuItem;
  chefId: string;
  chefName: string;
}

export function MenuItemCard({ item, chefId, chefName }: MenuItemCardProps) {
  // Read cart state for this item so the quantity control is reactive.
  const cartItems = useCartStore((s) => s.items);
  const updateQty = useCartStore((s) => s.updateQty);
  const cartEntry = cartItems.find((i) => i.menuItemId === item.id);
  const quantity = cartEntry?.quantity ?? 0;

  const handleAdd = () => {
    const cartItem = {
      menuItemId: item.id,
      name: item.name,
      price: item.price,
      quantity: 1,
      imageUrl: item.imageUrl,
    };

    const result = useCartStore.getState().addItem(cartItem, { id: chefId, name: chefName });

    if (result === 'cross_chef_conflict') {
      Alert.alert(
        'Replace Cart?',
        'You have items from another chef. Replace cart?',
        [
          { text: 'Cancel', style: 'cancel' },
          {
            text: 'Replace',
            style: 'destructive',
            onPress: () => {
              useCartStore.getState().clearCart();
              useCartStore.getState().addItem(cartItem, { id: chefId, name: chefName });
              void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
            },
          },
        ]
      );
    } else {
      void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    }
  };

  const handleDecrement = () => {
    if (quantity <= 1) {
      updateQty(item.id, 0); // removes the item
    } else {
      updateQty(item.id, quantity - 1);
    }
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
  };

  const handleIncrement = () => {
    if (quantity === 0) {
      handleAdd();
    } else {
      updateQty(item.id, quantity + 1);
      void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    }
  };

  return (
    // Outer wrapper: flat on white, inset hairline separator below (applied
    // in the list context via marginBottom on contentContainer — no card borders).
    <View style={styles.root} accessible={false}>
      {/* TEXT COLUMN — left side */}
      <View style={styles.textCol}>
        {/* Diet tags row (FSSAI convention — appears before name) */}
        {item.dietaryTags && item.dietaryTags.length > 0 ? (
          <View style={styles.tagRow}>
            {item.dietaryTags.map((tag) => (
              <View key={tag} style={styles.tag}>
                <Text style={styles.tagLabel}>{tag}</Text>
              </View>
            ))}
          </View>
        ) : null}

        <Text style={styles.name} numberOfLines={2}>
          {item.name}
        </Text>

        {item.description ? (
          <Text style={styles.description} numberOfLines={2}>
            {item.description}
          </Text>
        ) : null}

        {/* Per-dish rating rolled up from reviews (#145) — charcoal star per spec */}
        {item.rating != null && item.rating > 0 ? (
          <View style={styles.ratingRow}>
            <Text style={styles.ratingStar}>★</Text>
            <Text style={styles.ratingText}>
              {item.rating.toFixed(1)}
              {item.reviewCount ? ` (${item.reviewCount})` : ''}
            </Text>
          </View>
        ) : null}

        {/* Price — tabular figures per spec */}
        <Text style={styles.price}>₹{item.price.toFixed(0)}</Text>

        {/* Add / quantity control — coral, sits at bottom of text column */}
        {item.isAvailable ? (
          quantity === 0 ? (
            // Initial add button — plain wrapper to dodge iOS Pressable array bug.
            <View style={styles.addWrapper}>
              <Pressable
                onPress={handleAdd}
                accessibilityLabel={`Add ${item.name} to cart`}
                accessibilityRole="button"
              >
                {({ pressed }) => (
                  // Visual styles on inner View per iOS Pressable bug rule.
                  <View
                    style={[
                      styles.addButton,
                      pressed && styles.addButtonPressed,
                    ]}
                  >
                    <Plus
                      size={16}
                      color={customerColors.canvas}
                      strokeWidth={2.5}
                    />
                    <Text style={styles.addLabel}>Add</Text>
                  </View>
                )}
              </Pressable>
            </View>
          ) : (
            // Quantity stepper — coral outlined with +/- controls.
            <View style={styles.addWrapper}>
              <View style={styles.stepper}>
                {/* Decrement */}
                <Pressable
                  onPress={handleDecrement}
                  accessibilityLabel={`Remove one ${item.name}`}
                  accessibilityRole="button"
                  style={styles.stepperBtnWrap}
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.stepperBtn,
                        pressed && styles.stepperBtnPressed,
                      ]}
                    >
                      <Minus
                        size={14}
                        color={customerColors.coral.DEFAULT}
                        strokeWidth={2.5}
                      />
                    </View>
                  )}
                </Pressable>

                <Text style={styles.stepperQty}>{quantity}</Text>

                {/* Increment */}
                <Pressable
                  onPress={handleIncrement}
                  accessibilityLabel={`Add another ${item.name}`}
                  accessibilityRole="button"
                  style={styles.stepperBtnWrap}
                >
                  {({ pressed }) => (
                    <View
                      style={[
                        styles.stepperBtn,
                        pressed && styles.stepperBtnPressed,
                      ]}
                    >
                      <Plus
                        size={14}
                        color={customerColors.coral.DEFAULT}
                        strokeWidth={2.5}
                      />
                    </View>
                  )}
                </Pressable>
              </View>
            </View>
          )
        ) : (
          <Text style={styles.unavailable}>Unavailable</Text>
        )}
      </View>

      {/* PHOTO — right side (~88 × 88, rounded-xl) */}
      <View style={styles.photoWrap}>
        {item.imageUrl ? (
          <Image
            source={{ uri: item.imageUrl }}
            style={styles.photo}
            contentFit="cover"
            placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
            transition={200}
            accessibilityLabel={`Photo of ${item.name}`}
          />
        ) : (
          // Missing image — surface-soft placeholder + utensil glyph.
          <View style={[styles.photo, styles.photoPlaceholder]}>
            <UtensilsCrossed
              size={24}
              color={customerColors.charcoal.soft}
              strokeWidth={1.5}
            />
          </View>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  // Row: text left, photo right, full-width on white canvas.
  root: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    backgroundColor: customerColors.canvas,
    paddingVertical: 16,
    // Hairline below each row — applied via borderBottomWidth to avoid
    // card-soup appearance (spec §1: separation by hairline, not card outlines).
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
    gap: 12,
  },

  // Text column takes remaining space.
  textCol: {
    flex: 1,
    gap: 4,
  },

  // Diet tag row.
  tagRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 4,
    marginBottom: 2,
  },
  tag: {
    backgroundColor: customerColors.coral.tint,
    borderRadius: 9999,
    paddingHorizontal: 8,
    paddingVertical: 2,
  },
  tagLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 0.2,
    color: customerColors.coral.pressed,
  },

  // Item name — Inter-SemiBold charcoal (spec §2.4).
  name: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 15,
    lineHeight: 20,
    letterSpacing: 0,
    color: customerColors.charcoal.DEFAULT,
  },

  // Description — charcoal-soft, 2-line clamp.
  description: {
    fontFamily: 'Inter',
    fontSize: 13,
    lineHeight: 18,
    color: customerColors.charcoal.soft,
  },

  // Per-dish rating — charcoal star + value (spec: charcoal star, NOT gold).
  ratingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 3,
    marginTop: 4,
  },
  ratingStar: {
    fontSize: 12,
    color: customerColors.charcoal.DEFAULT,
  },
  ratingText: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },

  // Price — tabular figures, charcoal (spec: tabular for every price).
  price: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    lineHeight: 20,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
    marginTop: 2,
  },

  // ---- Add button ----
  addWrapper: {
    alignSelf: 'flex-start',
    marginTop: 8,
  },
  addButton: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 8,
    paddingHorizontal: 14,
    paddingVertical: 8,
    minHeight: 36,
  },
  addButtonPressed: {
    backgroundColor: customerColors.coral.pressed,
  },
  addLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.canvas,
  },

  // ---- Quantity stepper ----
  stepper: {
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: customerColors.coral.DEFAULT,
    borderRadius: 8,
    overflow: 'hidden',
    minHeight: 36,
  },
  stepperBtnWrap: {
    // Plain wrapper so Pressable layout is stable on iOS.
  },
  stepperBtn: {
    width: 36,
    height: 36,
    alignItems: 'center',
    justifyContent: 'center',
  },
  stepperBtnPressed: {
    backgroundColor: customerColors.coral.tint,
  },
  stepperQty: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.coral.DEFAULT,
    fontVariant: ['tabular-nums'],
    minWidth: 24,
    textAlign: 'center',
  },

  unavailable: {
    fontFamily: 'Inter',
    fontSize: 12,
    color: customerColors.charcoal.soft,
    marginTop: 6,
  },

  // ---- Photo ----
  // 88 × 88 square, rounded-xl (radius 12 per spec).
  photoWrap: {
    flexShrink: 0,
  },
  photo: {
    width: 88,
    height: 88,
    borderRadius: 12,
  },
  // Missing image — surface-soft bg + utensil glyph centred.
  photoPlaceholder: {
    backgroundColor: customerColors.surface.soft,
    alignItems: 'center',
    justifyContent: 'center',
  },
});
