// apps/mobile-vendor/app/(onboarding)/documents.tsx
// Step 4/6 — Upload ID proof + FSSAI license.
// StyleSheet only — no NativeWind className.
// Upload tiles: dashed border when empty, image/PDF preview + success row when filled.

import {
  ActivityIndicator,
  Alert,
  Image,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { useState } from 'react';
import * as ImagePicker from 'expo-image-picker';
import * as DocumentPicker from 'expo-document-picker';
import { router } from 'expo-router';
import { useTranslation } from 'react-i18next';
import {
  Camera,
  Image as ImageIcon,
  FileText,
  CheckCircle,
  X,
  ShieldCheck,
  CreditCard,
} from 'lucide-react-native';
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

function uriToFilename(uri: string, fileType: 'image' | 'pdf' | null): string {
  if (!fileType) return 'document';
  const segments = uri.split('/');
  const last = segments[segments.length - 1] ?? '';
  if (last.length > 0 && last.length < 50) return last;
  return fileType === 'pdf' ? 'document.pdf' : 'document.jpg';
}

export default function DocumentsScreen() {
  const { t } = useTranslation();
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
      // FSSAI uploads carry the chef-entered expiry date so the
      // expiry-reminder cron (services/fssai_reminder.go) can fire at
      // 30/15/7 days. Only sent if the chef has typed a parseable
      // YYYY-MM-DD; the backend handler treats missing as no-expiry.
      if (
        docType === 'fssai_license' &&
        /^\d{4}-\d{2}-\d{2}$/.test(documents.fssaiExpiryDate)
      ) {
        formData.append('expiryDate', documents.fssaiExpiryDate);
      }

      await api.post('/chef/documents', formData, multipartConfig());

      if (docType === 'id_proof') {
        updateDocuments({ idProofUri: uri, idProofType: fileType });
      } else {
        updateDocuments({ fssaiUri: uri, fssaiType: fileType });
      }
      setUploadState(docType, { uploading: false, error: null });
      showToast({
        message: docType === 'id_proof' ? t('onboarding.idProofUploaded') : t('onboarding.fssaiUploaded'),
        tone: 'success',
      });
    } catch (error: unknown) {
      const serverError =
        (error as { response?: { data?: { error?: string } } } | null)?.response?.data?.error;
      const message =
        serverError ?? (error instanceof Error ? error.message : t('onboarding.uploadFailed'));
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
        showToast({ message: t('onboarding.cameraDenied'), tone: 'error' });
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
      const message = err instanceof Error ? err.message : t('onboarding.cameraUnavailable');
      showToast({ message, tone: 'error' });
    }
  }

  async function handleGallery(docType: DocumentType): Promise<void> {
    try {
      const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
      if (!permission.granted) {
        showToast({ message: t('onboarding.galleryDenied'), tone: 'error' });
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
      const message = err instanceof Error ? err.message : t('onboarding.galleryFailed');
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
      const message = err instanceof Error ? err.message : t('onboarding.pickerFailed');
      showToast({ message, tone: 'error' });
    }
  }

  function onNext(): void {
    if (!documents.idProofUri || !documents.fssaiUri) {
      Alert.alert(
        t('onboarding.documentsRequired'),
        t('onboarding.documentsRequiredBody'),
      );
      return;
    }
    setStep(5);
    router.push('/(onboarding)/policies');
  }

  const bothUploaded = Boolean(documents.idProofUri && documents.fssaiUri);

  function renderUploadTile(
    title: string,
    subtitle: string,
    icon: React.ReactElement,
    docType: DocumentType,
    uri: string | null,
    fileType: 'image' | 'pdf' | null,
    uploadState: UploadState,
  ): React.ReactElement {
    const isUploaded = uri !== null;

    // ── UPLOADING STATE ──
    if (uploadState.uploading) {
      return (
        <View style={styles.tile}>
          <View style={styles.tileHeader}>
            <View style={styles.tileIconWrap}>{icon}</View>
            <View style={styles.tileTitleGroup}>
              <Text style={styles.tileTitle}>{title}</Text>
              <Text style={styles.tileSubtitle}>{subtitle}</Text>
            </View>
          </View>
          <View style={styles.uploadingRow}>
            <ActivityIndicator size="small" color={theme.colors.ink.DEFAULT} />
            <Text style={styles.uploadingLabel}>{t('onboarding.uploading')}</Text>
          </View>
        </View>
      );
    }

    // ── UPLOADED STATE ──
    if (isUploaded) {
      return (
        <View style={[styles.tile, styles.tileUploaded]}>
          <View style={styles.tileHeader}>
            <View style={[styles.tileIconWrap, styles.tileIconSuccess]}>
              <CheckCircle size={18} color={theme.colors.herb.DEFAULT} strokeWidth={2} />
            </View>
            <View style={styles.tileTitleGroup}>
              <Text style={styles.tileTitle}>{title}</Text>
              <Text style={[styles.tileSubtitle, styles.tileSubtitleSuccess]}>
                {uriToFilename(uri, fileType)}
              </Text>
            </View>
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
              <X size={14} color={theme.colors.ink.soft} strokeWidth={2.5} />
            </Pressable>
          </View>

          {/* Preview */}
          {fileType === 'image' ? (
            <Image
              source={{ uri }}
              style={styles.previewImage}
              resizeMode="cover"
              accessibilityLabel={`${title} preview`}
            />
          ) : (
            <View style={styles.pdfPreview}>
              <FileText size={24} color={theme.colors.ink.soft} strokeWidth={1.5} />
              <Text style={styles.pdfTypeLabel}>{t('onboarding.pdfDocument')}</Text>
            </View>
          )}

          {/* Tap-to-replace */}
          <View style={styles.replaceRow}>
            <Pressable
              onPress={() => handleGallery(docType)}
              style={({ pressed }) => [styles.replaceBtn, pressed && { opacity: 0.7 }]}
              accessibilityRole="button"
              accessibilityLabel={`Replace ${title} with gallery photo`}
            >
              <Text style={styles.replaceBtnLabel}>{t('onboarding.replace')}</Text>
            </Pressable>
          </View>
        </View>
      );
    }

    // ── EMPTY STATE — dashed upload tile ──
    return (
      <View style={styles.tile}>
        <View style={styles.tileHeader}>
          <View style={styles.tileIconWrap}>{icon}</View>
          <View style={styles.tileTitleGroup}>
            <Text style={styles.tileTitle}>{title}</Text>
            <Text style={styles.tileSubtitle}>{subtitle}</Text>
          </View>
        </View>

        {/* Dashed placeholder zone */}
        <View style={styles.dashedZone}>
          <Text style={styles.dashedZoneHint}>{t('onboarding.tapToUpload')}</Text>
        </View>

        {/* Three upload affordances: Camera (primary), Gallery, PDF */}
        <View style={styles.actionRow}>
          <Pressable
            onPress={() => handleCamera(docType)}
            style={({ pressed }) => [
              styles.actionBtnPrimary,
              pressed && { opacity: 0.85 },
            ]}
            accessibilityRole="button"
            accessibilityLabel={`Take photo for ${title}`}
          >
            <Camera size={15} color={theme.colors.paper} strokeWidth={2} />
            <Text style={styles.actionBtnPrimaryLabel}>{t('onboarding.camera')}</Text>
          </Pressable>
          <Pressable
            onPress={() => handleGallery(docType)}
            style={({ pressed }) => [
              styles.actionBtnSecondary,
              pressed && styles.actionBtnSecondaryPressed,
            ]}
            accessibilityRole="button"
            accessibilityLabel={`Choose from gallery for ${title}`}
          >
            <ImageIcon size={15} color={theme.colors.ink.soft} strokeWidth={2} />
            <Text style={styles.actionBtnSecondaryLabel}>{t('onboarding.gallery')}</Text>
          </Pressable>
          <Pressable
            onPress={() => handlePdf(docType)}
            style={({ pressed }) => [
              styles.actionBtnSecondary,
              pressed && styles.actionBtnSecondaryPressed,
            ]}
            accessibilityRole="button"
            accessibilityLabel={`Upload PDF for ${title}`}
          >
            <FileText size={15} color={theme.colors.ink.soft} strokeWidth={2} />
            <Text style={styles.actionBtnSecondaryLabel}>{t('onboarding.pdf')}</Text>
          </Pressable>
        </View>

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
      title={t('onboarding.documentsTitle')}
      subtitle={t('onboarding.documentsSubtitle')}
      primaryLabel={t('onboarding.continue')}
      onPrimary={onNext}
      primaryDisabled={!bothUploaded}
    >
      {/* Context note */}
      <View style={styles.noteRow}>
        <ShieldCheck size={14} color={theme.colors.herb.DEFAULT} strokeWidth={2} />
        <Text style={styles.noteText}>
          {t('onboarding.documentsNote')}
        </Text>
      </View>

      {/* ── REQUIRED ──────────────────────────────────────────── */}
      <View style={styles.sectionLabel}>
        <Text style={styles.sectionLabelText}>{t('onboarding.required')}</Text>
      </View>

      {renderUploadTile(
        t('onboarding.idProof'),
        t('onboarding.idProofSubtitle'),
        <CreditCard size={18} color={theme.colors.ink.soft} strokeWidth={1.5} />,
        'id_proof',
        documents.idProofUri,
        documents.idProofType,
        idUpload,
      )}

      <View style={styles.tileSpacer} />

      {renderUploadTile(
        t('onboarding.fssaiLicense'),
        t('onboarding.fssaiLicenseSubtitle'),
        <ShieldCheck size={18} color={theme.colors.ink.soft} strokeWidth={1.5} />,
        'fssai_license',
        documents.fssaiUri,
        documents.fssaiType,
        fssaiUpload,
      )}

      {/* FSSAI license number + expiry — collected as structured fields
          alongside the photo so admin tooling can validate, FoSCoS API
          can verify, and Wave 3 invoicing can print them. Always
          visible (not gated on photo upload) so the chef can type the
          number while their hands are on the keyboard. */}
      <View style={styles.fssaiFields}>
        <View style={styles.fssaiFieldGroup}>
          <Text style={styles.fssaiFieldLabel}>{t('onboarding.licenseNumber')}</Text>
          <TextInput
            value={documents.fssaiLicenseNumber}
            onChangeText={(v) =>
              updateDocuments({ fssaiLicenseNumber: v.replace(/\D/g, '').slice(0, 14) })
            }
            placeholder={t('onboarding.fssaiNumberPlaceholder')}
            placeholderTextColor={theme.colors.ink.muted}
            keyboardType="number-pad"
            maxLength={14}
            autoCorrect={false}
            style={styles.fssaiInput}
          />
          {documents.fssaiLicenseNumber.length > 0 &&
            documents.fssaiLicenseNumber.length !== 14 && (
              <Text style={styles.fssaiHelpError}>{t('onboarding.fssaiNumberError')}</Text>
            )}
        </View>
        <View style={styles.fssaiFieldGroup}>
          <Text style={styles.fssaiFieldLabel}>{t('onboarding.expiryDate')}</Text>
          <TextInput
            value={documents.fssaiExpiryDate}
            onChangeText={(v) => updateDocuments({ fssaiExpiryDate: v })}
            placeholder={t('onboarding.expiryDatePlaceholder')}
            placeholderTextColor={theme.colors.ink.muted}
            keyboardType="numbers-and-punctuation"
            maxLength={10}
            autoCorrect={false}
            style={styles.fssaiInput}
          />
          {documents.fssaiExpiryDate.length > 0 &&
            !/^\d{4}-\d{2}-\d{2}$/.test(documents.fssaiExpiryDate) && (
              <Text style={styles.fssaiHelpError}>{t('onboarding.expiryDateError')}</Text>
            )}
          <Text style={styles.fssaiHelpHint}>
            {t('onboarding.expiryReminder')}
          </Text>
        </View>

        {/* GSTIN — optional regulatory ID. Chefs below the GST
            threshold (₹20L turnover) skip this; everyone else uses
            it to claim input tax credit and to print on the customer
            invoice per Wave 3. */}
        <View style={styles.fssaiFieldGroup}>
          <Text style={styles.fssaiFieldLabel}>{t('onboarding.gstinOptional')}</Text>
          <TextInput
            value={documents.gstin}
            onChangeText={(v) =>
              updateDocuments({ gstin: v.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 15) })
            }
            placeholder={t('onboarding.gstinPlaceholder')}
            placeholderTextColor={theme.colors.ink.muted}
            autoCapitalize="characters"
            autoCorrect={false}
            maxLength={15}
            style={styles.fssaiInput}
          />
          {documents.gstin.length > 0 && documents.gstin.length !== 15 && (
            <Text style={styles.fssaiHelpError}>{t('onboarding.gstinError')}</Text>
          )}
          <Text style={styles.fssaiHelpHint}>
            {t('onboarding.gstinHint')}
          </Text>
        </View>
      </View>

      <View style={styles.bottomSpacer} />
    </OnboardingScaffold>
  );
}

