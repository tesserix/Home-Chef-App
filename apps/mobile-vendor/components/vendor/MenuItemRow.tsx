import { Pressable, StyleSheet, Switch, Text, View } from "react-native";
import { Image } from "expo-image";
import { theme } from "@homechef/mobile-shared/theme";
import { useToast } from "@homechef/mobile-shared/ui";
import type { MenuItem } from "../../hooks/useVendorMenu";
import { useToggleAvailability } from "../../hooks/useVendorMenu";
import { DietIcon } from "./DietIcon";

interface MenuItemRowProps {
  item: MenuItem;
  onPress: () => void;
}

/**
 * A single menu item row, designed to sit inside a white v2 "group card"
 * rendered by the menu screen (UI-V2 §1/§9).
 *
 * Design language: menu items are situational awareness — they don't
 * demand an immediate decision. Rows get a 44pt thumbnail, FSSAI diet
 * icon, name + price stack, and an inline native Switch flush right for
 * one-tap availability toggling. The bottom hairline is INSET (starts
 * after the thumb) so it reads as an iOS-modern grouped list, not an
 * edge-to-edge page divider.
 *
 * Unavailable items dim to 0.55 opacity and surface a "Hidden from
 * customers" caption — visible enough to scan, demoted enough to skip.
 */
export function MenuItemRow({ item, onPress }: MenuItemRowProps) {
  const toggleMutation = useToggleAvailability();
  const { show: showToast } = useToast();
  const isDimmed = !item.isAvailable;
  const photo = item.images?.[0]?.url;

  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={`Edit ${item.name}`}
    >
      {({ pressed }) => (
        // Inner View carries the flex layout. iOS Pressable's
        // function-style `style` prop drops flexbox in some cases —
        // same trick used by every other row-pattern Pressable.
        // The wrapper also hosts the inset separator so the hairline
        // never sits on the (opacity-dimmed, press-tinted) row itself.
        <View>
          <View
            style={[
              styles.root,
              pressed && { backgroundColor: theme.colors.bone },
              isDimmed && { opacity: 0.55 },
            ]}
          >
            <View style={styles.thumb}>
              {photo ? (
                <Image
                  source={{ uri: photo }}
                  style={styles.thumbImg}
                  contentFit="cover"
                />
              ) : null}
            </View>

            <View style={styles.body}>
              <View style={styles.nameRow}>
                <DietIcon isVeg={item.isVeg} size={12} />
                <Text style={styles.name} numberOfLines={1}>
                  {item.name}
                </Text>
              </View>
              <Text style={styles.price}>₹{item.price.toFixed(0)}</Text>
              {isDimmed && (
                <Text style={styles.hiddenLabel}>Hidden from customers</Text>
              )}
            </View>

            <Switch
              value={item.isAvailable}
              onValueChange={(v) =>
                toggleMutation.mutate(
                  { itemId: item.id, isAvailable: v },
                  {
                    onError: () => {
                      showToast({
                        message: `Couldn't update ${item.name}. Try again.`,
                        tone: "error",
                      });
                    },
                  },
                )
              }
              disabled={toggleMutation.isPending}
              trackColor={{
                false: theme.colors.mist.DEFAULT,
                true: theme.colors.herb.DEFAULT,
              }}
              thumbColor={theme.colors.paper}
              ios_backgroundColor={theme.colors.mist.DEFAULT}
              accessibilityLabel={`${item.name} ${item.isAvailable ? "available" : "unavailable"}, tap to toggle`}
            />
          </View>
          <View style={styles.separator} />
        </View>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  root: {
    flexDirection: "row",
    alignItems: "center",
    gap: theme.spacing[3],
    minHeight: 56,
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
    backgroundColor: theme.colors.paper,
  },
  // Inset hairline: starts after padding + thumb + gap so the line
  // never touches the thumbnail (UI-V2 §1 grouped-list rule).
  separator: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: theme.colors.mist.DEFAULT,
    marginLeft: theme.spacing[4] + 44 + theme.spacing[3],
  },
  thumb: {
    width: 44,
    height: 44,
    borderRadius: theme.radius.DEFAULT,
    backgroundColor: theme.colors.bone,
    overflow: "hidden",
    flexShrink: 0,
  },
  thumbImg: { width: 44, height: 44 },
  body: { flex: 1 },
  nameRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: theme.spacing[2],
  },
  name: {
    flex: 1,
    fontFamily: "Inter-SemiBold",
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  price: {
    fontFamily: "Geist-Bold",
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    fontVariant: ["tabular-nums"],
    marginTop: 2,
  },
  hiddenLabel: {
    fontFamily: "Inter",
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 2,
    letterSpacing: 0.2,
  },
});
