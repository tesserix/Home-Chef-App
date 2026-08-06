import { useState } from 'react';
import {
  ActivityIndicator,
  FlatList,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { Image } from 'expo-image';
import * as Haptics from 'expo-haptics';
import { CakeSlice } from 'lucide-react-native';
import { useAlert } from '@homechef/mobile-shared/ui';
import { customerColors } from '@homechef/mobile-shared/theme';
import { ScreenHeader } from '../components/ScreenHeader';
import { BakerySheet } from '../components/cart/BakerySheet';
import { useBakeryOptions, useBakeryProducts, type BakeryProduct } from '../hooks/useBakery';
import { useCartStore, makeLineId } from '../store/cart-store';
import type { CartBakeryConfig, CartItem } from '../types/customer';
import { formatMoney } from '../lib/format';

const ROW_RIPPLE = `${customerColors.charcoal.DEFAULT}14`;

// Bakery browse (#1065). A bake is priced per kg, so the card quotes "from ₹x"
// and the real price is settled in the configurator — the same sheet the chef
// page uses, so one flow prices every cake in the app.
export default function BakeryScreen() {
  const params = useLocalSearchParams<{ occasion?: string; productType?: string }>();
  const { showAlert } = useAlert();

  const [productType, setProductType] = useState(params.productType ?? '');
  const [occasion, setOccasion] = useState(params.occasion ?? '');
  const [dietary, setDietary] = useState('');
  const [configuring, setConfiguring] = useState<BakeryProduct | null>(null);

  const { data: vocab } = useBakeryOptions();
  const { data, isLoading, isError, refetch } = useBakeryProducts({
    productType: productType || undefined,
    occasion: occasion || undefined,
    dietary: dietary || undefined,
  });
  const products = data?.products ?? [];

  const addToCart = (item: BakeryProduct, cartItem: CartItem) => {
    const chef = { id: item.chefId, name: item.chefName ?? 'Bakery' };
    const result = useCartStore.getState().addItem(cartItem, chef);
    if (result === 'cross_chef_conflict') {
      showAlert('Replace Cart?', 'You have items from another kitchen. Replace cart?', [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Replace',
          style: 'destructive',
          onPress: () => {
            useCartStore.getState().clearCart();
            useCartStore.getState().addItem(cartItem, chef);
            void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
          },
        },
      ]);
    } else {
      void Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    }
  };

  const handleConfirm = (
    item: BakeryProduct,
    bakery: CartBakeryConfig,
    summary: string,
    unitPrice: number,
    quantity: number,
  ) => {
    setConfiguring(null);
    addToCart(item, {
      lineId: makeLineId(item.id, undefined, bakery),
      menuItemId: item.id,
      name: item.name,
      price: unitPrice,
      quantity,
      imageUrl: item.imageUrl,
      bakery,
      bakerySummary: summary,
      bakeryLeadTimeHours: item.bakery?.leadTimeHours ?? 0,
    });
  };

  return (
    <SafeAreaView style={styles.screen} edges={['top']}>
      <ScreenHeader title="Bakery" />

      <FilterRow
        options={vocab?.productTypes ?? []}
        selected={productType}
        onSelect={setProductType}
        allLabel="All bakes"
      />
      <FilterRow
        options={vocab?.occasions ?? []}
        selected={occasion}
        onSelect={setOccasion}
        allLabel="Any occasion"
      />
      <FilterRow
        options={vocab?.diets ?? []}
        selected={dietary}
        onSelect={setDietary}
        allLabel="Any diet"
      />

      {isLoading ? (
        <View style={styles.centre}>
          <ActivityIndicator color={customerColors.coral.DEFAULT} />
        </View>
      ) : isError ? (
        <View style={styles.centre}>
          <Text style={styles.emptyTitle}>We couldn't load the bakery</Text>
          <Pressable onPress={() => void refetch()} accessibilityRole="button">
            <Text style={styles.retry}>Try again</Text>
          </Pressable>
        </View>
      ) : (
        <FlatList
          data={products}
          keyExtractor={(p) => p.id}
          contentContainerStyle={products.length === 0 ? styles.emptyList : styles.list}
          ListEmptyComponent={
            <View style={styles.centre}>
              <CakeSlice size={28} color={customerColors.charcoal.soft} />
              <Text style={styles.emptyTitle}>Nothing matches those filters</Text>
              <Text style={styles.emptyBody}>Try a different occasion or bake type.</Text>
            </View>
          }
          renderItem={({ item }) => (
            <ProductRow
              product={item}
              onConfigure={() => setConfiguring(item)}
              onOpenChef={() => router.push(`/chef/${item.chefId}`)}
            />
          )}
        />
      )}

      {configuring ? (
        <BakerySheet
          item={configuring}
          visible
          onClose={() => setConfiguring(null)}
          onConfirm={(bakery, summary, unitPrice, quantity) =>
            handleConfirm(configuring, bakery, summary, unitPrice, quantity)
          }
        />
      ) : null}
    </SafeAreaView>
  );
}