const styles = StyleSheet.create({
  // Context note strip
  noteRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: theme.spacing[2],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.sm,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    marginBottom: theme.spacing[4],
    borderLeftWidth: 2,
    borderLeftColor: theme.colors.herb.DEFAULT,
  },
  noteText: {
    flex: 1,
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.soft,
    lineHeight: theme.typography.size.caption.size * 1.5,
  },

  // FSSAI structured fields — under the photo tile
  fssaiFields: {
    marginTop: theme.spacing[3],
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[1],
  },
  fssaiFieldGroup: { gap: 6 },
  fssaiFieldLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.caption.size,
    letterSpacing: 0.4,
    color: theme.colors.ink.muted,
  },
  fssaiInput: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
    paddingHorizontal: 12,
    paddingVertical: 10,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
    borderRadius: 6,
    backgroundColor: theme.colors.paper,
  },
  fssaiHelpError: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.destructive.DEFAULT,
  },
  fssaiHelpHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },

  // Section caps label
  sectionLabel: {
    marginBottom: theme.spacing[2],
  },
  sectionLabelText: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 10,
    letterSpacing: 1.2,
    color: theme.colors.ink.muted,
  },

  // ── TILE ────────────────────────────────────────────────────
  tile: {
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: theme.radius.DEFAULT,
    backgroundColor: theme.colors.paper,
    overflow: 'hidden',
    gap: 0,
  },

  // Uploaded tile gets a hairline persimmon tint on the top edge
  tileUploaded: {
    borderTopWidth: 2,
    borderTopColor: theme.colors.herb.DEFAULT,
  },

  tileHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    paddingHorizontal: theme.spacing[4],
    paddingTop: theme.spacing[4],
    paddingBottom: theme.spacing[3],
  },

  tileIconWrap: {
    width: 36,
    height: 36,
    borderRadius: theme.radius.sm,
    backgroundColor: theme.colors.bone,
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
  },
  tileIconSuccess: {
    backgroundColor: 'rgba(194, 65, 12, 0.08)',
  },

  tileTitleGroup: {
    flex: 1,
    gap: 2,
  },
  tileTitle: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  tileSubtitle: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },
  tileSubtitleSuccess: {
    color: theme.colors.ink.soft,
    fontFamily: 'Inter-Medium',
  },

  // Remove × button in uploaded state
  removeBtn: {
    width: 28,
    height: 28,
    borderRadius: theme.radius.full,
    backgroundColor: theme.colors.bone,
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
  },
  removeBtnPressed: {
    backgroundColor: theme.colors.mist.DEFAULT,
  },

  // Dashed placeholder — shown when empty, not uploading
  dashedZone: {
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[3],
    borderWidth: 1.5,
    borderColor: theme.colors.mist.strong,
    borderStyle: 'dashed',
    borderRadius: theme.radius.sm,
    paddingVertical: theme.spacing[5],
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: theme.colors.bone,
  },
  dashedZoneHint: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.ink.muted,
  },

  // Upload action row
  actionRow: {
    flexDirection: 'row',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[4],
  },

  // Camera — ink-filled primary (fast path)
  actionBtnPrimary: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: theme.spacing[1],
    backgroundColor: theme.colors.ink.DEFAULT,
    borderRadius: theme.radius.sm,
    paddingVertical: theme.spacing[3],
    minHeight: theme.touchTarget.vendor,
  },
  actionBtnPrimaryLabel: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.paper,
  },

  // Gallery + PDF — hairline outlined secondary
  actionBtnSecondary: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: theme.spacing[1],
    borderWidth: 1,
    borderColor: theme.colors.mist.strong,
    borderRadius: theme.radius.sm,
    paddingVertical: theme.spacing[3],
    minHeight: theme.touchTarget.vendor,
    backgroundColor: theme.colors.paper,
  },
  actionBtnSecondaryPressed: {
    backgroundColor: theme.colors.bone,
    borderColor: theme.colors.ink.DEFAULT,
  },
  actionBtnSecondaryLabel: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },

  // Uploading spinner row
  uploadingRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[4],
  },
  uploadingLabel: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.muted,
  },

  // Preview image (uploaded)
  previewImage: {
    width: '100%',
    height: 160,
    backgroundColor: theme.colors.mist.DEFAULT,
  },

  // PDF preview block
  pdfPreview: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[2],
    marginHorizontal: theme.spacing[4],
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    borderRadius: theme.radius.sm,
    backgroundColor: theme.colors.bone,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: theme.colors.mist.strong,
  },
  pdfTypeLabel: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
  },

  // Tap-to-replace row (uploaded state)
  replaceRow: {
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: theme.colors.mist.DEFAULT,
    marginTop: theme.spacing[2],
  },
  replaceBtn: {
    alignSelf: 'flex-start',
  },
  replaceBtnLabel: {
    fontFamily: 'Inter-Medium',
    fontSize: theme.typography.size.bodySm.size,
    color: theme.colors.ink.soft,
    textDecorationLine: 'underline',
  },

  errorText: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.caption.size,
    color: theme.colors.destructive.DEFAULT,
    paddingHorizontal: theme.spacing[4],
    paddingBottom: theme.spacing[3],
  },

  tileSpacer: {
    height: theme.spacing[3],
  },

  bottomSpacer: {
    height: theme.spacing[4],
  },
});
