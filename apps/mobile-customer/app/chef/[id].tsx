import { useRef, useState } from 'react';
import {
  ActivityIndicator,
  Dimensions,
  Pressable,
  ScrollView,
  Share,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { Image } from 'expo-image';
import { SafeAreaView, useSafeAreaInsets } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft, Heart, Share2, UtensilsCrossed } from 'lucide-react-native';
import * as Haptics from 'expo-haptics';
import BottomSheet from '@gorhom/bottom-sheet';
import { customerColors, customerTheme } from '@homechef/mobile-shared/theme';
import { useChef, useChefMenu } from '../../hooks/useChefs';
import { useFavorites, useToggleFavorite } from '../../hooks/useFavorites';
import { useCartStore } from '../../store/cart-store';
import { MenuItemCard } from '../../components/chef/MenuItemCard';
import { CartSheet } from '../../components/cart/CartSheet';

// Compact photo header — ~28% of viewport (capped at 260) so the menu shows
// higher up. 40% ate too much vertical space above the fold.
const HEADER_HEIGHT = Math.min(
  Math.round(Dimensions.get('window').height * 0.28),
  260,
);

export default function ChefDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const insets = useSafeAreaInsets();
  const cartSheetRef = useRef<BottomSheet>(null);

  const { data: chefData, isLoading: chefLoading, isError: chefError } = useChef(id ?? '');
  const { data: menuData, isLoading: menuLoading, isError: menuError } = useChefMenu(id ?? '');
  const { data: favData } = useFavorites();
  const toggleFavorite = useToggleFavorite();

  const chef = chefData?.data;
  const menuItems = menuData?.data ?? [];

  // Derive unique categories preserving order of first appearance.
  const categories = Array.from(
    new Set(menuItems.map((item) => item.category ?? 'Other'))
  );

  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const activeCategory = selectedCategory ?? categories[0] ?? null;

  const filteredItems = activeCategory
    ? menuItems.filter((item) => (item.category ?? 'Other') === activeCategory)
    : menuItems;

  // Cart derived values for the sticky CTA bar.
  const cartItems = useCartStore((s) => s.items);
  const cartTotal = useCartStore((s) => s.total());
  const cartCount = useCartStore((s) => s.totalCount());
  const hasCart = cartItems.length > 0;

  // Is this chef saved to favorites?
  const isSaved = favData?.data.some((f) => f.chefId === chef?.id) ?? false;

  const openCart = () => {
    cartSheetRef.current?.expand();
  };

  const handleToggleSave = () => {
    if (!chef) return;
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    toggleFavorite.mutate({ chefId: chef.id, isFavorited: isSaved });
  };

  const handleShare = async () => {
    if (!chef) return;
    try {
      await Share.share({ message: `Check out ${chef.name} on Home Chef!` });
    } catch {
      // Share cancelled or failed — silently ignore.
    }
  };

  // ── Loading state ────────────────────────────────────────────────────────
  if (chefLoading || menuLoading) {
    return (
      <View style={styles.centerFill}>
        <ActivityIndicator
          size="large"
          color={customerColors.coral.DEFAULT}
        />
      </View>
    );
  }

  // ── Error / not-found state ──────────────────────────────────────────────
  if (chefError || menuError || !chef) {
    return (
      <SafeAreaView style={styles.centerFill} edges={['top']}>
        {/* Floating back button even on error screen */}
        <View style={[styles.overlayBtnRow, { top: insets.top + 12 }]}>
          <Pressable
            onPress={() => router.back()}
            accessibilityRole="button"
            accessibilityLabel="Go back"
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.overlayBtn,
                  pressed && styles.overlayBtnPressed,
                ]}
              >
                <ChevronLeft
                  size={22}
                  color={customerColors.charcoal.DEFAULT}
                  strokeWidth={2}
                />
              </View>
            )}
          </Pressable>
        </View>

        <Text style={styles.errorText}>
          Failed to load chef details. Please try again.
        </Text>
      </SafeAreaView>
    );
  }

  // ── Main render ──────────────────────────────────────────────────────────
  // Architecture: the scroll content starts with a spacer equal to
  // HEADER_HEIGHT so it initially sits below the photo. As the user scrolls,
  // the white content sheet slides up over the photo. The header photo is
  // position:absolute at the top so it stays fixed while the content moves.
  //
  // Sticky bottom CTA bar floats over the scroll content — shadow on the outer
  // View, overflow:hidden on a separate inner clip View (per spec §6 gotcha).

  return (
    <View style={styles.root}>
      {/* ── FULL-BLEED PHOTO HEADER (absolute, behind scroll) ── */}
      <View style={[styles.heroContainer, { height: HEADER_HEIGHT }]}>
        {chef.imageUrl ? (
          <Image
            source={{ uri: chef.imageUrl }}
            style={StyleSheet.absoluteFillObject}
            contentFit="cover"
            placeholder={{ blurhash: 'L6PZfSi_.AyE_3t7t7R**0o#DgR4' }}
            transition={300}
            accessibilityLabel={`Photo of ${chef.name}`}
          />
        ) : (
          // Missing image: surface-soft placeholder + utensil glyph.
          <View style={styles.heroPlaceholder}>
            <UtensilsCrossed
              size={48}
              color={customerColors.charcoal.soft}
              strokeWidth={1.25}
            />
          </View>
        )}

        {/* Subtle scrim at top so overlay buttons stay legible on bright photos */}
        <View style={styles.heroScrim} />
      </View>

      {/* ── CIRCULAR OVERLAY BUTTONS (float over photo) ── */}
      {/* Shadow on the outer View; no overflow:hidden here (spec §6). */}
      <View
        style={[
          styles.overlayBtnRow,
          { top: insets.top + 12 },
        ]}
        pointerEvents="box-none"
      >
        {/* Back */}
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
        >
          {({ pressed }) => (
            <View
              style={[
                styles.overlayBtn,
                pressed && styles.overlayBtnPressed,
              ]}
            >
              <ChevronLeft
                size={22}
                color={customerColors.charcoal.DEFAULT}
                strokeWidth={2}
              />
            </View>
          )}
        </Pressable>

        {/* Share + Heart — grouped on the right */}
        <View style={styles.overlayBtnRightGroup}>
          <Pressable
            onPress={() => { void handleShare(); }}
            accessibilityRole="button"
            accessibilityLabel="Share chef"
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.overlayBtn,
                  pressed && styles.overlayBtnPressed,
                ]}
              >
                <Share2
                  size={20}
                  color={customerColors.charcoal.DEFAULT}
                  strokeWidth={2}
                />
              </View>
            )}
          </Pressable>

          <Pressable
            onPress={handleToggleSave}
            accessibilityRole="button"
            accessibilityLabel={isSaved ? 'Remove from saved' : 'Save chef'}
            accessibilityState={{ checked: isSaved }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.overlayBtn,
                  // Saved heart = coral fill bg (spec §2.4).
                  isSaved && styles.overlayBtnSaved,
                  pressed && styles.overlayBtnPressed,
                ]}
              >
                <Heart
                  size={20}
                  color={
                    isSaved
                      ? customerColors.canvas
                      : customerColors.charcoal.DEFAULT
                  }
                  fill={isSaved ? customerColors.canvas : 'transparent'}
                  strokeWidth={2}
                />
              </View>
            )}
          </Pressable>
        </View>
      </View>

      {/* ── SCROLLABLE CONTENT SHEET ── */}
      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={[
          styles.scrollContent,
          // Bottom padding = sticky bar height + safe-area bottom inset.
          { paddingBottom: 88 + insets.bottom },
        ]}
        showsVerticalScrollIndicator={false}
      >
        {/* Spacer: photo area the scroll starts behind */}
        <View style={{ height: HEADER_HEIGHT - 24 }} />

        {/* ── WHITE CONTENT SHEET (rounded top corners, overlaps photo) ── */}
        <View style={styles.contentSheet}>
          {/* Chef identity block */}
          <View style={styles.identityBlock}>
            <View style={styles.identityRow}>
              <Text style={styles.chefName} numberOfLines={1}>
                {chef.name}
              </Text>
              {/* Open / closed as small text (spec: not a loud badge) */}
              <Text
                style={[
                  styles.openStatus,
                  chef.isOpen ? styles.openStatusOpen : styles.openStatusClosed,
                ]}
              >
                {chef.isOpen ? 'Open' : 'Closed'}
              </Text>
            </View>

            {/* Rating row — star + count (charcoal star, NOT gold, per spec §2 item 3) */}
            <View style={styles.ratingRow}>
              {/* Unicode star glyph: charcoal colour to match spec */}
              <Text style={styles.star}>★</Text>
              <Text style={styles.ratingValue}>
                {chef.rating.toFixed(1)}
              </Text>
              <Text style={styles.ratingCount}>({chef.reviewCount})</Text>

              {chef.deliveryTime ? (
                <>
                  <View style={styles.metaDot} />
                  <Text style={styles.metaText}>{chef.deliveryTime}</Text>
                </>
              ) : null}

              {chef.minimumOrder != null ? (
                <>
                  <View style={styles.metaDot} />
                  <Text style={styles.metaText}>
                    Min ₹{chef.minimumOrder}
                  </Text>
                </>
              ) : null}

              {chef.deliveryFee != null ? (
                <>
                  <View style={styles.metaDot} />
                  <Text style={styles.metaText}>
                    {chef.deliveryFee === 0
                      ? 'Free delivery'
                      : `₹${chef.deliveryFee} delivery`}
                  </Text>
                </>
              ) : null}
            </View>

            {/* Cuisine */}
            <Text style={styles.cuisine}>{chef.cuisine}</Text>
          </View>

          {/* Hairline divider */}
          <View style={styles.hairline} />

          {/* ── CATEGORY CHIP ROW (Airbnb underline style, spec §2 item 2) ── */}
          {categories.length > 1 ? (
            <>
              <ScrollView
                horizontal
                showsHorizontalScrollIndicator={false}
                contentContainerStyle={styles.categoryRow}
              >
                {categories.map((cat) => (
                  <Pressable
                    key={cat}
                    onPress={() => setSelectedCategory(cat)}
                    accessibilityRole="button"
                    accessibilityLabel={`Filter by ${cat}`}
                    accessibilityState={{ selected: activeCategory === cat }}
                  >
                    {/* Inner View: visual styles here to dodge iOS Pressable bug */}
                    <View
                      style={[
                        styles.categoryChip,
                        activeCategory === cat && styles.categoryChipActive,
                      ]}
                    >
                      <Text
                        style={[
                          styles.categoryChipLabel,
                          activeCategory === cat && styles.categoryChipLabelActive,
                        ]}
                      >
                        {cat}
                      </Text>
                      {/* 2px underline for selected (Airbnb category bar) */}
                      {activeCategory === cat ? (
                        <View style={styles.categoryChipUnderline} />
                      ) : null}
                    </View>
                  </Pressable>
                ))}
              </ScrollView>
              <View style={styles.hairline} />
            </>
          ) : null}

          {/* ── MENU ITEMS ── */}
          {filteredItems.length === 0 ? (
            <View style={styles.emptyMenu}>
              <Text style={styles.emptyMenuText}>No items in this category</Text>
            </View>
          ) : (
            <View style={styles.menuList}>
              {filteredItems.map((item) => (
                <MenuItemCard
                  key={item.id}
                  item={item}
                  chefId={chef.id}
                  chefName={chef.name}
                />
              ))}
            </View>
          )}
        </View>
      </ScrollView>

      {/* ── STICKY BOTTOM CTA BAR ── */}
      {/* Spec §2.4: white, top hairline + shadow[2], coral filled button. */}
      {/* Shadow MUST live on the outer View; overflow:hidden on the inner clip. */}
      {hasCart ? (
        <View
          style={[
            styles.ctaBarOuter,
            { paddingBottom: insets.bottom > 0 ? insets.bottom : 16 },
          ]}
        >
          {/* Inner clip keeps radius clean without killing the outer shadow */}
          <View style={styles.ctaBarInner}>
            <Pressable
              onPress={openCart}
              accessibilityRole="button"
              accessibilityLabel={`View cart — ${cartCount} items, ₹${cartTotal.toFixed(0)}`}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.ctaButton,
                    pressed && styles.ctaButtonPressed,
                  ]}
                >
                  <Text style={styles.ctaLabel}>
                    Add to cart
                  </Text>
                  <Text style={styles.ctaTotal}>
                    ₹{cartTotal.toFixed(0)}
                  </Text>
                </View>
              )}
            </Pressable>
          </View>
        </View>
      ) : null}

      {/* Cart bottom sheet */}
      <CartSheet ref={cartSheetRef} />
    </View>
  );
}

