import { Platform, StyleSheet } from 'react-native';

// hairline.ts — one border width that actually renders on Android.
//
// `StyleSheet.hairlineWidth` is 1 physical pixel expressed in dp: at the 3x
// density of a modern phone (the Pixel 8 Pro emulator reports 480dpi) that is
// 0.333dp. iOS draws it exactly. Android does not: its border rasteriser
// resolves widths to integer dp when the shape also has a `borderRadius`, so a
// 0.333dp border is rounded to 0 on some edges and 1 on others — per edge, per
// corner, per element. The visible result is a border that appears clipped at
// the top or missing around the corners, and it changes between elements and
// between renders, which is why it reads as random.
//
// So on Android we ask for a whole dp. It is heavier than a true hairline
// (3 physical pixels at 3x), but a border that is consistently there beats one
// that is sometimes 1px and sometimes absent. iOS keeps the real hairline.
//
// Use this for EVERY bordered surface with a radius. `StyleSheet.hairlineWidth`
// is still correct for a plain 1px rule with no radius (dividers), where the
// rasteriser has no corners to resolve and draws it faithfully.
export const HAIRLINE = Platform.select({
  android: 1,
  default: StyleSheet.hairlineWidth,
}) as number;
