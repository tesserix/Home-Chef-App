import { useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  Alert,
  ActivityIndicator,
  Image,
} from 'react-native';
import * as ImagePicker from 'expo-image-picker';
import * as DocumentPicker from 'expo-document-picker';
import { router } from 'expo-router';
import { SafeAreaView } from 'react-native-safe-area-context';
import { api } from '../../lib/api';
import { useDriverOnboardingStore } from '../../store/onboarding-store';

type DocumentType = 'driving_license' | 'id_proof' | 'vehicle_rc';

interface DocumentSlot {
  type: DocumentType;
  label: string;
  required: boolean;
  uri: string | null;
}

async function uploadDocument(uri: string, type: DocumentType, mimeType?: string): Promise<void> {
  const formData = new FormData();
  formData.append('type', type);
  formData.append('file', {
    uri,
    name: `document.${mimeType?.includes('pdf') ? 'pdf' : 'jpg'}`,
    type: mimeType ?? 'image/jpeg',
  } as unknown as Blob);
  // CRITICAL: upload to /driver/onboarding/documents (not /delivery/documents)
  await api.post('/driver/onboarding/documents', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
}

export default function DocumentsScreen() {
  const { documents, updateDocuments, setStep } = useDriverOnboardingStore();
  const [uploading, setUploading] = useState<DocumentType | null>(null);
  const [isNavigating, setIsNavigating] = useState(false);

  const [slots, setSlots] = useState<DocumentSlot[]>([
    { type: 'driving_license', label: 'Driving License', required: true, uri: documents.drivingLicenseUri },
    { type: 'id_proof', label: 'ID Proof', required: true, uri: documents.idProofUri },
    { type: 'vehicle_rc', label: 'Vehicle RC (optional)', required: false, uri: documents.vehicleRcUri },
  ]);

  const canProceed =
    slots.find((s: DocumentSlot) => s.type === 'driving_license')?.uri !== null &&
    slots.find((s: DocumentSlot) => s.type === 'id_proof')?.uri !== null;

  const updateSlotUri = (type: DocumentType, uri: string) => {
    setSlots((prev: DocumentSlot[]) =>
      prev.map((slot: DocumentSlot) => (slot.type === type ? { ...slot, uri } : slot))
    );
    if (type === 'driving_license') updateDocuments({ drivingLicenseUri: uri });
    else if (type === 'id_proof') updateDocuments({ idProofUri: uri });
    else if (type === 'vehicle_rc') updateDocuments({ vehicleRcUri: uri });
  };

  const handleCamera = async (type: DocumentType) => {
    const { status } = await ImagePicker.requestCameraPermissionsAsync();
    if (status !== 'granted') {
      Alert.alert('Permission Required', 'Camera permission is required to capture documents.');
      return;
    }
    const result = await ImagePicker.launchCameraAsync({
      mediaTypes: ['images'],
      quality: 0.8,
      allowsEditing: true,
    });
    if (!result.canceled && result.assets[0]) {
      setUploading(type);
      try {
        await uploadDocument(result.assets[0].uri, type, 'image/jpeg');
        updateSlotUri(type, result.assets[0].uri);
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : 'Upload failed. Please try again.';
        Alert.alert('Upload Error', message);
      } finally {
        setUploading(null);
      }
    }
  };

  const handleGallery = async (type: DocumentType) => {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      quality: 0.8,
      allowsEditing: true,
    });
    if (!result.canceled && result.assets[0]) {
      setUploading(type);
      try {
        await uploadDocument(result.assets[0].uri, type, 'image/jpeg');
        updateSlotUri(type, result.assets[0].uri);
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : 'Upload failed. Please try again.';
        Alert.alert('Upload Error', message);
      } finally {
        setUploading(null);
      }
    }
  };

  const handlePdf = async (type: DocumentType) => {
    const result = await DocumentPicker.getDocumentAsync({
      type: ['application/pdf'],
      copyToCacheDirectory: true,
    });
    if (!result.canceled && result.assets[0]) {
      setUploading(type);
      try {
        await uploadDocument(result.assets[0].uri, type, 'application/pdf');
        updateSlotUri(type, result.assets[0].uri);
      } catch (err: unknown) {
        const message = err instanceof Error ? err.message : 'Upload failed. Please try again.';
        Alert.alert('Upload Error', message);
      } finally {
        setUploading(null);
      }
    }
  };

  const handleNext = () => {
    if (!canProceed) {
      Alert.alert('Required Documents', 'Please upload Driving License and ID Proof to continue.');
      return;
    }
    setStep(4);
    setIsNavigating(true);
    router.push('/(onboarding)/payout');
  };

  return (
    <SafeAreaView className="flex-1 bg-bone" edges={['bottom']}>
      <ScrollView className="flex-1 px-6" keyboardShouldPersistTaps="handled">
        {/* Progress bar */}
        <View className="mt-4 mb-6 h-1 bg-mist rounded-full">
          <View className="h-1 bg-herb rounded-full" style={{ width: '50%' }} />
        </View>

        <Text className="font-display text-2xl font-semibold text-ink mb-2">Upload Documents</Text>
        <Text className="text-ink-muted mb-6">
          Please upload clear photos or PDFs of your documents
        </Text>

        {slots.map((slot: DocumentSlot) => (
          <View key={slot.type} className="mb-6 border border-mist rounded-xl p-4">
            <Text className="text-base font-semibold text-ink mb-3">
              {slot.label}
              {slot.required && <Text className="text-paprika"> *</Text>}
            </Text>

            {/* Preview or placeholder */}
            {slot.uri ? (
              <View className="mb-3 rounded-lg overflow-hidden border border-mist">
                {slot.uri.endsWith('.pdf') ? (
                  <View className="h-24 bg-mist items-center justify-center">
                    <Text className="text-ink-muted text-sm">PDF uploaded</Text>
                  </View>
                ) : (
                  <Image source={{ uri: slot.uri }} className="w-full h-32" resizeMode="cover" />
                )}
              </View>
            ) : (
              <View className="mb-3 h-24 bg-paper rounded-lg border border-dashed border-mist-strong items-center justify-center">
                <Text className="text-ink-muted text-sm">No document uploaded</Text>
              </View>
            )}

            {/* Upload buttons */}
            {uploading === slot.type ? (
              <View className="items-center py-2">
                <ActivityIndicator color="#C2410C" />
                <Text className="text-ink-muted text-sm mt-1">Uploading...</Text>
              </View>
            ) : (
              <View className="flex-row gap-2">
                <TouchableOpacity
                  onPress={() => handleCamera(slot.type)}
                  disabled={uploading !== null}
                  className="flex-1 py-2 border border-herb rounded-lg items-center"
                >
                  <Text className="text-herb text-sm font-medium">Camera</Text>
                </TouchableOpacity>
                <TouchableOpacity
                  onPress={() => handleGallery(slot.type)}
                  disabled={uploading !== null}
                  className="flex-1 py-2 border border-herb rounded-lg items-center"
                >
                  <Text className="text-herb text-sm font-medium">Gallery</Text>
                </TouchableOpacity>
                <TouchableOpacity
                  onPress={() => handlePdf(slot.type)}
                  disabled={uploading !== null}
                  className="flex-1 py-2 border border-herb rounded-lg items-center"
                >
                  <Text className="text-herb text-sm font-medium">Upload PDF</Text>
                </TouchableOpacity>
              </View>
            )}
          </View>
        ))}

        <View className="mb-8" />
      </ScrollView>

      {/* Next Button */}
      <View className="px-6 py-4 border-t border-mist">
        <TouchableOpacity
          onPress={handleNext}
          disabled={!canProceed || isNavigating}
          className={`w-full py-4 rounded-xl items-center ${canProceed && !isNavigating ? 'bg-herb' : 'bg-mist-strong'}`}
        >
          <Text className="text-paper font-semibold text-base">Next</Text>
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
}
