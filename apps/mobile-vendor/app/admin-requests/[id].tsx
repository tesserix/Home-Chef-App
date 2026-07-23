import { useState } from 'react';
import {
  ActionSheetIOS,
  Alert,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft } from 'lucide-react-native';
import * as ImagePicker from 'expo-image-picker';
import * as DocumentPicker from 'expo-document-picker';
import { theme } from '@homechef/mobile-shared/theme';
import { EmptyState, Skeleton, useToast } from '@homechef/mobile-shared/ui';
import { multipartConfig } from '@homechef/mobile-shared/api';
import { api } from '../../lib/api';
import { AdminRequest, useAdminRequests } from '../../hooks/useAdminRequests';

interface AttachedFile {
  uri: string;
  name: string;
  mimeType: string;
}

// Fetches the specific request from the cached list query — no per-id
// endpoint exists today; we'd add one if list size grew, but for the
// chef's < ~20 requests at a time the list cache is fine.
function useAdminRequest(id: string): {
  request: AdminRequest | null;
  isLoading: boolean;
  isError: boolean;
  refetch: () => void;
} {
  const { data, isLoading, isError, refetch } = useAdminRequests();
  const request = (data ?? []).find((r) => r.id === id) ?? null;
  return { request, isLoading, isError, refetch };
}

function useRespondToRequest(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (response: string) =>
      api.put(`/chef/admin-requests/${id}/respond`, { response }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chef', 'admin-requests'] });
    },
  });
}