function FilterRow({
  options,
  selected,
  onSelect,
  allLabel,
}: {
  options: { value: string; label: string }[];
  selected: string;
  onSelect: (v: string) => void;
  allLabel: string;
}) {
  if (options.length === 0) return null;
  const chips = [{ value: '', label: allLabel }, ...options];
  return (
    <ScrollView
      horizontal
      showsHorizontalScrollIndicator={false}
      contentContainerStyle={styles.filterRow}
    >
      {chips.map((c) => {
        const active = c.value === selected;
        return (
          <Pressable
            key={c.value || 'all'}
            onPress={() => onSelect(c.value)}
            accessibilityRole="radio"
            accessibilityState={{ selected: active }}
            accessibilityLabel={c.label}
            android_ripple={{ color: ROW_RIPPLE, borderless: false }}
          >
            <View style={[styles.chip, active && styles.chipActive]}>
              <Text style={[styles.chipLabel, active && styles.chipLabelActive]}>{c.label}</Text>
            </View>
          </Pressable>
        );
      })}
    </ScrollView>
  );
}

function ProductRow({
  product,
  onConfigure,
  onOpenChef,
}: {
  product: BakeryProduct;
  onConfigure: () => void;
  onOpenChef: () => void;
}) {
  const spec = product.bakery;
  // Per-kg bakes quote the smallest size; a flat-priced bake quotes its price.
  const from =
    spec && spec.pricePerKg > 0
      ? spec.pricePerKg * (spec.minWeightKg > 0 ? spec.minWeightKg : 0.5)
      : product.price;

  return (
    <View style={styles.row}>
      <View style={styles.rowText}>
        <Text style={styles.name} numberOfLines={1}>
          {product.name}
        </Text>
        <Pressable onPress={onOpenChef} accessibilityRole="link">
          <Text style={styles.chefLine} numberOfLines={1}>
            {product.chefName ?? 'View bakery'}
            {product.chefCity ? ` · ${product.chefCity}` : ''}
          </Text>
        </Pressable>
        <Text style={styles.price}>
          {spec && spec.pricePerKg > 0 ? `from ${formatMoney(from)}` : formatMoney(from)}
        </Text>
        {spec && spec.leadTimeHours > 0 ? (
          <Text style={styles.lead}>Needs {spec.leadTimeHours}h notice</Text>
        ) : null}
        <Pressable
          onPress={onConfigure}
          accessibilityRole="button"
          accessibilityLabel={`Customise ${product.name}`}
          android_ripple={{ color: ROW_RIPPLE, borderless: false }}
        >
          {({ pressed }) => (
            <View style={[styles.cta, pressed && Platform.OS === 'ios' && styles.ctaPressed]}>
              <Text style={styles.ctaLabel}>Customise</Text>
            </View>
          )}
        </Pressable>
      </View>

      <View style={styles.photoWrap}>
        {product.imageUrl ? (
          <Image
            source={{ uri: product.imageUrl }}
            style={styles.photo}
            contentFit="cover"
            transition={150}
            accessibilityElementsHidden
          />
        ) : (
          <View style={[styles.photo, styles.photoPlaceholder]}>
            <CakeSlice size={20} color={customerColors.charcoal.soft} />
          </View>
        )}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: customerColors.canvas },
  filterRow: { paddingHorizontal: 16, paddingVertical: 8, gap: 8 },
  chip: {
    minHeight: 36,
    justifyContent: 'center',
    paddingHorizontal: 12,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: customerColors.hairline,
    backgroundColor: customerColors.surface.soft,
  },
  chipActive: {
    borderColor: customerColors.coral.DEFAULT,
    backgroundColor: customerColors.coral.tint,
  },
  chipLabel: { fontSize: 13, fontFamily: 'Inter-Medium', color: customerColors.charcoal.DEFAULT },
  chipLabelActive: { color: customerColors.coral.DEFAULT },
  list: { paddingBottom: 32 },
  emptyList: { flexGrow: 1, justifyContent: 'center' },
  centre: { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 6, padding: 24 },
  emptyTitle: { fontSize: 15, fontFamily: 'Inter-SemiBold', color: customerColors.charcoal.DEFAULT },
  emptyBody: { fontSize: 13, color: customerColors.charcoal.soft, textAlign: 'center' },
  retry: { fontSize: 14, fontFamily: 'Inter-SemiBold', color: customerColors.coral.DEFAULT },
  row: {
    flexDirection: 'row',
    gap: 12,
    padding: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  rowText: { flex: 1, gap: 2 },
  name: { fontSize: 15, fontFamily: 'Inter-SemiBold', color: customerColors.charcoal.DEFAULT },
  chefLine: { fontSize: 12, color: customerColors.charcoal.soft },
  price: {
    fontSize: 14,
    fontFamily: 'Inter-SemiBold',
    color: customerColors.charcoal.DEFAULT,
    fontVariant: ['tabular-nums'],
  },
  lead: { fontSize: 12, color: customerColors.charcoal.soft },
  cta: {
    alignSelf: 'flex-start',
    marginTop: 8,
    minHeight: 44,
    justifyContent: 'center',
    paddingHorizontal: 16,
    borderRadius: 12,
    backgroundColor: customerColors.coral.DEFAULT,
  },
  ctaPressed: { opacity: 0.85 },
  ctaLabel: { fontSize: 14, fontFamily: 'Inter-SemiBold', color: customerColors.canvas },
  photoWrap: { width: 96 },
  photo: { width: 96, height: 96, borderRadius: 12 },
  photoPlaceholder: {
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: customerColors.surface.soft,
  },
});
