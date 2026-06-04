// apps/mobile-vendor/app/(onboarding)/documents.tsx
// Step 4/6 — Upload ID proof + FSSAI license.
// StyleSheet only — no NativeWind className.
// Upload slots: bone bg / mist border. When uploaded: filename row + remove ×.

import {
  ActivityIndicator,
  Alert,
  Image,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { useState } from 'react';
import * as ImagePicker from 'expo-image-picker';
import * as DocumentPicker from 'expo-document-picker';
import { router } from 'expo-router';
import { OnboardingScaffold, useToast } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { multipartConfig } from '@homechef/mobile-shared/api';
import { api } from '../../lib/api';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

type DocumentType = 'id_proof' | 'fssai_license';

interface UploadState {
  uploading: boolean;
  error: string | null;
}

// Derive a display-safe filename from a file URI.
function uriToFilename(uri: string, fileType: 'image' | 'pdf' | null): string {
  if (!fileType) return 'document';
  const segments = uri.split('/');
  const last = segments[segments.length - 1] ?? '';
  if (last.length > 0 && last.length < 50) return last;
  return fileType === 'pdf' ? 'document.pdf' : 'document.jpg';
}

export default function DocumentsScreen() {
  const { documents, updateDocuments, setStep } = useVendorOnboardingStore();
  const { show: showToast } = useToast();

  const [idUpload, setIdUpload] = useState<UploadState>({ uploading: false, error: null });
  const [fssaiUpload, setFssaiUpload] = useState<UploadState>({ uploading: false, error: null });

  function setUploadState(docType: DocumentType, state: UploadState): void {
    if (docType === 'id_proof') {
      setIdUpload(state);
    } else {
      setFssaiUpload(state);
    }
  }

  async function uploadFile(
    uri: string,
    fileType: 'image' | 'pdf',
    mimeType: string,
    docType: DocumentType,
  ): Promise<void> {
    setUploadState(docType, { uploading: true, error: null });
    try {
      const formData = new FormData();
      formData.append('type', docType);
      const filename = uri.split('/').pop() ?? (fileType === 'pdf' ? 'document.pdf' : 'document.jpg');
      formData.append('file', { uri, name: filename, type: mimeType } as unknown as Blob);

      await api.post('/chef/documents', formData, multipartConfig());

      if (docType === 'id_proof') {
        updateDocuments({ idProofUri: uri, idProofType: fileType });
      } else {
        updateDocuments({ fssaiUri: uri, fssaiType: fileType });
      }
      setUploadState(docType, { uploading: false, error: null });
      showToast({
        message: docType === 'id_proof' ? 'ID proof uploaded.' : 'FSSAI license uploaded.',
        tone: 'success',
      });
    } catch (error: unknown) {
      const serverError =
        (error as { response?: { data?: { error?: string } } } | null)?.response?.data?.error;
      const message =
        serverError ?? (error instanceof Error ? error.message : 'Upload failed. Please try again.');
      setUploadState(docType, { uploading: false, error: message });
      showToast({ message, tone: 'error' });
    }
  }

  function removeDocument(docType: DocumentType): void {
    if (docType === 'id_proof') {
      updateDocuments({ idProofUri: null, idProofType: null });
    } else {
      updateDocuments({ fssaiUri: null, fssaiType: null });
    }
  }

  async function handleCamera(docType: DocumentType): Promise<void> {
    try {
      const permission = await ImagePicker.requestCameraPermissionsAsync();
      if (!permission.granted) {
        showToast({ message: 'Camera permission denied — tap Gallery to use a saved photo.', tone: 'error' });
        return;
      }
      const result = await ImagePicker.launchCameraAsync({
        mediaTypes: ['images'],
        quality: 0.85,
        allowsEditing: true,
      });
      if (!result.canceled && result.assets[0]) {
        await uploadFile(result.assets[0].uri, 'image', 'image/jpeg', docType);
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Camera unavailable. Try Gallery.';
      showToast({ message, tone: 'error' });
    }
  }

  async function handleGallery(docType: DocumentType): Promise<void> {
    try {
      const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
      if (!permission.granted) {
        showToast({ message: 'Gallery permission denied — please allow photo access in Settings.', tone: 'error' });
        return;
      }
      const result = await ImagePicker.launchImageLibraryAsync({
        mediaTypes: ['images'],
        quality: 0.85,
        allowsEditing: true,
      });
      if (!result.canceled && result.assets[0]) {
        await uploadFile(result.assets[0].uri, 'image', 'image/jpeg', docType);
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Could not open gallery. Please try again.';
      showToast({ message, tone: 'error' });
    }
  }

  async function handlePdf(docType: DocumentType): Promise<void> {
    try {
      const result = await DocumentPicker.getDocumentAsync({
        type: ['application/pdf'],
        copyToCacheDirectory: true,
      });
      if (!result.canceled && result.assets[0]) {
        await uploadFile(result.assets[0].uri, 'pdf', 'application/pdf', docType);
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Could not open document picker. Please try again.';
      showToast({ message, tone: 'error' });
    }
  }

  function onNext(): void {
    if (!documents.idProofUri || !documents.fssaiUri) {
      Alert.alert(
        'Documents required',
        'Please upload both ID proof and FSSAI license to continue.',
      );
      return;
    }
    setStep(5);
    router.push('/(onboarding)/policies');
  }

  const bothUploaded = Boolean(documents.idProofUri && documents.fssaiUri);

  function renderUploadSlot(
    title: string,
    subtitle: string,
    docType: DocumentType,
    uri: string | null,
    fileType: 'image' | 'pdf' | null,
    uploadState: UploadState,
  ): React.ReactElement {
    const isUploaded = uri !== null;

    return (
      <View style={styles.slot}>
        {/* Slot header */}
        <View style={styles.slotHeader}>
          <Text style={styles.slotTitle}>{title}</Text>
          <Text style={styles.slotSubtitle}>{subtitle}</Text>
        </View>

        {/* Uploaded state — preview + filename + remove */}
        {isUploaded ? (
          <View style={styles.uploadedBlock}>
            {fileType === 'image' ? (
              <Image
                source={{ uri }}
                style={styles.previewImage}
                resizeMode="cover"
                accessibilityLabel={`${title} preview`}
              />
            ) : (
              <View style={styles.pdfPreview}>
                {/* Simple PDF representation — no emoji, just type indicator */}
                <Text style={styles.pdfTypeLabel}>PDF</Text>
              </View>
            )}

            {/* Filename + remove row */}
            <View style={styles.filenameRow}>
              <View style={styles.filenameDot} />
              <Text style={styles.filenameText} numberOfLines={1}>
                {uriToFilename(uri, fileType)}
              </Text>
              <Pressable
                onPress={() => removeDocument(docType)}
                hitSlop={8}
                style={({ pressed }) => [
                  styles.removeBtn,
                  pressed && styles.removeBtnPressed,
                ]}
                accessibilityLabel={`Remove ${title}`}
                accessibilityRole="button"
              >
                <Text style={styles.removeIcon}>×</Text>
              </Pressable>
            </View>
          </View>
        ) : null}

        {/* Uploading spinner */}
        {uploadState.uploading ? (
          <View style={styles.uploadingRow}>
            <ActivityIndicator
              size="small"
              color={theme.colors.ink.DEFAULT}
            />
            <Text style={styles.uploadingLabel}>Uploading…</Text>
          </View>
        ) : null}

        {/* Upload action buttons — only shown when not yet uploaded and not uploading */}
        {!isUploaded && !uploadState.uploading ? (
          <View style={styles.actionRow}>
            <Pressable
              onPress={() => handleCamera(docType)}
              style={({ pressed }) => [
                styles.actionBtn,
                pressed && styles.actionBtnPressed,
              ]}
              accessibilityRole="button"
              accessibilityLabel={`Take photo for ${title}`}
            >
              <Text style={styles.actionBtnLabel}>Camera</Text>
            </Pressable>
            <Pressable
              onPress={() => handleGallery(docType)}
              style={({ pressed }) => [
                styles.actionBtn,
                pressed && styles.actionBtnPressed,
              ]}
              accessibilityRole="button"
              accessibilityLabel={`Choose from gallery for ${title}`}
            >
              <Text style={styles.actionBtnLabel}>Gallery</Text>
            </Pressable>
            <Pressable
              onPress={() => handlePdf(docType)}
              style={({ pressed }) => [
                styles.actionBtn,
                pressed && styles.actionBtnPressed,
              ]}
              accessibilityRole="button"
              accessibilityLabel={`Upload PDF for ${title}`}
            >
              <Text style={styles.actionBtnLabel}>PDF</Text>
            </Pressable>
          </View>
        ) : null}

        {/* Error */}
        {uploadState.error ? (
          <Text style={styles.errorText}>{uploadState.error}</Text>
        ) : null}
      </View>
    );
  }

  return (
    <OnboardingScaffold
      step={4}
      total={6}
      title="Documents"
      subtitle="We need these to verify your kitchen before you go live."
      primaryLabel="Continue"
      onPrimary={onNext}
      primaryDisabled={!bothUploaded}
    >
      {renderUploadSlot(
        'ID proof',
        'Aadhaar, PAN, or Passport',
        'id_proof',
        documents.idProofUri,
        documents.idProofType,
        idUpload,
      )}

      {renderUploadSlot(
        'FSSAI license',
        'Food safety registration certificate',
        'fssai_license',
        documents.fssaiUri,
        documents.fssaiType,
        fssaiUpload,
      )}

      <View style={styles.bottomSpacer} />
    </OnboardingScaffold>
  );
}

const styles = StyleSheet.create({
  // Upload slot — bone background, mist border.
  slot: {
    backgroundColor: theme.colors.bone,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    padding: theme.spacing[4],
    gap: theme.spacing[3],
  },

  slotHeader: {
    gap: theme.spacing[0.5],
  },

  slotTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.DEFAULT,
  },

  slotSubtitle: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },

  uploadedBlock: {
    gap: theme.spacing[2],
  },

  previewImage: {
    width: '100%',
    height: 140,
    borderRadius: theme.radius.sm,
    backgroundColor: theme.colors.mist.DEFAULT,
  },

  pdfPreview: {
    width: '100%',
    height: 64,
    borderRadius: theme.radius.sm,
    backgroundColor: theme.colors.mist.DEFAULT,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
  },

  pdfTypeLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.label.size,
    color: theme.colors.ink.soft,
    letterSpacing: 1,
  },

  // Filename + remove × row below the preview.
  filenameRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
  },

  filenameDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
    backgroundColor: theme.colors.herb.DEFAULT,
  },

  filenameText: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
  },

  removeBtn: {
    padding: theme.spacing[1],
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.mist.DEFAULT,
    width: 24,
    height: 24,
    alignItems: 'center',
    justifyContent: 'center',
  },

  removeBtnPressed: {
    backgroundColor: theme.colors.mist.strong,
  },

  removeIcon: {
    fontFamily: 'Inter',
    fontSize: 16,
    color: theme.colors.ink.soft,
    lineHeight: 18,
    textAlign: 'center',
  },

  uploadingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    paddingVertical: theme.spacing[3],
  },

  uploadingLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },

  // Three equal action buttons — Camera / Gallery / PDF.
  actionRow: {
    flexDirection: 'row',
    gap: theme.spacing[2],
  },

  actionBtn: {
    flex: 1,
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.sm,
    paddingVertical: theme.spacing[3],
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: theme.touchTarget.vendor,
    backgroundColor: theme.colors.paper,
  },

  actionBtnPressed: {
    backgroundColor: theme.colors.bone,
    borderColor: theme.colors.ink.DEFAULT,
  },

  actionBtnLabel: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },

  errorText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.destructive.DEFAULT,
  },

  bottomSpacer: {
    height: theme.spacing[2],
  },
});
