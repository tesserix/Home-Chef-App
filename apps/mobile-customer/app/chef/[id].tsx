import { useEffect, useRef, useState } from 'react';
import { formatMoney } from '../../lib/format';
import {
  ActivityIndicator,
  Alert,
  Dimensions,
  Platform,
  Pressable,
  ScrollView,
  Share,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import Animated, {
  Easing,
  FadeInDown,
  useAnimatedStyle,
  useReducedMotion,
  useSharedValue,
  withSequence,
  withTiming,
} from 'react-native-reanimated';
import { Image } from 'expo-image';
import { SafeAreaView, useSafeAreaInsets } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft, Heart, Share2, UtensilsCrossed, ShoppingCart } from 'lucide-react-native';
import * as Haptics from 'expo-haptics';
import { customerColors, customerTheme } from '@homechef/mobile-shared/theme';
import { useChef, useChefMenu } from '../../hooks/useChefs';
import { useCustomerCoords } from '../../hooks/useCustomerCoords';
import { useChefWeeklyMenu } from '../../hooks/useMealPlans';
import { useMealChefOffer } from '../../hooks/useMealSubscription';
import { useCreateGroupOrder, type GroupType } from '../../hooks/useGroupOrder';
import { useFavorites, useToggleFavorite } from '../../hooks/useFavorites';
import { useCartStore } from '../../store/cart-store';
import {
  ChefDetailTabs,
  type ChefDetailTab,
  type ChefDetailTabKey,
} from '../../components/chef/ChefDetailTabs';
import { ChefMenuTab } from '../../components/chef/ChefMenuTab';
import { ChefWeeklyPlanTab } from '../../components/chef/ChefWeeklyPlanTab';
import { ChefReviewList } from '../../components/chef/ChefReviewList';
import { TIFFIN_ENABLED } from '../../lib/features';

// Entrance easing — ease-out-quart, matches the app-wide motion spec (§3.5).
const ENTRANCE_EASING = Easing.bezier(0.22, 1, 0.36, 1);

// Android ripple tints — translucent colours derived from existing tokens
// (never a new literal colour), matching the ChefCard `withAlpha` convention.
const OVERLAY_BTN_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;
const RATING_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

// Full-bleed photo header — ~40% of viewport per canonical spec §2.4, capped
// so it stays sane on very tall devices/phablets.
const HEADER_HEIGHT = Math.min(
  Math.round(Dimensions.get('window').height * 0.4),
  400,
);

// In-page tabs under the identity block. Weekly plan only exists while the
// tiffin flows are enabled (TIFFIN_ENABLED) — same gating as before, the
// content just lives in a tab now.
const DETAIL_TABS: ChefDetailTab[] = TIFFIN_ENABLED
  ? [
      { key: 'menu', label: 'Menu' },
      { key: 'weekly', label: 'Weekly plan' },
      { key: 'reviews', label: 'Reviews' },
    ]
  : [
      { key: 'menu', label: 'Menu' },
      { key: 'reviews', label: 'Reviews' },
    ];

// Truncate the cuisine list to one line: first couple of cuisines + "+N"
// (e.g. "North Indian, South Indian +6"). The full list added 2-3 wrapped
// lines of noise to the header.
function formatCuisines(cuisine?: string): string {
  const parts = (cuisine ?? '')
    .split('·')
    .map((p) => p.trim())
    .filter(Boolean);
  if (parts.length <= 3) return parts.join(', ');
  return `${parts.slice(0, 2).join(', ')} +${parts.length - 2}`;
}

