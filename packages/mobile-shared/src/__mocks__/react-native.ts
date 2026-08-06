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
export const Pressable = () => null;
export const AccessibilityInfo = {
  isReduceMotionEnabled: async () => false,
  addEventListener: () => ({ remove: () => {} }),
};
export const Easing = {
  bezier: () => (t: number) => t,
  linear: (t: number) => t,
  out: (fn: unknown) => fn,
  inOut: (fn: unknown) => fn,
  ease: (t: number) => t,
};
export const Animated = {
  View,
  Text,
  Value: class {
    constructor(public value: number) {}
    setValue(v: number) {
      this.value = v;
    }
    interpolate() {
      return this;
    }
  },
  timing: () => ({ start: (cb?: () => void) => cb?.() }),
  parallel: () => ({ start: (cb?: () => void) => cb?.() }),
  sequence: () => ({ start: (cb?: () => void) => cb?.() }),
  loop: () => ({ start: () => {}, stop: () => {} }),
};
