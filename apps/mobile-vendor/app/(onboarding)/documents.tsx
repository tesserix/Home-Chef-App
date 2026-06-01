import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  Image,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { useState } from 'react';
import * as ImagePicker from 'expo-image-picker';
import * as DocumentPicker from 'expo-document-picker';
import { router } from 'expo-router';
import { multipartConfig } from '@homechef/mobile-shared/api';
import { api } from '../../lib/api';
import { useVendorOnboardingStore } from '../../store/onboarding-store';

type DocumentType = 'id_proof' | 'fssai_license';

interface UploadState {
  uploading: boolean;
  error: string | null;
}

export default function DocumentsScreen() {
  const { documents, updateDocuments, setStep } = useVendorOnboardingStore();

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
      formData.append('file', {
        uri,
        name: fileType === 'pdf' ? 'document.pdf' : 'document.jpg',
        type: mimeType,
      } as unknown as Blob);

      await api.post('/chef/documents', formData, multipartConfig());

      if (docType === 'id_proof') {
        updateDocuments({ idProofUri: uri, idProofType: fileType });
      } else {
        updateDocuments({ fssaiUri: uri, fssaiType: fileType });
      }
      setUploadState(docType, { uploading: false, error: null });
    } catch (error: unknown) {
      const serverError = (error as { response?: { data?: { error?: string } } } | null)?.response?.data?.error;
      const fallback = error instanceof Error ? error.message : 'Upload failed. Please try again.';
      setUploadState(docType, { uploading: false, error: serverError ?? fallback });
    }
  }

  async function handleCamera(docType: DocumentType): Promise<void> {
    const permission = await ImagePicker.requestCameraPermissionsAsync();
    if (!permission.granted) {
      Alert.alert('Permission Required', 'Camera permission is needed to take photos.');
      return;
    }
    const result = await ImagePicker.launchCameraAsync({
      mediaTypes: ['images'],
      quality: 0.85,
      allowsEditing: true,
    });
    if (!result.canceled && result.assets[0]) {
      const asset = result.assets[0];
      await uploadFile(asset.uri, 'image', 'image/jpeg', docType);
    }
  }

  async function handleGallery(docType: DocumentType): Promise<void> {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      Alert.alert('Permission Required', 'Gallery permission is needed to select photos.');
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      quality: 0.85,
      allowsEditing: true,
    });
    if (!result.canceled && result.assets[0]) {
      const asset = result.assets[0];
      await uploadFile(asset.uri, 'image', 'image/jpeg', docType);
    }
  }

  async function handlePdf(docType: DocumentType): Promise<void> {
    const result = await DocumentPicker.getDocumentAsync({
      type: ['application/pdf'],
      copyToCacheDirectory: true,
    });
    if (!result.canceled && result.assets[0]) {
      const asset = result.assets[0];
      await uploadFile(asset.uri, 'pdf', 'application/pdf', docType);
    }
  }

  function onNext(): void {
    if (!documents.idProofUri || !documents.fssaiUri) {
      Alert.alert('Documents Required', 'Please upload both ID proof and FSSAI license to continue.');
      return;
    }
    setStep(5);
    router.push('/(onboarding)/policies');
  }

  const bothUploaded = Boolean(documents.idProofUri && documents.fssaiUri);

  function renderUploadSlot(
    label: string,
    docType: DocumentType,
    uri: string | null,
    fileType: 'image' | 'pdf' | null,
    uploadState: UploadState,
  ): React.ReactElement {
    return (
      <View className="mb-6 border border-mist rounded-xl p-4">
        <Text className="text-sm font-semibold text-ink mb-3">{label}</Text>

        {uri ? (
          <View className="items-center mb-3">
            {fileType === 'image' ? (
              <Image
                source={{ uri }}
                className="w-full h-40 rounded-lg"
                resizeMode="cover"
              />
            ) : (
              <View className="w-full h-24 bg-mist rounded-lg items-center justify-center">
                <Text className="text-3xl">📄</Text>
                <Text className="text-sm text-ink-muted mt-1">PDF uploaded</Text>
              </View>
            )}
            <Text className="text-xs text-herb font-medium mt-2">Uploaded successfully</Text>
          </View>
        ) : null}

        {uploadState.uploading ? (
          <View className="items-center py-4">
            <ActivityIndicator color="#C2410C" />
            <Text className="text-sm text-ink-muted mt-2">Uploading...</Text>
          </View>
        ) : (
          <View className="flex-row gap-2">
            <TouchableOpacity
              className="flex-1 border border-mist-strong rounded-lg py-2.5 items-center"
              onPress={() => handleCamera(docType)}
            >
              <Text className="text-sm text-ink-soft">Camera</Text>
            </TouchableOpacity>
            <TouchableOpacity
              className="flex-1 border border-mist-strong rounded-lg py-2.5 items-center"
              onPress={() => handleGallery(docType)}
            >
              <Text className="text-sm text-ink-soft">Gallery</Text>
            </TouchableOpacity>
            <TouchableOpacity
              className="flex-1 border border-mist-strong rounded-lg py-2.5 items-center"
              onPress={() => handlePdf(docType)}
            >
              <Text className="text-sm text-ink-soft">PDF</Text>
            </TouchableOpacity>
          </View>
        )}

        {uploadState.error ? (
          <Text className="text-paprika text-xs mt-2">{uploadState.error}</Text>
        ) : null}
      </View>
    );
  }

  return (
    <ScrollView className="flex-1 bg-bone">
      <View className="px-6 pt-4 pb-8">
        <View className="h-1.5 rounded-full bg-mist mb-6">
          <View className="h-1.5 rounded-full bg-herb" style={{ width: `${(4 / 6) * 100}%` }} />
        </View>

        <Text className="font-display text-2xl font-semibold text-ink mb-1">Documents</Text>
        <Text className="text-sm text-ink-muted mb-6">Upload your identity and FSSAI documents</Text>

        {renderUploadSlot('ID Proof', 'id_proof', documents.idProofUri, documents.idProofType, idUpload)}
        {renderUploadSlot('FSSAI License', 'fssai_license', documents.fssaiUri, documents.fssaiType, fssaiUpload)}

        <TouchableOpacity
          className={`rounded-xl py-4 items-center ${bothUploaded ? 'bg-herb' : 'bg-mist-strong'}`}
          onPress={onNext}
          disabled={!bothUploaded}
        >
          <Text className={`font-semibold text-base ${bothUploaded ? 'text-paper' : 'text-ink-muted'}`}>
            Next
          </Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}