// Optional: re-upload a document as part of the response. Backend
// already accepts standalone file uploads via POST /chef/documents/:id/replace
// (for an existing doc) — for a new doc attached to a response, we POST
// /chef/documents which creates a fresh doc + approval pair, then the
// admin sees both the chef's text and the new file in their queue.
function useAttachDocumentToResponse() {
  return useMutation({
    mutationFn: async (args: { file: AttachedFile; docType: string }) => {
      const form = new FormData();
      form.append('type', args.docType);
      form.append('file', {
        uri: args.file.uri,
        name: args.file.name,
        type: args.file.mimeType,
      } as unknown as Blob);
      return api.post('/chef/documents', form, multipartConfig());
    },
  });
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString('en-IN', {
      day: 'numeric',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

export default function AdminRequestDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const requestId = typeof id === 'string' ? id : '';
  const { request, isLoading, isError, refetch } = useAdminRequest(requestId);

  const [responseText, setResponseText] = useState('');
  const [attachedFile, setAttachedFile] = useState<AttachedFile | null>(null);

  const respond = useRespondToRequest(requestId);
  const attachDoc = useAttachDocumentToResponse();
  const { show: showToast } = useToast();

  async function pickFile(source: 'camera' | 'gallery' | 'pdf'): Promise<void> {
    try {
      if (source === 'camera') {
        const perm = await ImagePicker.requestCameraPermissionsAsync();
        if (!perm.granted) {
          showToast({ message: 'Camera permission denied.', tone: 'error' });
          return;
        }
        const result = await ImagePicker.launchCameraAsync({
          mediaTypes: ['images'],
          quality: 0.85,
        });
        if (result.canceled || !result.assets[0]) return;
        setAttachedFile({
          uri: result.assets[0].uri,
          name: result.assets[0].uri.split('/').pop() ?? 'photo.jpg',
          mimeType: 'image/jpeg',
        });
      } else if (source === 'gallery') {
        const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
        if (!perm.granted) {
          showToast({ message: 'Gallery permission denied.', tone: 'error' });
          return;
        }
        const result = await ImagePicker.launchImageLibraryAsync({
          mediaTypes: ['images'],
          quality: 0.85,
        });
        if (result.canceled || !result.assets[0]) return;
        setAttachedFile({
          uri: result.assets[0].uri,
          name: result.assets[0].uri.split('/').pop() ?? 'photo.jpg',
          mimeType: 'image/jpeg',
        });
      } else {
        const result = await DocumentPicker.getDocumentAsync({
          type: ['application/pdf'],
          copyToCacheDirectory: true,
        });
        if (result.canceled || !result.assets[0]) return;
        setAttachedFile({
          uri: result.assets[0].uri,
          name: result.assets[0].name ?? 'document.pdf',
          mimeType: 'application/pdf',
        });
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'File pick failed.';
      showToast({ message: msg, tone: 'error' });
    }
  }

  function openAttachSheet(): void {
    if (Platform.OS === 'ios') {
      ActionSheetIOS.showActionSheetWithOptions(
        {
          options: ['Take photo', 'Choose from gallery', 'Pick PDF', 'Cancel'],
          cancelButtonIndex: 3,
        },
        (index) => {
          if (index === 0) pickFile('camera');
          else if (index === 1) pickFile('gallery');
          else if (index === 2) pickFile('pdf');
        },
      );
      return;
    }
    Alert.alert('Attach a file', '', [
      { text: 'Camera', onPress: () => pickFile('camera') },
      { text: 'Gallery', onPress: () => pickFile('gallery') },
      { text: 'PDF', onPress: () => pickFile('pdf') },
      { text: 'Cancel', style: 'cancel' },
    ]);
  }

  async function handleSubmit(): Promise<void> {
    const text = responseText.trim();
    if (text.length === 0) {
      showToast({ message: 'Write a response before sending.', tone: 'error' });
      return;
    }
    try {
      // If a file is attached, upload it FIRST so the admin sees it
      // alongside the response text. We tie the attachment to the
      // existing approval via its entityType — backend already
      // associates uploaded docs with a fresh approval row, which the
      // admin sees in their queue. (For tighter linking we'd add an
      // explicit attachment endpoint later.)
      if (attachedFile && request?.entityType === 'chef_document') {
        await attachDoc.mutateAsync({
          file: attachedFile,
          docType: request.entityType,
        });
      }
      await respond.mutateAsync(text);
      showToast({ message: 'Response sent to admin.', tone: 'success' });
      router.back();
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } } | null)?.response?.data?.error ??
        (err instanceof Error ? err.message : 'Failed to send response.');
      showToast({ message: msg, tone: 'error' });
    }
  }

  if (isLoading) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.commandBar}>
          <View style={styles.backBtnSpacer} />
        </View>
        <View style={styles.skeletonStack}>
          <Skeleton height={16} style={{ width: '40%', marginBottom: theme.spacing[2] }} />
          <Skeleton height={40} style={{ marginBottom: theme.spacing[4] }} />
          <Skeleton height={140} />
        </View>
      </SafeAreaView>
    );
  }

  if (isError || !request) {
    return (
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.commandBar}>
          <Pressable
            onPress={() => router.back()}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel="Back"
            android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
          >
            {({ pressed }) => (
              <View style={pressed && Platform.OS === 'ios' ? { opacity: 0.6 } : undefined}>
                <ChevronLeft size={26} color={theme.colors.ink.DEFAULT} strokeWidth={1.75} />
              </View>
            )}
          </Pressable>
        </View>
        <EmptyState
          title="This request couldn't be loaded"
          body="Check your connection and try again."
          ctaLabel="Retry"
          onCtaPress={() => refetch()}
        />
      </SafeAreaView>
    );
  }

  const submitting = respond.isPending || attachDoc.isPending;

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <KeyboardAvoidingView
        style={{ flex: 1 }}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      >
        <View style={styles.commandBar}>
          <Pressable
            onPress={() => router.back()}
            hitSlop={8}
            accessibilityRole="button"
            accessibilityLabel="Back"
            android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: true }}
          >
            {({ pressed }) => (
              <View style={pressed && Platform.OS === 'ios' ? { opacity: 0.6 } : undefined}>
                <ChevronLeft size={26} color={theme.colors.ink.DEFAULT} strokeWidth={1.75} />
              </View>
            )}
          </Pressable>
          <Text style={styles.commandTitle} numberOfLines={1}>
            Admin request
          </Text>
        </View>

        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
        >
          <Text style={styles.requestDate}>{formatDate(request.createdAt)}</Text>
          <Text style={styles.requestTitle}>{request.title}</Text>
          {request.description ? (
            <Text style={styles.requestBody}>{request.description}</Text>
          ) : null}

          {request.adminNotes && request.status === 'info_requested' ? (
            <View style={styles.adminNotesBlock}>
              <Text style={styles.adminNotesLabel}>WHAT THE ADMIN NEEDS</Text>
              <Text style={styles.adminNotesBody}>{request.adminNotes}</Text>
            </View>
          ) : null}

          <Text style={styles.sectionLabel}>YOUR RESPONSE</Text>
          <TextInput
            value={responseText}
            onChangeText={setResponseText}
            placeholder="Type your response to the admin…"
            placeholderTextColor={theme.colors.ink.muted}
            multiline
            textAlignVertical="top"
            style={styles.responseInput}
            editable={!submitting}
            maxLength={2000}
          />

          {attachedFile ? (
            <View style={styles.attachmentRow}>
              <Text style={styles.attachmentName} numberOfLines={1}>
                {attachedFile.name}
              </Text>
              <Pressable
                onPress={() => setAttachedFile(null)}
                disabled={submitting}
                hitSlop={6}
                accessibilityRole="button"
                accessibilityLabel="Remove attachment"
                android_ripple={{ color: `${theme.colors.destructive.DEFAULT}14`, borderless: true }}
              >
                {({ pressed }) => (
                  <Text
                    style={[
                      styles.attachmentRemove,
                      pressed && Platform.OS === 'ios' && { opacity: 0.6 },
                    ]}
                  >
                    Remove
                  </Text>
                )}
              </Pressable>
            </View>
          ) : (
            <Pressable
              onPress={openAttachSheet}
              disabled={submitting}
              accessibilityRole="button"
              accessibilityLabel="Attach a file"
              android_ripple={{ color: `${theme.colors.ink.DEFAULT}14`, borderless: false }}
            >
              {({ pressed }) => (
                <View
                  style={[
                    styles.attachLinkWrap,
                    pressed && Platform.OS === 'ios' && { opacity: 0.7 },
                  ]}
                >
                  <Text style={styles.attachLinkLabel}>+ Attach a file (optional)</Text>
                </View>
              )}
            </Pressable>
          )}

          <Pressable
            onPress={handleSubmit}
            disabled={submitting || responseText.trim().length === 0}
            accessibilityRole="button"
            accessibilityLabel="Send response to admin"
            android_ripple={
              submitting || responseText.trim().length === 0
                ? undefined
                : { color: `${theme.colors.paper}30`, borderless: false }
            }
          >
            {({ pressed }) => (
              <View
                style={[
                  styles.submitBtn,
                  pressed && Platform.OS === 'ios' && { opacity: 0.85 },
                  (submitting || responseText.trim().length === 0) && {
                    opacity: 0.4,
                  },
                ]}
              >
                <Text style={styles.submitLabel}>
                  {submitting ? 'Sending…' : 'Send response'}
                </Text>
              </View>
            )}
          </Pressable>
        </ScrollView>
      </KeyboardAvoidingView>
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
    fontSize: 24,
    lineHeight: 28,
    letterSpacing: -0.2,
    color: theme.colors.ink.DEFAULT,
  },
  backBtnSpacer: { width: 26, height: 26 },
  skeletonStack: {
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[2],
  },
  scroll: { flex: 1 },
  scrollContent: {
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[10],
  },
  requestDate: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
    marginBottom: 4,
  },
  requestTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 24,
    marginBottom: theme.spacing[2],
  },
  requestBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
    lineHeight: 20,
    marginBottom: theme.spacing[4],
  },

  adminNotesBlock: {
    backgroundColor: theme.colors.bone,
    borderLeftWidth: 3,
    borderLeftColor: theme.colors.destructive.DEFAULT,
    borderRadius: 2,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    marginBottom: theme.spacing[6],
    gap: 6,
  },
  adminNotesLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 11,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
  },
  adminNotesBody: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    lineHeight: 20,
  },

  sectionLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 1.4,
    color: theme.colors.ink.muted,
    paddingTop: theme.spacing[2],
    paddingBottom: theme.spacing[2],
  },
  responseInput: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    minHeight: 140,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
    borderRadius: 6,
    paddingHorizontal: 12,
    paddingTop: 12,
    paddingBottom: 12,
    backgroundColor: theme.colors.paper,
    lineHeight: 22,
  },

  attachLinkWrap: {
    paddingVertical: theme.spacing[3],
    paddingHorizontal: theme.spacing[1],
  },
  attachLinkLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
    textDecorationLine: 'underline',
  },
  attachmentRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: theme.spacing[3],
    gap: theme.spacing[3],
  },
  attachmentName: {
    flex: 1,
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },
  attachmentRemove: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.destructive.DEFAULT,
    textDecorationLine: 'underline',
  },

  submitBtn: {
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: 8,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: theme.spacing[4],
  },
  submitLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.paper,
    letterSpacing: 0.3,
  },
});
