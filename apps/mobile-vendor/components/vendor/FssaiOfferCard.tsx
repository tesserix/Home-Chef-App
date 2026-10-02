// The FSSAI filing offer, wherever a chef might realise they need one.
//
// Self-hiding: it renders nothing when the service is switched off, and it
// changes into a tracker link once a request exists — so a surface can drop it
// in without knowing anything about the request's state.

import React from 'react';
import { Pressable, StyleSheet, Text, View } from 'react-native';
import { router } from 'expo-router';
import { useQuery } from '@tanstack/react-query';
import { theme } from '@homechef/mobile-shared/theme';
import { api } from '../../lib/api';
import { formatMoney } from '../../lib/format';
import { fssaiStatusLabel, isFssaiClosed } from '../../lib/fssai';
import { useFssaiQuote, useFssaiRequest } from '../../hooks/useFssai';
import { useKitchenMarket } from '../../hooks/useKitchenMarket';

/** Whether a licence is already on file. Shares the documents query key, so
 *  placing this on the Documents screen costs no extra request. */
function useHasFssaiLicence(): boolean | undefined {
  const { data } = useQuery<{ type: string }[]>({
    queryKey: ['chef', 'documents'],
    queryFn: () => api.get<{ type: string }[]>('/chef/documents').then((r) => r.data),
    staleTime: 30_000,
  });
  return data?.some((d) => d.type === 'fssai_license');
}

export function FssaiOfferCard() {
  const quoteQuery = useFssaiQuote();
  const requestQuery = useFssaiRequest();
  const hasLicence = useHasFssaiLicence();
  const market = useKitchenMarket();

  const enabled = quoteQuery.data?.enabled ?? false;
  const request = requestQuery.data?.request ?? null;
  const live = request && !isFssaiClosed(request.status) ? request : null;

  // FSSAI filing is an Indian service.
  if (market?.code !== 'IN') return null;
  // Nothing to offer and nothing to track — stay out of the way entirely.
  if (!enabled && !live) return null;
  // A chef who already holds a licence is not a candidate for the service. The
  // check lives here so every placement behaves the same and no caller has to
  // remember its own guard.
  if (hasLicence && !live) return null;

  const total = quoteQuery.data?.quote.total;

  return (
    <Pressable
      onPress={() => router.push('/fssai')}
      accessibilityRole="button"
      accessibilityLabel={live ? 'Track your FSSAI request' : 'Apply for your FSSAI registration'}
      style={styles.card}
    >
      <View style={styles.text}>
        <Text style={styles.title}>
          {live ? 'Your FSSAI request' : 'No FSSAI registration yet?'}
        </Text>
        <Text style={styles.body}>
          {live
            ? fssaiStatusLabel(live.status)
            : total
              ? `We can obtain it for you. ${formatMoney(total)} all in, including the government fee.`
              : 'We can complete the whole application for you.'}
        </Text>
      </View>
      <Text style={styles.chevron}>›</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing[3],
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.lg,
    padding: theme.spacing[4],
    ...theme.shadow[1],
  },
  text: { flex: 1, gap: theme.spacing[1] },
  title: {
    fontFamily: 'Inter-SemiBold',
    fontSize: theme.typography.size.body.size,
    color: theme.colors.ink.DEFAULT,
  },
  body: {
    fontFamily: 'Inter',
    fontSize: theme.typography.size.bodySm.size,
    lineHeight: 20,
    color: theme.colors.ink.soft,
  },
  chevron: { fontSize: 24, color: theme.colors.ink.soft },
});
