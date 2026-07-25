// Typed wrapper over the /auth/mfa/* endpoints.
//
// Kept free of React so it can be used from a hook, a screen, or the auth
// bootstrap path. The axios instance is passed in because each app builds its
// own with its own base URL and app scope.

import type { AxiosInstance } from 'axios';

export type MFAChannel = 'email' | 'phone';

export interface MFAStatus {
  /** Whether the server has the feature switched on at all. */
  featureEnabled: boolean;
  enabled: boolean;
  emailEnrolled: boolean;
  phoneEnrolled: boolean;
  maskedEmail: string;
  maskedPhone: string;
  backupCodesRemaining: number;
  channels: MFAChannel[];
}

export interface TrustedDevice {
  id: string;
  app: string;
  label: string;
  platform: string;
  createdAt: string;
  lastSeenAt: string;
}

export interface ChallengeResult {
  channel: MFAChannel;
  masked: string;
  /** "email" when a code was sent; "firebase" when the client drives the SMS. */
  delivery: 'email' | 'firebase';
  expiresInSeconds?: number;
}

export interface VerifyResult {
  verified: boolean;
  /** Present it on later requests. Persistent when remembered, else session-scoped. */
  deviceToken: string;
  remembered: boolean;
}

export function createMFAApi(api: AxiosInstance) {
  return {
    async status(): Promise<MFAStatus> {
      const { data } = await api.get<MFAStatus>('/auth/mfa/status');
      return data;
    },

    async requestEmailEnrollment(): Promise<{ maskedEmail: string }> {
      const { data } = await api.post('/auth/mfa/enroll/email/request');
      return data;
    },

    async verifyEmailEnrollment(code: string): Promise<void> {
      await api.post('/auth/mfa/enroll/email/verify', { code });
    },

    /** firebaseIdToken comes from a completed Firebase phone verification. */
    async enrollPhone(firebaseIdToken: string): Promise<{ maskedPhone: string }> {
      const { data } = await api.post('/auth/mfa/enroll/phone', { firebaseIdToken });
      return data;
    },

    /** Returns the backup codes — shown exactly once, never retrievable again. */
    async enable(): Promise<{ backupCodes: string[] }> {
      const { data } = await api.post('/auth/mfa/enable');
      return data;
    },

    async disable(): Promise<void> {
      await api.post('/auth/mfa/disable');
    },

    async regenerateBackupCodes(): Promise<{ backupCodes: string[] }> {
      const { data } = await api.post('/auth/mfa/backup-codes/regenerate');
      return data;
    },

    async challenge(channel: MFAChannel): Promise<ChallengeResult> {
      const { data } = await api.post<ChallengeResult>('/auth/mfa/challenge', { channel });
      return data;
    },

    async verify(input: {
      channel?: MFAChannel;
      code?: string;
      backupCode?: string;
      firebaseIdToken?: string;
      rememberDevice: boolean;
      deviceLabel?: string;
      platform?: string;
    }): Promise<VerifyResult> {
      const { data } = await api.post<VerifyResult>('/auth/mfa/verify', input);
      return data;
    },

    async devices(): Promise<TrustedDevice[]> {
      const { data } = await api.get<{ devices: TrustedDevice[] }>('/auth/mfa/devices');
      return data.devices ?? [];
    },

    async revokeDevice(id: string): Promise<void> {
      await api.delete(`/auth/mfa/devices/${id}`);
    },

    async revokeAllDevices(): Promise<void> {
      await api.post('/auth/mfa/devices/revoke-all');
    },
  };
}

export type MFAApi = ReturnType<typeof createMFAApi>;