export default function ChefDetailScreen() {
  const { id, tab } = useLocalSearchParams<{ id: string; tab?: string }>();
  const insets = useSafeAreaInsets();
  const reduceMotion = useReducedMotion();

  // Heart save scale-pop: 1 → 1.25 → 1 in 150ms, gated by reduced-motion —
  // mirrors the ChefCard heart (spec §2 item 3 / motion §5).
  const heartScale = useSharedValue(1);
  const heartAnimStyle = useAnimatedStyle(() => ({
    transform: [{ scale: heartScale.value }],
  }));

  // Customer coords let the server compute deliverableToYou (can this chef reach
  // you) so the detail screen can show delivery as pickup-only when out of range.
  const coords = useCustomerCoords();
  const { data: chefData, isLoading: chefLoading, isError: chefError } = useChef(
    id ?? '',
    coords ?? undefined,
  );
  // The route param may be a slug (SEO/universal links, #58) or a UUID. GetChef
  // resolves both; the menu endpoint takes a UUID, so use the resolved chef.id
  // once it loads (falls back to the raw param for the initial render).
  const { data: menuData, isLoading: menuLoading, isError: menuError } = useChefMenu(
    chefData?.data?.id ?? id ?? ''
  );
  const { data: favData } = useFavorites();
  const toggleFavorite = useToggleFavorite();
  const createGroup = useCreateGroupOrder();
  // Chef's published fixed weekly menu (#1) — read-only preview below the CTAs.
  const { data: weeklyMenu } = useChefWeeklyMenu(chefData?.data?.id ?? id ?? '');
  // Daily tiffin subscription offer (#283) — shown only when the chef offers one.
  const { data: mealOffer } = useMealChefOffer(chefData?.data?.id ?? id ?? '');

  // Start a group / office order (#46): pick the context, then open the hub.
  function startGroupOrder(chefId: string) {
    const start = (type: GroupType) =>
      createGroup.mutate(
        { chefId, type, splitMode: 'split' },
        {
          onSuccess: (d) => router.push(`/group-order/${d.groupOrder.id}` as never),
          onError: (err: unknown) => {
            const e = err as { response?: { status?: number; data?: { error?: string } } };
            const status = e?.response?.status;
            const serverMsg = e?.response?.data?.error;
            const msg =
              status === 503
                ? "Group orders aren't available right now. Please try again later."
                : serverMsg || "We couldn't start the group order. Please try again.";
            Alert.alert('Could not start', msg);
          },
        },
      );
    Alert.alert('Start a group order', 'Who is this for?', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Office / corporate', onPress: () => start('office') },
      { text: 'Personal group', onPress: () => start('personal') },
    ]);
  }

  const chef = chefData?.data;
  const menuItems = menuData?.data ?? [];

  // Derive unique categories preserving order of first appearance.
  const categories = Array.from(
    new Set(menuItems.map((item) => item.category ?? 'Other'))
  );

  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const activeCategory = selectedCategory ?? categories[0] ?? null;

  // Which in-page tab is showing — Menu is the landing view.
  // Initial tab is deep-linkable via ?tab= (e.g. a "see reviews" link); defaults
  // to the à-la-carte menu. Only honour tabs that exist for this chef.
  const initialTab: ChefDetailTabKey =
    (tab === 'weekly' && TIFFIN_ENABLED) || tab === 'reviews' ? tab : 'menu';
  const [activeTab, setActiveTab] = useState<ChefDetailTabKey>(initialTab);

  // The Reviews tab's content is conditionally rendered (unmounts when the
  // user switches away), so ChefReviewList remounts — and its entrance
  // stagger would replay — every time the tab is revisited. Track whether
  // it's already been revealed once so only the FIRST reveal animates.
  const reviewsRevealedRef = useRef(false);
  useEffect(() => {
    if (activeTab === 'reviews') {
      reviewsRevealedRef.current = true;
    }
  }, [activeTab]);

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
    router.push('/cart');
  };

  const handleToggleSave = () => {
    if (!chef) return;
    if (!reduceMotion) {
      heartScale.value = withSequence(
        withTiming(1.25, { duration: 75 }),
        withTiming(1, { duration: 75 }),
      );
    }
    void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    toggleFavorite.mutate({ chefId: chef.id, isFavorited: isSaved });
  };

  const handleShare = async () => {
    if (!chef) return;
    try {
      await Share.share({ message: `Check out ${chef.name} on Fe3dr!` });
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
            android_ripple={{ color: OVERLAY_BTN_RIPPLE, borderless: true, radius: 22 }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.overlayBtn,
                  pressed && Platform.OS === 'ios' && styles.overlayBtnPressed,
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
            style={StyleSheet.absoluteFill}
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
          android_ripple={{ color: OVERLAY_BTN_RIPPLE, borderless: true, radius: 22 }}
        >
          {({ pressed }) => (
            <View
              style={[
                styles.overlayBtn,
                pressed && Platform.OS === 'ios' && styles.overlayBtnPressed,
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
            android_ripple={{ color: OVERLAY_BTN_RIPPLE, borderless: true, radius: 22 }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.overlayBtn,
                  pressed && Platform.OS === 'ios' && styles.overlayBtnPressed,
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
            android_ripple={{ color: OVERLAY_BTN_RIPPLE, borderless: true, radius: 22 }}
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.overlayBtn,
                  // Saved heart = coral fill bg (spec §2.4).
                  isSaved && styles.overlayBtnSaved,
                  pressed && Platform.OS === 'ios' && styles.overlayBtnPressed,
                ]}
              >
                <Animated.View style={heartAnimStyle}>
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
                </Animated.View>
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
          {/* Chef identity block — staggered entrance, step 1 of 3 (§3.5). */}
          <Animated.View
            style={styles.identityBlock}
            entering={
              reduceMotion
                ? undefined
                : FadeInDown.delay(0).duration(250).easing(ENTRANCE_EASING)
            }
          >
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
              {/* Tap the rating to jump to the in-page Reviews tab (the
                  /chef/reviews/[id] route still exists for deep links). */}
              <Pressable
                onPress={() => setActiveTab('reviews')}
                hitSlop={6}
                accessibilityRole="button"
                accessibilityLabel={
                  chef.reviewCount === 0
                    ? 'New chef — no reviews yet, see reviews'
                    : `See ${chef.reviewCount} reviews, rated ${chef.rating.toFixed(1)} out of 5`
                }
                style={styles.ratingTap}
                android_ripple={{ color: RATING_RIPPLE }}
              >
                {chef.reviewCount === 0 ? (
                  // R1 — never render "★ 0.0 (0)". A fresh chef reads as
                  // "New" instead of a badly-rated one.
                  <View style={styles.newChip}>
                    <Text style={styles.newChipText}>New</Text>
                  </View>
                ) : (
                  <>
                    {/* Unicode star glyph: charcoal colour to match spec */}
                    <Text style={styles.star}>★</Text>
                    <Text style={styles.ratingValue}>
                      {chef.rating.toFixed(1)}
                    </Text>
                    <Text style={styles.ratingCount}>({chef.reviewCount})</Text>
                  </>
                )}
              </Pressable>

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

            {/* Cuisine — truncated to a single line ("A, B +N") */}
            <Text style={styles.cuisine} numberOfLines={1}>
              {formatCuisines(chef.cuisine)}
            </Text>

            {/* Hygiene / food-safety badge (#35): verified, non-expired FSSAI.
                Restrained text — trust signal at the point of ordering. */}
            {chef.foodSafetyBadge ? (
              <Text style={styles.foodSafe}>✓ Food safety verified</Text>
            ) : null}

            {/* Outside this chef's delivery range: they're orderable for pickup
                only. Shown only when the app knows the customer's location and
                the chef offers pickup (delivery-only, out-of-range chefs don't
                appear in discovery at all). */}
            {chef.deliverableToYou === false && chef.offersPickup ? (
              <Text style={styles.pickupOnlyNote}>
                Outside delivery area · pickup only
              </Text>
            ) : null}
          </Animated.View>

          {/* ── IN-PAGE TABS: Menu · Weekly plan · Reviews ── */}
          {/* Staggered entrance, step 2 of 3 (§3.5). */}
          <Animated.View
            entering={
              reduceMotion
                ? undefined
                : FadeInDown.delay(60).duration(250).easing(ENTRANCE_EASING)
            }
          >
            <ChefDetailTabs
              tabs={DETAIL_TABS}
              activeTab={activeTab}
              onChange={setActiveTab}
            />
          </Animated.View>

          {/* Tab content — staggered entrance, step 3 of 3 (§3.5). */}
          <Animated.View
            entering={
              reduceMotion
                ? undefined
                : FadeInDown.delay(120).duration(250).easing(ENTRANCE_EASING)
            }
          >
            {/* ── MENU TAB (default) ── */}
            {/* Category chips + à-la-carte items + the small group-order row
                (GROUP_ORDERS_ENABLED gating lives inside ChefMenuTab). */}
            {activeTab === 'menu' ? (
              <ChefMenuTab
                chefId={chef.id}
                chefName={chef.name}
                categories={categories}
                activeCategory={activeCategory}
                onSelectCategory={setSelectedCategory}
                filteredItems={filteredItems}
                menuIsEmpty={menuItems.length === 0}
                onStartGroupOrder={() => startGroupOrder(chef.id)}
              />
            ) : null}

            {/* ── WEEKLY PLAN TAB ── */}
            {/* Tiffin pre-booking + subscription + weekly menu (#196/#283/#1) —
                DEFERRED flows stay gated: the tab itself only exists while
                TIFFIN_ENABLED (see DETAIL_TABS). */}
            {activeTab === 'weekly' && TIFFIN_ENABLED ? (
              <ChefWeeklyPlanTab
                chefId={chef.id}
                mealOfferAvailable={mealOffer?.available === true}
                weeklyMenuItems={weeklyMenu?.items ?? []}
                weeklyMenuPublished={weeklyMenu?.isPublished === true}
              />
            ) : null}

            {/* ── REVIEWS TAB ── */}
            {activeTab === 'reviews' ? (
              <View style={styles.reviewsPane}>
                {/* Only animate the very first reveal — see reviewsRevealedRef. */}
                <ChefReviewList
                  chefId={chef.id}
                  animateOnMount={!reviewsRevealedRef.current}
                />
              </View>
            ) : null}
          </Animated.View>
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
              accessibilityLabel={`View cart — ${cartCount} items, ${formatMoney(cartTotal)}`}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.ctaButton,
                    pressed && styles.ctaButtonPressed,
                  ]}
                >
                  <View style={styles.ctaLeft}>
                    <ShoppingCart size={18} color={customerColors.canvas} />
                    {/* Live count — Zustand selector, updates instantly on add/remove */}
                    <View style={styles.ctaCountBadge}>
                      <Text style={styles.ctaCountText}>{cartCount}</Text>
                    </View>
                    <Text style={styles.ctaLabel}>
                      View cart
                    </Text>
                  </View>
                  <Text style={styles.ctaTotal}>
                    {formatMoney(cartTotal)}
                  </Text>
                </View>
              )}
            </Pressable>
          </View>
        </View>
      ) : null}

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
    ...StyleSheet.absoluteFill,
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
  ratingTap: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
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
    fontVariant: ['tabular-nums'],
  },
  // R1 zero-review state — surface-soft bg + charcoal-soft text (never a
  // gold/coral badge; that budget stays with the accent). Matches ChefCard.
  newChip: {
    backgroundColor: customerColors.surface.soft,
    borderRadius: 9999,
    paddingHorizontal: 8,
    paddingVertical: 3,
  },
  newChipText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
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
  // Food-safety badge (#35) — calm trust-green, matches the ChefCard badge.
  foodSafe: {
    fontFamily: 'Inter',
    fontSize: 13,
    fontWeight: '600',
    color: customerColors.success.DEFAULT,
    marginTop: 6,
  },
  pickupOnlyNote: {
    fontFamily: 'Inter',
    fontSize: 13,
    fontWeight: '600',
    color: customerColors.charcoal.soft,
    marginTop: 6,
  },

  // ── Reviews tab pane (embedded ChefReviewList) ────────────────────────────
  reviewsPane: {
    paddingHorizontal: 20,
    paddingTop: 16,
    paddingBottom: 24,
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
  ctaLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
  },
  ctaCountBadge: {
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    paddingHorizontal: 6,
    backgroundColor: customerColors.canvas,
    alignItems: 'center',
    justifyContent: 'center',
  },
  ctaCountText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: customerColors.coral.DEFAULT,
    fontVariant: ['tabular-nums'],
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
