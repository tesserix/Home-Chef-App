// Collapsible stack of active-order cards for the Home screen.
//
// One active order → a single card. Multiple → a COLLAPSED stack: the newest
// card is fully visible with the others peeking just behind it (narrower,
// offset up), so the floating overlay stays compact instead of eating the
// feed. Tapping the stack EXPANDS it into the full vertical list (each card
// then opens its order); a "Hide" handle collapses it again.

import { useState } from 'react';
import {
  LayoutAnimation,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  UIManager,
  View,
} from 'react-native';
import { customerColors } from '@homechef/mobile-shared/theme';
import type { Order } from '../../types/customer';
import { ActiveOrderCard } from './ActiveOrderCard';

// LayoutAnimation needs an opt-in on (old-arch) Android; iOS works as-is.
if (
  Platform.OS === 'android' &&
  UIManager.setLayoutAnimationEnabledExperimental
) {
  UIManager.setLayoutAnimationEnabledExperimental(true);
}

interface ActiveOrderStackProps {
  orders: Order[];
}

export function ActiveOrderStack({ orders }: ActiveOrderStackProps) {
  const [expanded, setExpanded] = useState(false);

  if (orders.length === 0) return null;
  if (orders.length === 1) return <ActiveOrderCard order={orders[0]} />;

  function toggle(next: boolean) {
    LayoutAnimation.configureNext(
      LayoutAnimation.create(
        200,
        LayoutAnimation.Types.easeInEaseOut,
        LayoutAnimation.Properties.opacity,
      ),
    );
    setExpanded(next);
  }

  if (expanded) {
    return (
      <View>
        <Pressable
          onPress={() => toggle(false)}
          accessibilityRole="button"
          accessibilityLabel="Collapse active orders"
        >
          <View style={styles.handle}>
            <Text style={styles.handleText}>Hide</Text>
          </View>
        </Pressable>
        {orders.map((o) => (
          <ActiveOrderCard key={o.id} order={o} />
        ))}
      </View>
    );
  }

  // Collapsed — front card with up to two peeking layers behind it.
  const behind = Math.min(orders.length - 1, 2);
  return (
    <Pressable
      onPress={() => toggle(true)}
      accessibilityRole="button"
      accessibilityLabel={`${orders.length} active orders. Tap to expand.`}
    >
      <View style={styles.collapsedWrap}>
        {behind >= 2 ? <View style={[styles.peek, styles.peek2]} /> : null}
        {behind >= 1 ? <View style={[styles.peek, styles.peek1]} /> : null}
        {/* Front card — tap expands the stack rather than opening the order. */}
        <ActiveOrderCard order={orders[0]} onPress={() => toggle(true)} />
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  // Room above the front card for the peeking layers to show through.
  collapsedWrap: {
    position: 'relative',
    paddingTop: 14,
  },
  // A peeking card edge behind the front card — only its top sliver shows; the
  // front card (painted last) covers the rest. Narrower than the front card so
  // the layers fan inward like a stack of cards.
  peek: {
    position: 'absolute',
    height: 48,
    borderRadius: 16,
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.06,
    shadowRadius: 6,
    elevation: 2,
  },
  peek1: {
    top: 7,
    left: 26,
    right: 26,
  },
  peek2: {
    top: 0,
    left: 34,
    right: 34,
    backgroundColor: customerColors.surface.soft,
  },
  // "Hide" handle shown above the expanded list.
  handle: {
    alignSelf: 'center',
    backgroundColor: customerColors.surface.DEFAULT,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: customerColors.hairline,
    borderRadius: 9999,
    paddingHorizontal: 16,
    paddingVertical: 5,
    marginBottom: 8,
  },
  handleText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 12,
    color: customerColors.charcoal.soft,
  },
});
