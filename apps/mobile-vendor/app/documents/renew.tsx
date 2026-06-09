import { useState } from 'react';
import {
  ActionSheetIOS,
  ActivityIndicator,
  Alert,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import * as ImagePicker from 'expo-image-picker';
import * as DocumentPicker from 'expo-document-picker';
import { theme } from '@homechef/mobile-shared/theme';
import { useToast } from '@homechef/mobile-shared/ui';
import { multipartConfig } from '@homechef/mobile-shared/api';
import { api } from '../../lib/api';
import { describeDocumentType } from '../../hooks/useExpiringDocuments';

// Mirrors apps/api/models/document.go ChefDocumentResponse — the
// camelCase JSON keys the handler emits, NOT the Go struct names.
interface ChefDocument {
  id: string;
  type: string;
  fileName: string;
  fileUrl?: string;
  status: 'pending' | 'verified' | 'rejected';
  rejectionReason?: string;
  expiryDate?: string | null;
  createdAt: string;
}

// Which doc types are accepted as photos vs PDFs — keeps the action
// sheet relevant. Mirrors models.IsPhotoDoc in the Go backend.
function isPhotoDoc(type: string): boolean {
  return type.startsWith('kitchen_photo_') || type === 'profile_image';
}

function useChefDocuments() {
  return useQuery<ChefDocument[]>({
    queryKey: ['chef', 'documents'],
    queryFn: () => api.get<ChefDocument[]>('/chef/documents').then((r) => r.data),
    staleTime: 30_000,
  });
}

interface ReplaceArgs {
  docId: string;
  uri: string;
  mimeType: string;
  filename: string;
  expiryDate?: string;
}

function useReplaceDocument() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (args: ReplaceArgs) => {
      const form = new FormData();
      form.append('file', {
        uri: args.uri,
        name: args.filename,
        type: args.mimeType,
      } as unknown as Blob);
      if (args.expiryDate) form.append('expiryDate', args.expiryDate);
      return api.post(
        `/chef/documents/${args.docId}/replace`,
        form,
        multipartConfig(),
      );
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chef', 'documents'] });
      qc.invalidateQueries({ queryKey: ['chef', 'documents', 'expiring'] });
    },
  });
}

function StatusBadge({ status }: { status: ChefDocument['status'] }) {
  const palette = {
    verified: { bg: theme.colors.diet.veg, fg: theme.colors.paper, text: 'Verified' },
    pending: { bg: theme.colors.amber.DEFAULT, fg: theme.colors.ink.DEFAULT, text: 'Under review' },
    rejected: { bg: theme.colors.destructive.DEFAULT, fg: theme.colors.paper, text: 'Needs re-upload' },
  }[status] ?? { bg: theme.colors.bone, fg: theme.colors.ink.muted, text: status };
  return (
    <View style={[styles.badge, { backgroundColor: palette.bg }]}>
      <Text style={[styles.badgeText, { color: palette.fg }]}>{palette.text}</Text>
    </View>
  );
}