const styles = StyleSheet.create({
  // ── Root ──────────────────────────────────────────────────────────────────
  root: {
    flex: 1,
    backgroundColor: customerColors.canvas,
  },
  centerFill: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.canvas,
    paddingHorizontal: 24,
  },
  errorText: {
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.soft,
    textAlign: 'center',
    lineHeight: 22,
  },

  // ── Hero photo ────────────────────────────────────────────────────────────
  heroContainer: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    backgroundColor: customerColors.surface.soft,
  },
  heroPlaceholder: {
    ...StyleSheet.absoluteFillObject,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.surface.soft,
  },
  // Subtle gradient-like scrim at the top so overlay buttons stay readable.
  heroScrim: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    height: 100,
    // RN doesn't support CSS gradient inline; use a semi-transparent overlay.
    // Spec intent: keep buttons legible on bright sky/food photos.
    backgroundColor: 'rgba(0,0,0,0.08)',
  },

  // ── Overlay buttons ───────────────────────────────────────────────────────
  // Floating circular white buttons over the hero photo (spec §2.4).
  // shadow[2] on this outer row View — no overflow:hidden so shadow is visible.
  overlayBtnRow: {
    position: 'absolute',
    left: 16,
    right: 16,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    zIndex: 10,
  },
  overlayBtnRightGroup: {
    flexDirection: 'row',
    gap: 8,
  },
  // Each button: white bg, radius-full, shadow[2] (spec §1 floating).
  overlayBtn: {
    width: 44,
    height: 44,
    borderRadius: 9999,
    backgroundColor: customerColors.canvas,
    alignItems: 'center',
    justifyContent: 'center',
    // shadow[2] from customerTheme — split to avoid TS spread type issue.
    shadowColor: customerTheme.shadow[2].shadowColor,
    shadowOffset: customerTheme.shadow[2].shadowOffset,
    shadowOpacity: customerTheme.shadow[2].shadowOpacity,
    shadowRadius: customerTheme.shadow[2].shadowRadius,
    elevation: customerTheme.shadow[2].elevation,
  },
  overlayBtnPressed: {
    backgroundColor: customerColors.surface.soft,
  },
  // Saved heart: coral fill background.
  overlayBtnSaved: {
    backgroundColor: customerColors.coral.DEFAULT,
  },

  // ── Scroll ────────────────────────────────────────────────────────────────
  scrollView: {
    flex: 1,
  },
  scrollContent: {
    // No horizontal padding here — the content sheet handles it.
  },

  // ── Content sheet (white, rounded-top, slides over hero) ─────────────────
  contentSheet: {
    backgroundColor: customerColors.canvas,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    // Overlap photo by 24pt via the spacer calculation in scrollContent.
    paddingTop: 20,
    minHeight: 600, // ensures content fills the screen even for short menus
  },

  // ── Identity block ────────────────────────────────────────────────────────
  identityBlock: {
    paddingHorizontal: 20,
    paddingBottom: 16,
  },
  identityRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 8,
    marginBottom: 6,
  },
  // Chef name: Geist-Bold charcoal, ~26pt (spec §4 + §2.4).
  chefName: {
    flex: 1,
    fontFamily: 'Geist-Bold',
    fontSize: 26,
    lineHeight: 30,
    letterSpacing: -0.3,
    color: customerColors.charcoal.DEFAULT,
  },
  openStatus: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    letterSpacing: 0.1,
  },
  openStatusOpen: {
    color: customerColors.success.DEFAULT,
  },
  openStatusClosed: {
    color: customerColors.charcoal.soft,
  },

  // Rating row: charcoal star (NOT gold), tabular figures.
  ratingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    flexWrap: 'wrap',
    gap: 4,
    marginBottom: 4,
  },
  star: {
    // Charcoal star per spec §2 item 3: "star is charcoal, NOT gold".
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    lineHeight: 18,
  },
  ratingValue: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  ratingCount: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
  },
  metaDot: {
    width: 3,
    height: 3,
    borderRadius: 2,
    backgroundColor: customerColors.charcoal.soft,
    marginHorizontal: 2,
    alignSelf: 'center',
  },
  metaText: {
    fontFamily: 'Inter',
    fontSize: 13,
    color: customerColors.charcoal.soft,
    fontVariant: ['tabular-nums'],
  },
  cuisine: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
  },

  // ── Hairline divider ──────────────────────────────────────────────────────
  hairline: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: customerColors.hairline,
    marginHorizontal: 20,
  },

  // ── Category chip row (Airbnb underline style, spec §2 item 2) ───────────
  categoryRow: {
    paddingHorizontal: 20,
    paddingVertical: 0,
    gap: 0,
  },
  categoryChip: {
    paddingHorizontal: 4,
    paddingTop: 14,
    paddingBottom: 10,
    marginRight: 20,
    alignItems: 'center',
    position: 'relative',
  },
  categoryChipActive: {
    // Underline drawn as a child View (see below)
  },
  categoryChipLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.soft,
    letterSpacing: 0.1,
  },
  categoryChipLabelActive: {
    // Selected = charcoal text (spec §2 item 2).
    color: customerColors.charcoal.DEFAULT,
  },
  // 2px charcoal underline for selected chip (Airbnb category-bar style).
  categoryChipUnderline: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    height: 2,
    borderRadius: 1,
    backgroundColor: customerColors.charcoal.DEFAULT,
  },

  // ── Menu list ─────────────────────────────────────────────────────────────
  menuList: {
    paddingHorizontal: 20,
    // Last item has a hairline bottom — that is sufficient; no extra padding needed.
  },
  emptyMenu: {
    paddingVertical: 40,
    alignItems: 'center',
  },
  emptyMenuText: {
    fontFamily: 'Inter',
    fontSize: 14,
    color: customerColors.charcoal.soft,
  },

  // ── Sticky CTA bar ────────────────────────────────────────────────────────
  // Spec §2.4: white bg, top hairline + shadow[2], coral button.
  // CRITICAL: shadow lives on the outer View; overflow:hidden on the inner
  // View so the radius works without killing the outer shadow (spec §6 gotcha).
  ctaBarOuter: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    backgroundColor: customerColors.canvas,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: customerColors.hairline,
    shadowColor: customerTheme.shadow[2].shadowColor,
    shadowOffset: { width: 0, height: -2 },
    shadowOpacity: customerTheme.shadow[2].shadowOpacity,
    shadowRadius: customerTheme.shadow[2].shadowRadius,
    elevation: customerTheme.shadow[2].elevation,
    paddingTop: 12,
    paddingHorizontal: 20,
  },
  ctaBarInner: {
    // No overflow:hidden needed here since the button itself has radius 8.
  },
  ctaButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 8,
    paddingHorizontal: 20,
    minHeight: 52, // spec §3: 52pt min-height for primary CTA
  },
  ctaButtonPressed: {
    backgroundColor: customerColors.coral.pressed,
  },
  ctaLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.canvas,
    letterSpacing: 0,
  },
  ctaTotal: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 16,
    color: customerColors.canvas,
    fontVariant: ['tabular-nums'],
  },
});
