// Type stubs for Expo SDK managed-workflow modules not yet in node_modules.
// These packages are listed in package.json and will be installed at build time.
// Stubs provide TypeScript coverage until `pnpm install` runs.

declare module 'expo-image-picker' {
  export type MediaTypeOptions = 'Images' | 'Videos' | 'All';
  export type ImagePickerAsset = {
    uri: string;
    width: number;
    height: number;
    type?: string;
    fileName?: string;
    fileSize?: number;
    exif?: Record<string, unknown>;
    base64?: string;
  };
  export type ImagePickerResult =
    | { canceled: true; assets: null }
    | { canceled: false; assets: ImagePickerAsset[] };
  export type ImagePickerOptions = {
    mediaTypes?: string[];
    allowsEditing?: boolean;
    aspect?: [number, number];
    quality?: number;
    base64?: boolean;
    exif?: boolean;
    allowsMultipleSelection?: boolean;
  };
  export type PermissionResponse = {
    granted: boolean;
    canAskAgain: boolean;
    status: string;
  };
  export function launchImageLibraryAsync(options?: ImagePickerOptions): Promise<ImagePickerResult>;
  export function launchCameraAsync(options?: ImagePickerOptions): Promise<ImagePickerResult>;
  export function requestMediaLibraryPermissionsAsync(): Promise<PermissionResponse>;
  export function requestCameraPermissionsAsync(): Promise<PermissionResponse>;
}

declare module 'expo-document-picker' {
  export type DocumentPickerAsset = {
    uri: string;
    name: string;
    mimeType?: string;
    size?: number;
    lastModified?: number;
  };
  export type DocumentPickerResult =
    | { canceled: true; assets: null }
    | { canceled: false; assets: DocumentPickerAsset[] };
  export type DocumentPickerOptions = {
    type?: string | string[];
    copyToCacheDirectory?: boolean;
    multiple?: boolean;
  };
  export function getDocumentAsync(options?: DocumentPickerOptions): Promise<DocumentPickerResult>;
}