function expiryHint(doc: ChefDocument): { text: string; isUrgent: boolean } | null {
  if (!doc.expiryDate) return null;
  const expiry = new Date(doc.expiryDate);
  if (Number.isNaN(expiry.getTime())) return null;
  const days = Math.floor((expiry.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
  if (days < 0) return { text: `Expired ${Math.abs(days)} day${days === -1 ? '' : 's'} ago`, isUrgent: true };
  if (days === 0) return { text: 'Expires today', isUrgent: true };
  if (days <= 7) return { text: `Expires in ${days} day${days === 1 ? '' : 's'}`, isUrgent: true };
  if (days <= 30) return { text: `Expires in ${days} days`, isUrgent: false };
  return { text: `Expires ${expiry.toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })}`, isUrgent: false };
}

export default function DocumentsRenewScreen() {
  const { data: docs, isLoading, isError } = useChefDocuments();
  const replace = useReplaceDocument();
  const { show: showToast } = useToast();
  const [busyDocId, setBusyDocId] = useState<string | null>(null);

  async function pickAndUpload(doc: ChefDocument, source: 'camera' | 'gallery' | 'pdf'): Promise<void> {
    setBusyDocId(doc.id);
    try {
      let uri = '';
      let mimeType = '';
      let filename = '';

      if (source === 'camera') {
        const perm = await ImagePicker.requestCameraPermissionsAsync();
        if (!perm.granted) {
          showToast({ message: 'Camera permission denied.', tone: 'error' });
          return;
        }
        const result = await ImagePicker.launchCameraAsync({
          mediaTypes: ['images'],
          quality: 0.85,
          allowsEditing: true,
        });
        if (result.canceled || !result.assets[0]) return;
        uri = result.assets[0].uri;
        mimeType = 'image/jpeg';
        filename = uri.split('/').pop() ?? 'document.jpg';
      } else if (source === 'gallery') {
        const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
        if (!perm.granted) {
          showToast({ message: 'Gallery permission denied.', tone: 'error' });
          return;
        }
        const result = await ImagePicker.launchImageLibraryAsync({
          mediaTypes: ['images'],
          quality: 0.85,
          allowsEditing: true,
        });
        if (result.canceled || !result.assets[0]) return;
        uri = result.assets[0].uri;
        mimeType = 'image/jpeg';
        filename = uri.split('/').pop() ?? 'document.jpg';
      } else {
        const result = await DocumentPicker.getDocumentAsync({
          type: ['application/pdf'],
          copyToCacheDirectory: true,
        });
        if (result.canceled || !result.assets[0]) return;
        uri = result.assets[0].uri;
        mimeType = 'application/pdf';
        filename = result.assets[0].name ?? 'document.pdf';
      }

      await replace.mutateAsync({ docId: doc.id, uri, mimeType, filename });
      showToast({
        message: `${describeDocumentType(doc.type)} re-uploaded for verification.`,
        tone: 'success',
      });
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } } | null)?.response?.data?.error ??
        (err instanceof Error ? err.message : 'Upload failed.');
      showToast({ message: msg, tone: 'error' });
    } finally {
      setBusyDocId(null);
    }
  }

  function openSourceSheet(doc: ChefDocument): void {
    const photoOnly = isPhotoDoc(doc.type);
    const options = photoOnly
      ? ['Take photo', 'Choose from gallery', 'Cancel']
      : ['Take photo', 'Choose from gallery', 'Pick PDF', 'Cancel'];
    const cancelIndex = options.length - 1;

    if (Platform.OS === 'ios') {
      ActionSheetIOS.showActionSheetWithOptions(
        { options, cancelButtonIndex: cancelIndex },
        (index) => {
          if (index === 0) pickAndUpload(doc, 'camera');
          else if (index === 1) pickAndUpload(doc, 'gallery');
          else if (!photoOnly && index === 2) pickAndUpload(doc, 'pdf');
        },
      );
      return;
    }
    Alert.alert('Replace document', `Pick a new file for ${describeDocumentType(doc.type)}.`, [
      { text: 'Camera', onPress: () => pickAndUpload(doc, 'camera') },
      { text: 'Gallery', onPress: () => pickAndUpload(doc, 'gallery') },
      ...(photoOnly
        ? []
        : [{ text: 'PDF', onPress: () => pickAndUpload(doc, 'pdf') }]),
      { text: 'Cancel', style: 'cancel' as const },
    ]);
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.commandBar}>
        <Pressable
          onPress={() => router.back()}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel="Back"
        >
          <ChevronLeft size={26} color={theme.colors.ink.DEFAULT} strokeWidth={1.75} />
        </Pressable>
        <Text style={styles.commandTitle}>Documents</Text>
      </View>

      {isLoading ? (
        <View style={styles.center}>
          <ActivityIndicator size="large" color={theme.colors.ink.DEFAULT} />
        </View>
      ) : isError ? (
        <View style={styles.center}>
          <Text style={styles.muted}>Could not load your documents.</Text>
        </View>
      ) : (docs?.length ?? 0) === 0 ? (
        <View style={styles.center}>
          <Text style={styles.muted}>You haven't uploaded any documents yet.</Text>
        </View>
      ) : (
        <ScrollView style={styles.scroll} contentContainerStyle={styles.scrollContent}>
          <Text style={styles.helperText}>
            Re-upload an expiring or expired document. The new file goes back to admins
            for verification — your kitchen stays open while they review.
          </Text>

          <View style={styles.sectionGroup}>
            {docs!.map((doc, idx) => {
              const hint = expiryHint(doc);
              const isBusy = busyDocId === doc.id;
              const isLast = idx === docs!.length - 1;
              return (
                <View
                  key={doc.id}
                  style={[styles.row, !isLast && styles.rowBorderBottom]}
                >
                  <View style={styles.rowMain}>
                    <Text style={styles.rowTitle}>{describeDocumentType(doc.type)}</Text>
                    <View style={styles.rowMeta}>
                      <StatusBadge status={doc.status} />
                      {hint && (
                        <Text
                          style={[
                            styles.rowExpiry,
                            hint.isUrgent && styles.rowExpiryUrgent,
                          ]}
                        >
                          {hint.text}
                        </Text>
                      )}
                    </View>
                    {doc.rejectionReason ? (
                      <Text style={styles.rowReject}>
                        Reason: {doc.rejectionReason}
                      </Text>
                    ) : null}
                  </View>
                  <Pressable
                    onPress={() => openSourceSheet(doc)}
                    disabled={isBusy}
                    accessibilityRole="button"
                    accessibilityLabel={`Replace ${describeDocumentType(doc.type)}`}
                  >
                    {({ pressed }) => (
                      <View
                        style={[
                          styles.replaceBtn,
                          pressed && { opacity: 0.85 },
                          isBusy && { opacity: 0.5 },
                        ]}
                      >
                        <Text style={styles.replaceLabel}>
                          {isBusy ? 'Uploading…' : 'Replace'}
                        </Text>
                      </View>
                    )}
                  </Pressable>
                </View>
              );
            })}
          </View>
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.paper },
  commandBar: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[3],
    paddingBottom: theme.spacing[3],
    gap: theme.spacing[2],
  },
  commandTitle: {
    fontFamily: 'Geist-Bold',
    fontSize: 28,
    lineHeight: 32,
    letterSpacing: -0.3,
    color: theme.colors.ink.DEFAULT,
  },

  scroll: { flex: 1 },
  scrollContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[10],
  },

  helperText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    lineHeight: 20,
    paddingBottom: theme.spacing[4],
  },

  sectionGroup: {
    borderTopWidth: StyleSheet.hairlineWidth,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: theme.spacing[3],
    gap: theme.spacing[3],
    minHeight: 56,
  },
  rowBorderBottom: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  rowMain: { flex: 1, gap: 4 },
  rowTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  rowMeta: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    marginTop: 2,
    flexWrap: 'wrap',
  },
  rowExpiry: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },
  rowExpiryUrgent: {
    color: theme.colors.destructive.DEFAULT,
    fontFamily: 'Inter-SemiBold',
  },
  rowReject: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginTop: 4,
    fontStyle: 'italic',
  },

  badge: {
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: 4,
  },
  badgeText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 0.5,
  },

  replaceBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[2],
    borderRadius: 6,
  },
  replaceLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
  },

  center: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing[6],
  },
  muted: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.muted,
    textAlign: 'center',
  },
});
