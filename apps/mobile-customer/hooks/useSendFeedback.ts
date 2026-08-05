import { Platform } from 'react-native';
import Constants from 'expo-constants';
import { useMutation } from '@tanstack/react-query';
import {
  submitFeedback,
  type FeedbackKind,
  type FeedbackReceipt,
} from '@homechef/mobile-shared/feedback';

import { api } from '../lib/api';

// Send feedback / share an idea. The API files it on the product backlog as a
// labelled issue, so what a customer writes reaches triage unedited.

export interface SendFeedbackInput {
  kind: FeedbackKind;
  title: string;
  message: string;
}

export function useSendFeedback() {
  return useMutation<FeedbackReceipt, Error, SendFeedbackInput>({
    mutationFn: (input) =>
      submitFeedback(api, {
        ...input,
        app: 'customer',
        platform: Platform.OS,
        appVersion: Constants.expoConfig?.version ?? undefined,
      }),
  });
}
