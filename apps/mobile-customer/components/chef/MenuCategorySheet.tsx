import { Modal, Pressable, ScrollView, Text, View } from 'react-native';

import { customerColors } from '@homechef/mobile-shared/theme';

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
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable
        className="flex-1 justify-end"
        style={{ backgroundColor: '#00000066' }}
        onPress={onClose}
        accessibilityRole="button"
        accessibilityLabel="Close menu"
      >
        <Pressable
          className="rounded-t-2xl"
          style={{ backgroundColor: customerColors.canvas, maxHeight: '80%' }}
          onPress={(e) => e.stopPropagation()}
        >
          <View className="items-center pb-3 pt-4">
            <Text
              className="text-[18px] font-semibold"
              style={{ color: customerColors.charcoal.DEFAULT }}
            >
              Menu
            </Text>
            {hours ? (
              <Text className="mt-0.5 text-[14px]" style={{ color: customerColors.charcoal.soft }}>
                {hours}
              </Text>
            ) : null}
          </View>

          <ScrollView bounces={false}>
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

          <View className="px-5 pb-9 pt-3">
            <Pressable
              onPress={onClose}
              className="min-h-[48px] items-center justify-center rounded-xl"
              style={{ backgroundColor: customerColors.surface.soft }}
              accessibilityRole="button"
              accessibilityLabel="Close menu"
            >
              <Text
                className="text-[16px] font-semibold"
                style={{ color: customerColors.charcoal.DEFAULT }}
              >
                Close
              </Text>
            </Pressable>
          </View>
        </Pressable>
      </Pressable>
    </Modal>
  );
}
