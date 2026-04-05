// Minimal react-native mock for vitest node environment
// Screen components import View from react-native for layout only

export const View = () => null;
export const Text = () => null;
export const TextInput = () => null;
export const TouchableOpacity = () => null;
export const ScrollView = () => null;
export const StyleSheet = {
  create: (styles: Record<string, unknown>) => styles,
  flatten: (style: unknown) => style,
};
export const Platform = { OS: 'ios', select: (obj: Record<string, unknown>) => obj.ios ?? obj.default };
export const Dimensions = { get: () => ({ width: 375, height: 812 }) };
