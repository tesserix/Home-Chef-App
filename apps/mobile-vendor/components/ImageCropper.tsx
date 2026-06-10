import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Image as RNImage,
  Modal,
  Pressable,
  StyleSheet,
  Text,
  useWindowDimensions,
  View,
} from 'react-native';
import { Image } from 'expo-image';
import { Gesture, GestureDetector } from 'react-native-gesture-handler';
import Animated, {
  runOnJS,
  useAnimatedStyle,
  useSharedValue,
} from 'react-native-reanimated';
import { theme } from '@homechef/mobile-shared/theme';

interface ImageCropperProps {
  /** Local URI of the image to crop. When null the modal is hidden. */
  uri: string | null;
  /** Target aspect ratio (width / height). Default 16:9. */
  aspect?: number;
  onCancel: () => void;
  onCropped: (uri: string) => void;
}

/**
 * Full-screen 16:9 crop modal. The chef pans + pinch-zooms the photo inside a
 * fixed-aspect frame to frame exactly what customers see; on confirm we map
 * the on-screen transform back to image pixels and crop via
 * expo-image-manipulator. The image always "covers" the frame, so there are
 * never empty bars.
 */
export function ImageCropper({
  uri,
  aspect = 16 / 9,
  onCancel,
  onCropped,
}: ImageCropperProps) {
  const { width: screenW } = useWindowDimensions();
  const frameW = screenW - theme.spacing[4] * 2;
  const frameH = frameW / aspect;

  const [natural, setNatural] = useState<{ w: number; h: number } | null>(null);
  const [busy, setBusy] = useState(false);

  // Live transform (shared with the UI thread).
  const translateX = useSharedValue(0);
  const translateY = useSharedValue(0);
  const scale = useSharedValue(1);
  // Gesture start offsets.
  const startX = useSharedValue(0);
  const startY = useSharedValue(0);
  const startScale = useSharedValue(1);

  useEffect(() => {
    if (!uri) return;
    setNatural(null);
    RNImage.getSize(
      uri,
      (w, h) => setNatural({ w, h }),
      () => setNatural({ w: frameW, h: frameH }),
    );
    translateX.value = 0;
    translateY.value = 0;
    scale.value = 1;
  }, [uri, frameW, frameH, translateX, translateY, scale]);

  // baseScale makes the image cover the frame at zoom 1.
  const baseScale = natural
    ? Math.max(frameW / natural.w, frameH / natural.h)
    : 1;

  // Clamp pan so the frame is always fully inside the (possibly zoomed) image.
  function clamp() {
    'worklet';
    if (!natural) return;
    const dispW = natural.w * baseScale * scale.value;
    const dispH = natural.h * baseScale * scale.value;
    const maxX = Math.max(0, (dispW - frameW) / 2);
    const maxY = Math.max(0, (dispH - frameH) / 2);
    translateX.value = Math.min(maxX, Math.max(-maxX, translateX.value));
    translateY.value = Math.min(maxY, Math.max(-maxY, translateY.value));
  }

  const pan = Gesture.Pan()
    .onStart(() => {
      startX.value = translateX.value;
      startY.value = translateY.value;
    })
    .onUpdate((e) => {
      translateX.value = startX.value + e.translationX;
      translateY.value = startY.value + e.translationY;
    })
    .onEnd(clamp);

  const pinch = Gesture.Pinch()
    .onStart(() => {
      startScale.value = scale.value;
    })
    .onUpdate((e) => {
      scale.value = Math.min(4, Math.max(1, startScale.value * e.scale));
    })
    .onEnd(clamp);

  const gesture = Gesture.Simultaneous(pan, pinch);

  const imageStyle = useAnimatedStyle(() => ({
    transform: [
      { translateX: translateX.value },
      { translateY: translateY.value },
      { scale: scale.value },
    ],
  }));

  async function confirm() {
    if (!uri || !natural || busy) return;
    setBusy(true);
    try {
      // Lazy-require so importing this screen never triggers the native
      // module load — on a build that hasn't linked expo-image-manipulator
      // yet, this throws here (caught below) instead of redboxing the whole
      // profile screen at import time.
      const ImageManipulator = require('expo-image-manipulator');

      const s = baseScale * scale.value;
      const dispW = natural.w * s;
      const dispH = natural.h * s;
      // Frame top-left relative to the image's top-left, in display px.
      const offX = (dispW - frameW) / 2 - translateX.value;
      const offY = (dispH - frameH) / 2 - translateY.value;
      // Back to image pixels, clamped to bounds.
      const cropX = Math.max(0, Math.min(natural.w - 1, offX / s));
      const cropY = Math.max(0, Math.min(natural.h - 1, offY / s));
      const cropW = Math.min(natural.w - cropX, frameW / s);
      const cropH = Math.min(natural.h - cropY, frameH / s);

      const result = await ImageManipulator.manipulateAsync(
        uri,
        [
          {
            crop: {
              originX: Math.round(cropX),
              originY: Math.round(cropY),
              width: Math.round(cropW),
              height: Math.round(cropH),
            },
          },
          { resize: { width: 1280 } },
        ],
        { compress: 0.85, format: ImageManipulator.SaveFormat.JPEG },
      );
      onCropped(result.uri);
    } catch {
      // Fall back to the uncropped image so the chef isn't blocked.
      onCropped(uri);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      visible={uri !== null}
      animationType="slide"
      onRequestClose={onCancel}
    >
      <View style={styles.root}>
        <Text style={styles.title}>Frame your cover</Text>
        <Text style={styles.hint}>Drag to reposition · pinch to zoom</Text>

        <View style={[styles.frame, { width: frameW, height: frameH }]}>
          {uri && natural ? (
            <GestureDetector gesture={gesture}>
              <Animated.View style={styles.fill}>
                <Animated.View style={[styles.fill, imageStyle]}>
                  <Image
                    source={{ uri }}
                    style={styles.fill}
                    contentFit="cover"
                  />
                </Animated.View>
              </Animated.View>
            </GestureDetector>
          ) : (
            <ActivityIndicator color={theme.colors.paper} />
          )}
        </View>

        <View style={styles.actions}>
          <Pressable onPress={onCancel} accessibilityRole="button" accessibilityLabel="Cancel">
            {({ pressed }) => (
              <View style={[styles.btnGhost, pressed && { opacity: 0.7 }]}>
                <Text style={styles.btnGhostLabel}>Cancel</Text>
              </View>
            )}
          </Pressable>
          <Pressable onPress={confirm} disabled={busy} accessibilityRole="button" accessibilityLabel="Use this crop">
            {({ pressed }) => (
              <View style={[styles.btn, pressed && { opacity: 0.85 }, busy && { opacity: 0.5 }]}>
                {busy ? (
                  <ActivityIndicator size="small" color={theme.colors.paper} />
                ) : (
                  <Text style={styles.btnLabel}>Use photo</Text>
                )}
              </View>
            )}
          </Pressable>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: theme.colors.ink.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    gap: theme.spacing[4],
    paddingHorizontal: theme.spacing[4],
  },
  title: {
    fontFamily: 'Geist-Bold',
    fontSize: 22,
    color: theme.colors.paper,
  },
  hint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.mist.DEFAULT,
    marginTop: -theme.spacing[2],
  },
  frame: {
    borderRadius: theme.radius.lg,
    overflow: 'hidden',
    backgroundColor: theme.colors.ink.soft,
  },
  fill: { width: '100%', height: '100%' },
  actions: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    marginTop: theme.spacing[2],
  },
  btn: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[6],
    paddingVertical: theme.spacing[3],
    minHeight: 48,
    minWidth: 140,
    alignItems: 'center',
    justifyContent: 'center',
  },
  btnLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  btnGhost: {
    paddingHorizontal: theme.spacing[5],
    paddingVertical: theme.spacing[3],
    minHeight: 48,
    alignItems: 'center',
    justifyContent: 'center',
  },
  btnGhostLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.mist.DEFAULT,
  },
});
