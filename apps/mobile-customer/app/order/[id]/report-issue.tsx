// Report an order issue (#37): reason + optional affected items + description +
// optional photo. Small/clear cases are refunded to the wallet instantly; others
// go to assisted review. Mirrors the review screen's form pattern.

import { useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Image,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { ChevronLeft, Camera, Check } from 'lucide-react-native';
import * as ImagePicker from 'expo-image-picker';
import { customerColors } from '@homechef/mobile-shared/theme';
import { useOrder } from '../../../hooks/useOrderHistory';
import { useReportIssue, type IssueReason } from '../../../hooks/useReportIssue';

const REASONS: { value: IssueReason; label: string }[] = [
  { value: 'missing_item', label: 'Missing item' },
  { value: 'wrong_item', label: 'Wrong item' },
  { value: 'damaged', label: 'Damaged / spilled' },
  { value: 'quality_issue', label: 'Quality issue' },
  { value: 'other', label: 'Something else' },
];

function money(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}

export default function ReportIssueScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { data, isLoading } = useOrder(id ?? '');
  const report = useReportIssue();

  const [reason, setReason] = useState<IssueReason | null>(null);
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [description, setDescription] = useState('');
  const [photoUri, setPhotoUri] = useState<string | undefined>();

  const order = data?.data;
  const items = order?.items ?? [];

  function toggleItem(itemId: string) {
    setSelectedItems((prev) => {
      const next = new Set(prev);
      if (next.has(itemId)) next.delete(itemId);
      else next.add(itemId);
      return next;
    });
  }

  async function pickPhoto() {
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted) {
      Alert.alert('Permission needed', 'Allow photo access to attach a photo.');
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      quality: 0.7,
    });
    if (!result.canceled && result.assets[0]) {
      setPhotoUri(result.assets[0].uri);
    }
  }

  function submit() {
    if (!reason || !id) return;
    report.mutate(
      {
        orderId: id,
        reason,
        description,
        affectedItemIds: Array.from(selectedItems),
        photoUri,
      },
      {
        onSuccess: (res) => {
          if (res.status === 'auto_refunded' && res.refundAmount > 0) {
            Alert.alert(
              'Refunded to your wallet',
              `${money(res.refundAmount)} has been added to your Fe3dr wallet. Sorry about that!`,
              [{ text: 'View wallet', onPress: () => router.replace('/wallet' as never) }, { text: 'Done', onPress: () => router.back() }],
            );
          } else {
            Alert.alert('Thanks for reporting', res.message, [{ text: 'OK', onPress: () => router.back() }]);
          }
        },
        onError: () => Alert.alert('Could not report', 'Please try again in a moment.'),
      },
    );
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={10} accessibilityLabel="Go back">
          <ChevronLeft size={26} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>Report an issue</Text>
        <View style={{ width: 26 }} />
      </View>

      {isLoading || !order ? (
        <View style={styles.centered}>
          <ActivityIndicator color={customerColors.coral.DEFAULT} />
        </View>
      ) : (
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <Text style={styles.sectionLabel}>What went wrong?</Text>
          <View style={styles.reasonWrap}>
            {REASONS.map((r) => {
              const active = reason === r.value;
              return (
                <Pressable key={r.value} onPress={() => setReason(r.value)} accessibilityRole="button">
                  <View style={[styles.reasonChip, active && styles.reasonChipActive]}>
                    <Text style={[styles.reasonText, active && styles.reasonTextActive]}>{r.label}</Text>
                  </View>
                </Pressable>
              );
            })}
          </View>

          {items.length > 0 ? (
            <>
              <Text style={styles.sectionLabel}>Which items? (optional)</Text>
              <View style={styles.itemsCard}>
                {items.map((it, i) => {
                  const itemId = it.id ?? '';
                  const sel = itemId ? selectedItems.has(itemId) : false;
                  return (
                    <Pressable
                      key={itemId || String(i)}
                      onPress={() => itemId && toggleItem(itemId)}
                      disabled={!itemId}
                      accessibilityRole="checkbox"
                      accessibilityState={{ checked: sel }}
                    >
                      <View style={[styles.itemRow, i === items.length - 1 && styles.itemRowLast]}>
                        <View style={[styles.checkbox, sel && styles.checkboxOn]}>
                          {sel ? <Check size={14} color={customerColors.canvas} strokeWidth={3} /> : null}
                        </View>
                        <Text style={styles.itemName} numberOfLines={1}>
                          {it.quantity}× {it.name}
                        </Text>
                      </View>
                    </Pressable>
                  );
                })}
              </View>
            </>
          ) : null}

          <Text style={styles.sectionLabel}>Tell us more (optional)</Text>
          <TextInput
            style={styles.textArea}
            placeholder="Describe the problem…"
            placeholderTextColor={customerColors.charcoal.soft}
            value={description}
            onChangeText={setDescription}
            multiline
            maxLength={500}
          />

          <Text style={styles.sectionLabel}>Add a photo (optional)</Text>
          {photoUri ? (
            <Pressable onPress={pickPhoto} accessibilityLabel="Change photo">
              <Image source={{ uri: photoUri }} style={styles.photo} />
            </Pressable>
          ) : (
            <Pressable onPress={pickPhoto} accessibilityRole="button" accessibilityLabel="Add a photo">
              <View style={styles.photoAdd}>
                <Camera size={22} color={customerColors.charcoal.soft} />
                <Text style={styles.photoAddText}>Add a photo</Text>
              </View>
            </Pressable>
          )}

          <Pressable onPress={submit} disabled={!reason || report.isPending} accessibilityRole="button" accessibilityLabel="Submit report">
            <View style={[styles.submit, (!reason || report.isPending) && styles.submitDisabled]}>
              {report.isPending ? (
                <ActivityIndicator color={customerColors.canvas} />
              ) : (
                <Text style={styles.submitText}>Submit report</Text>
              )}
            </View>
          </Pressable>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: customerColors.surface.soft },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  headerTitle: { fontFamily: 'Geist-Bold', fontSize: 20, color: customerColors.charcoal.DEFAULT },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  scroll: { padding: 16, paddingBottom: 40, gap: 8 },

  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 14,
    color: customerColors.charcoal.DEFAULT,
    marginTop: 12,
    marginBottom: 4,
  },
  reasonWrap: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  reasonChip: {
    borderWidth: 1,
    borderColor: customerColors.hairline,
    borderRadius: 9999,
    paddingHorizontal: 14,
    paddingVertical: 8,
    backgroundColor: customerColors.canvas,
  },
  reasonChipActive: { borderColor: customerColors.coral.DEFAULT, backgroundColor: customerColors.coral.tint },
  reasonText: { fontFamily: 'Inter', fontSize: 14, color: customerColors.charcoal.DEFAULT },
  reasonTextActive: { color: customerColors.coral.pressed, fontFamily: 'Inter-SemiBold' },

  itemsCard: { backgroundColor: customerColors.canvas, borderRadius: 12, paddingHorizontal: 14 },
  itemRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  itemRowLast: { borderBottomWidth: 0 },
  checkbox: {
    width: 22,
    height: 22,
    borderRadius: 6,
    borderWidth: 1.5,
    borderColor: customerColors.hairline,
    alignItems: 'center',
    justifyContent: 'center',
  },
  checkboxOn: { backgroundColor: customerColors.coral.DEFAULT, borderColor: customerColors.coral.DEFAULT },
  itemName: { flex: 1, fontFamily: 'Inter', fontSize: 15, color: customerColors.charcoal.DEFAULT },

  textArea: {
    minHeight: 90,
    backgroundColor: customerColors.canvas,
    borderRadius: 12,
    padding: 14,
    fontFamily: 'Inter',
    fontSize: 15,
    color: customerColors.charcoal.DEFAULT,
    textAlignVertical: 'top',
  },

  photoAdd: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    borderWidth: 1,
    borderStyle: 'dashed',
    borderColor: customerColors.hairline,
    borderRadius: 12,
    paddingVertical: 18,
    backgroundColor: customerColors.canvas,
  },
  photoAddText: { fontFamily: 'Inter-SemiBold', fontSize: 14, color: customerColors.charcoal.soft },
  photo: { width: '100%', height: 180, borderRadius: 12 },

  submit: {
    marginTop: 20,
    backgroundColor: customerColors.coral.DEFAULT,
    borderRadius: 12,
    minHeight: 52,
    alignItems: 'center',
    justifyContent: 'center',
  },
  submitDisabled: { backgroundColor: customerColors.surface.soft, borderWidth: 1, borderColor: customerColors.hairline },
  submitText: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: customerColors.canvas },
});
