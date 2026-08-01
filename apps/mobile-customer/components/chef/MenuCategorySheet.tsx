import { useEffect, useRef } from 'react';
import { Animated, Modal, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { customerColors } from '@homechef/mobile-shared/theme';
import { useSheetDrag } from '@homechef/mobile-shared/ui';

// MenuCategorySheet — the whole menu at a glance, with a jump.
//
// A long menu buries its later categories: the customer has to scroll past
// everything to discover there is a dessert section at all. This lists every
// category with how many items it holds, and tapping one scrolls straight to it.

export interface MenuCategoryEntry {
  name: string;
  count: number;
}

export interface MenuCategorySheetProps {
  visible: boolean;
  categories: MenuCategoryEntry[];
  /** The category currently in view, highlighted with the accent rule. */
  activeCategory: string | null;
  /** Chef's serving window, e.g. "11:30 am – 9:30 pm". Omitted when unknown. */
  hours?: string;
  onSelect: (category: string) => void;
  onClose: () => void;
}

export function MenuCategorySheet({
  visible,
  categories,
  activeCategory,
  hours,
  onSelect,
  onClose,
}: MenuCategorySheetProps) {
  const insets = useSafeAreaInsets();
  const translateY = useRef(new Animated.Value(0)).current;
  const { panHandlers } = useSheetDrag({ translateY, onDismiss: onClose });

  // Reset on OPEN, not on close: resetting while the Modal plays its slide-out
  // would snap the panel back up mid-exit. Opening from 0 every time gives the
  // same guarantee without fighting the exit animation.
  useEffect(() => {
    if (visible) translateY.setValue(0);
  }, [visible, translateY]);

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable
        className="flex-1 justify-end"
        style={{ backgroundColor: '#00000066' }}
        onPress={onClose}
        accessibilityRole="button"
        accessibilityLabel="Close menu"
      >
        {/* maxHeight lives on THIS node, not the panel below it. A percentage
            height resolves against the parent, and this wrapper is the last
            node in the chain with a resolvable one (the flex-1 backdrop). Left
            on the panel it would resolve against this auto-height wrapper and
            silently stop capping. */}
        <Animated.View style={{ maxHeight: '80%', transform: [{ translateY }] }}>
          <Pressable
            className="rounded-t-2xl"
            style={{ backgroundColor: customerColors.canvas }}
            onPress={(e) => e.stopPropagation()}
          >
            {/* The grabber doubles as the drag handle — dragging it down
                dismisses, which is what replaced the Close button (#877). */}
            <View style={styles.handleGrip} {...panHandlers}>
              <View style={styles.handle} />
            </View>

            <View className="items-center pb-3 pt-1">
              <Text
                className="text-[18px] font-semibold"
                style={{ color: customerColors.charcoal.DEFAULT }}
              >
                Menu
              </Text>
              {hours ? (
                <Text
                  className="mt-0.5 text-[14px]"
                  style={{ color: customerColors.charcoal.soft }}
                >
                  {hours}
                </Text>
              ) : null}
            </View>

            {/* The footer Close button used to supply the bottom inset. With it
                gone the last category would sit flush against the screen edge,
                so the scroll content carries the safe-area padding itself. */}
            <ScrollView
              bounces={false}
              contentContainerStyle={{ paddingBottom: insets.bottom + 12 }}
            >
              {categories.map((c) => {
                const active = c.name === activeCategory;
                return (
                  <Pressable
                    key={c.name}
                    onPress={() => onSelect(c.name)}
                    className="min-h-[56px] flex-row items-center justify-between border-t px-5"
                    style={{ borderColor: customerColors.hairline }}
                    accessibilityRole="button"
                    accessibilityLabel={`${c.name}, ${c.count} items`}
                    accessibilityState={{ selected: active }}
                  >
                    {/* The 3px accent bar marks the section in view — coral used for
                        selection, which is the one thing it is reserved for. */}
                    <View
                      className="absolute bottom-0 left-0 top-0 w-[3px]"
                      style={{
                        backgroundColor: active ? customerColors.coral.DEFAULT : 'transparent',
                      }}
                    />
                    <Text
                      className={`flex-1 text-[16px] ${active ? 'font-semibold' : ''}`}
                      style={{ color: customerColors.charcoal.DEFAULT }}
                    >
                      {c.name}
                    </Text>
                    <Text
                      className="text-[16px]"
                      style={{
                        color: customerColors.charcoal.soft,
                        fontVariant: ['tabular-nums'],
                      }}
                    >
                      {c.count}
                    </Text>
                  </Pressable>
                );
              })}
            </ScrollView>
          </Pressable>
        </Animated.View>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  handleGrip: {
    alignItems: 'center',
    paddingTop: 8,
    paddingBottom: 4,
  },
  handle: {
    width: 40,
    height: 4,
    borderRadius: 2,
    backgroundColor: customerColors.hairline,
  },
});
