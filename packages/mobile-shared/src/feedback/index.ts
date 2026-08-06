// In-app "Send feedback" / "Share an idea", filed onto the product backlog as a
// labelled GitHub issue by the API. Transport and rules live here so the
// customer and vendor apps only own their own layout.

import type { AxiosInstance } from 'axios';

import { apiVersionPrefix } from '../api/version-prefix';

export type FeedbackKind = 'feedback' | 'idea';
export type FeedbackApp = 'customer' | 'vendor' | 'delivery';

export const FEEDBACK_TITLE_MAX = 200;
export const FEEDBACK_MESSAGE_MAX = 5000;
const FEEDBACK_MESSAGE_MIN = 10;

export interface FeedbackInput {
  kind: FeedbackKind;
  title: string;
  message: string;
  app: FeedbackApp;
  platform?: string;
  appVersion?: string;
}

export interface FeedbackReceipt {
  /** Issue number on the backlog — shown back to the user as a reference. */
  reference: number;
  url: string;
}

/** Returns a user-facing problem with the report, or null when it can be sent. */
export function validateFeedback(input: { title: string; message: string }): string | null {
  const title = input.title.trim();
  const message = input.message.trim();
  if (!title) return 'Add a short title so we know what this is about.';
  if (title.length > FEEDBACK_TITLE_MAX) return `Keep the title under ${FEEDBACK_TITLE_MAX} characters.`;
  if (!message) return 'Tell us a little about it.';
  if (message.length < FEEDBACK_MESSAGE_MIN) return 'Tell us a bit more — a sentence or two helps us act on it.';
  if (message.length > FEEDBACK_MESSAGE_MAX) return `That's too long — keep it under ${FEEDBACK_MESSAGE_MAX} characters.`;
  return null;
}

export async function submitFeedback(api: AxiosInstance, input: FeedbackInput): Promise<FeedbackReceipt> {
  const r = await api.post(`${apiVersionPrefix(api)}/feedback`, {
    kind: input.kind,
    title: input.title.trim(),
    message: input.message.trim(),
    app: input.app,
    platform: input.platform,
    appVersion: input.appVersion,
  });
  return r.data.data as FeedbackReceipt;
}
