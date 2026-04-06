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

declare module 'lucide-react-native' {
  import type { ComponentType } from 'react';
  interface IconProps {
    size?: number;
    color?: string;
    strokeWidth?: number;
  }
  export const LayoutDashboard: ComponentType<IconProps>;
  export const ClipboardList: ComponentType<IconProps>;
  export const UtensilsCrossed: ComponentType<IconProps>;
  export const MoreHorizontal: ComponentType<IconProps>;
  export const ChevronRight: ComponentType<IconProps>;
  export const Check: ComponentType<IconProps>;
  export const X: ComponentType<IconProps>;
  export const Camera: ComponentType<IconProps>;
  export const Upload: ComponentType<IconProps>;
  export const FileText: ComponentType<IconProps>;
  export const Star: ComponentType<IconProps>;
  export const Settings: ComponentType<IconProps>;
  export const User: ComponentType<IconProps>;
  export const DollarSign: ComponentType<IconProps>;
  export const BarChart2: ComponentType<IconProps>;
  export const LogOut: ComponentType<IconProps>;
}

declare module '@gorhom/bottom-sheet' {
  import type { ComponentType, ReactNode, RefObject } from 'react';
  import type { ViewStyle } from 'react-native';

  interface BottomSheetModalProviderProps {
    children: ReactNode;
  }
  export const BottomSheetModalProvider: ComponentType<BottomSheetModalProviderProps>;

  interface BottomSheetProps {
    index?: number;
    snapPoints?: (string | number)[];
    children?: ReactNode;
    style?: ViewStyle;
    onChange?: (index: number) => void;
    onClose?: () => void;
    enablePanDownToClose?: boolean;
    backgroundStyle?: ViewStyle;
  }
  export class BottomSheet extends Object {
    snapToIndex(index: number): void;
    snapToPosition(position: string | number): void;
    expand(): void;
    collapse(): void;
    close(): void;
  }
  export const BottomSheetView: ComponentType<{ children?: ReactNode; style?: ViewStyle }>;
  export const BottomSheetScrollView: ComponentType<{ children?: ReactNode; style?: ViewStyle }>;

  interface BottomSheetModalProps extends BottomSheetProps {
    ref?: RefObject<unknown>;
  }
  export class BottomSheetModal extends Object {
    present(): void;
    dismiss(): void;
  }
  export function useBottomSheetModal(): { dismiss: () => void; dismissAll: () => void };
}
